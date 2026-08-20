package trust

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"

	"github.com/dipolehq/dipole/cloud/internal/crypto"
	"github.com/dipolehq/dipole/cloud/internal/domain"
)

func TestOIDCValidator_Discovery_Attested(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	kid := "oidc-kid-1"

	jwk := jose.JSONWebKey{Key: &key.PublicKey, KeyID: kid, Algorithm: "RS256", Use: "sig"}
	keySet := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{jwk}}

	endpoint := "https://gitlab.com"
	cache := &JWKSCache{entries: map[string]*jwksCacheEntry{
		endpoint: {keys: &keySet, fetchedAt: time.Now()},
	}, logger: nil}

	claims := map[string]interface{}{
		"project_path": "myorg/myproject",
		"exp":          float64(time.Now().Add(1 * time.Hour).Unix()),
	}
	token := signJWT(t, key, kid, claims)

	provider := &domain.TrustProvider{
		Type:       "gitlab_job",
		Config:     json.RawMessage(`{"attestationMethod":"oidc_discovery","oidcEndpoint":"` + endpoint + `"}`),
		MatchRules: []domain.MatchRule{{Attribute: "project_path", Value: "myorg/myproject"}},
	}
	evidence := &domain.ClientEvidence{
		GitLab: &domain.IdentityTokenEvidence{IdentityToken: token},
	}

	v := NewOIDCValidator(nil)
	result, err := v.Validate(context.Background(), provider, evidence, cache)

	if err != nil {
		t.Fatal(err)
	}
	if !result.Attested {
		t.Errorf("expected attested=true, got reason=%s", result.Reason)
	}
}

func TestOIDCValidator_UploadJWKS_Attested(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	kid := "oidc-kid-2"

	jwk := jose.JSONWebKey{Key: &key.PublicKey, KeyID: kid, Algorithm: "RS256", Use: "sig"}
	keySet := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{jwk}}
	jwksBytes, _ := json.Marshal(keySet)

	claims := map[string]interface{}{
		"sub": "workload-123",
		"exp": float64(time.Now().Add(1 * time.Hour).Unix()),
	}
	token := signJWT(t, key, kid, claims)

	jwksStr := string(jwksBytes)
	// Escape for embedding in JSON string
	jwksEscaped, _ := json.Marshal(jwksStr)

	provider := &domain.TrustProvider{
		Type:       "oidc_id_token",
		Config:     json.RawMessage(`{"attestationMethod":"upload_jwks","jwks":` + string(jwksEscaped) + `}`),
		MatchRules: []domain.MatchRule{{Attribute: "sub", Value: "workload-123"}},
	}
	evidence := &domain.ClientEvidence{
		GitLab: &domain.IdentityTokenEvidence{IdentityToken: token},
	}

	cache := &JWKSCache{entries: map[string]*jwksCacheEntry{}, logger: nil}
	v := NewOIDCValidator(nil)
	result, err := v.Validate(context.Background(), provider, evidence, cache)

	if err != nil {
		t.Fatal(err)
	}
	if !result.Attested {
		t.Errorf("expected attested=true, got reason=%s", result.Reason)
	}
}

// encryptSymmetricKeyForTest encrypts a base64-encoded HMAC secret exactly
// as TrustProviderService does, so these tests exercise the real
// symmetric_key attestation path (TrustProvider.SymmetricKeyEncrypted must
// be a real ciphertext, not a plaintext value embedded in Config).
// testEncryptionKey is shared package-wide (kerberos_validator_test.go).
func encryptSymmetricKeyForTest(t *testing.T, secretB64 string) []byte {
	t.Helper()
	encrypted, err := crypto.Encrypt([]byte(secretB64), testEncryptionKey)
	if err != nil {
		t.Fatalf("encrypt test symmetric key: %v", err)
	}
	return encrypted
}

func TestOIDCValidator_SymmetricKey_Attested(t *testing.T) {
	secret := []byte("my-super-secret-key-for-hmac256!")
	secretB64 := base64.StdEncoding.EncodeToString(secret)

	// Build a HS256 JWT manually
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	claims := map[string]interface{}{
		"sub": "test-workload",
		"exp": float64(time.Now().Add(1 * time.Hour).Unix()),
	}
	claimsBytes, _ := json.Marshal(claims)
	payload := base64.RawURLEncoding.EncodeToString(claimsBytes)

	signingInput := header + "." + payload
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(signingInput))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	token := signingInput + "." + sig

	provider := &domain.TrustProvider{
		Type:                  "oidc_id_token",
		Config:                json.RawMessage(`{"attestationMethod":"symmetric_key"}`),
		SymmetricKeyEncrypted: encryptSymmetricKeyForTest(t, secretB64),
		MatchRules:            []domain.MatchRule{{Attribute: "sub", Value: "test-workload"}},
	}
	evidence := &domain.ClientEvidence{
		GitLab: &domain.IdentityTokenEvidence{IdentityToken: token},
	}

	cache := &JWKSCache{entries: map[string]*jwksCacheEntry{}, logger: nil}
	v := NewOIDCValidator(testEncryptionKey)
	result, err := v.Validate(context.Background(), provider, evidence, cache)

	if err != nil {
		t.Fatal(err)
	}
	if !result.Attested {
		t.Errorf("expected attested=true, got reason=%s", result.Reason)
	}
}

func TestOIDCValidator_SymmetricKey_WrongSecret(t *testing.T) {
	secret := []byte("correct-secret-key-for-signing!!")
	wrongSecret := []byte("wrong-secret-key-for-validating!")
	wrongB64 := base64.StdEncoding.EncodeToString(wrongSecret)

	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	claimsBytes, _ := json.Marshal(map[string]interface{}{"sub": "x", "exp": float64(time.Now().Add(1 * time.Hour).Unix())})
	payload := base64.RawURLEncoding.EncodeToString(claimsBytes)

	signingInput := header + "." + payload
	mac := hmac.New(sha256.New, secret) // signed with correct
	mac.Write([]byte(signingInput))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	token := signingInput + "." + sig

	provider := &domain.TrustProvider{
		Type:                  "oidc_id_token",
		Config:                json.RawMessage(`{"attestationMethod":"symmetric_key"}`),
		SymmetricKeyEncrypted: encryptSymmetricKeyForTest(t, wrongB64),
	}
	evidence := &domain.ClientEvidence{
		GitLab: &domain.IdentityTokenEvidence{IdentityToken: token},
	}

	cache := &JWKSCache{entries: map[string]*jwksCacheEntry{}, logger: nil}
	v := NewOIDCValidator(testEncryptionKey)
	result, _ := v.Validate(context.Background(), provider, evidence, cache)

	if result.Attested {
		t.Error("expected attested=false for wrong symmetric key")
	}
	if result.Reason != "InvalidSignature" {
		t.Errorf("reason = %q, want InvalidSignature", result.Reason)
	}
}

func TestOIDCValidator_SymmetricKey_NoEncryptedKeyConfigured(t *testing.T) {
	provider := &domain.TrustProvider{
		Type:   "oidc_id_token",
		Config: json.RawMessage(`{"attestationMethod":"symmetric_key"}`),
	}
	evidence := &domain.ClientEvidence{
		GitLab: &domain.IdentityTokenEvidence{IdentityToken: "a.b.c"},
	}

	cache := &JWKSCache{entries: map[string]*jwksCacheEntry{}, logger: nil}
	v := NewOIDCValidator(testEncryptionKey)
	result, err := v.Validate(context.Background(), provider, evidence, cache)

	if err != nil {
		t.Fatal(err)
	}
	if result.Attested {
		t.Error("expected attested=false when no symmetric key is configured")
	}
	if result.Reason != "InvalidSignature" {
		t.Errorf("reason = %q, want InvalidSignature", result.Reason)
	}
}

func TestOIDCValidator_NoToken_NoDataFound(t *testing.T) {
	provider := &domain.TrustProvider{Type: "gitlab_job", Config: json.RawMessage(`{}`)}
	evidence := &domain.ClientEvidence{}

	cache := &JWKSCache{entries: map[string]*jwksCacheEntry{}, logger: nil}
	v := NewOIDCValidator(nil)
	result, _ := v.Validate(context.Background(), provider, evidence, cache)

	if result.Reason != "NoDataFound" {
		t.Errorf("reason = %q, want NoDataFound", result.Reason)
	}
}

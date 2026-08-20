package trust

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

func signJWT(t *testing.T, key *rsa.PrivateKey, kid string, claims map[string]interface{}) string {
	t.Helper()

	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key}, (&jose.SignerOptions{}).WithHeader("kid", kid))
	if err != nil {
		t.Fatal(err)
	}

	payload, _ := json.Marshal(claims)
	jws, err := signer.Sign(payload)
	if err != nil {
		t.Fatal(err)
	}

	token, err := jws.CompactSerialize()
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func jwksServer(t *testing.T, key *rsa.PrivateKey, kid string) (*httptest.Server, *JWKSCache) {
	t.Helper()

	jwk := jose.JSONWebKey{Key: &key.PublicKey, KeyID: kid, Algorithm: "RS256", Use: "sig"}
	keySet := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{jwk}}

	// Create cache and pre-populate with the key set (bypasses HTTPS requirement in tests)
	cache := &JWKSCache{
		entries: map[string]*jwksCacheEntry{},
		logger:  nil,
	}

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	// Store in cache using the server URL as endpoint
	cache.entries[server.URL] = &jwksCacheEntry{keys: &keySet, fetchedAt: time.Now()}

	return server, cache
}

func TestK8sValidator_ValidToken_Attested(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	kid := "test-kid-1"
	server, cache := jwksServer(t, key, kid)
	defer server.Close()

	claims := map[string]interface{}{
		"iss": server.URL,
		"sub": "system:serviceaccount:production:myapp-sa",
		"exp": float64(time.Now().Add(1 * time.Hour).Unix()),
		"kubernetes.io": map[string]interface{}{
			"namespace":      "production",
			"serviceaccount": map[string]interface{}{"name": "myapp-sa"},
		},
	}
	token := signJWT(t, key, kid, claims)

	provider := &domain.TrustProvider{
		Config:     json.RawMessage(`{"oidcEndpoint":"` + server.URL + `"}`),
		MatchRules: []domain.MatchRule{{Attribute: "kubernetes.io.namespace", Value: "production"}},
	}
	evidence := &domain.ClientEvidence{
		K8s: &domain.K8sEvidence{ServiceAccountToken: token},
	}

	v := &K8sValidator{}
	result, err := v.Validate(context.Background(), provider, evidence, cache)

	if err != nil {
		t.Fatal(err)
	}
	if !result.Attested {
		t.Errorf("expected attested=true, got false: reason=%s attr=%s expected=%s actual=%s", result.Reason, result.Attribute, result.ExpectedValue, result.ActualValue)
	}
}

func TestK8sValidator_WrongKey_InvalidSignature(t *testing.T) {
	signingKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	jwksKey, _ := rsa.GenerateKey(rand.Reader, 2048) // different key
	kid := "test-kid-2"
	server, cache := jwksServer(t, jwksKey, kid) // JWKS has wrong key
	defer server.Close()

	claims := map[string]interface{}{"exp": float64(time.Now().Add(1 * time.Hour).Unix())}
	token := signJWT(t, signingKey, kid, claims)

	provider := &domain.TrustProvider{
		Config: json.RawMessage(`{"oidcEndpoint":"` + server.URL + `"}`),
	}
	evidence := &domain.ClientEvidence{
		K8s: &domain.K8sEvidence{ServiceAccountToken: token},
	}

	v := &K8sValidator{}
	result, _ := v.Validate(context.Background(), provider, evidence, cache)

	if result.Attested {
		t.Error("expected attested=false for wrong key")
	}
	if result.Reason != "InvalidSignature" {
		t.Errorf("reason = %q, want InvalidSignature", result.Reason)
	}
}

func TestK8sValidator_ExpiredToken(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	kid := "test-kid-3"
	server, cache := jwksServer(t, key, kid)
	defer server.Close()

	claims := map[string]interface{}{
		"iss": server.URL,
		"exp": float64(time.Now().Add(-1 * time.Hour).Unix()),
	}
	token := signJWT(t, key, kid, claims)

	provider := &domain.TrustProvider{
		Config: json.RawMessage(`{"oidcEndpoint":"` + server.URL + `"}`),
	}
	evidence := &domain.ClientEvidence{
		K8s: &domain.K8sEvidence{ServiceAccountToken: token},
	}

	v := &K8sValidator{}
	result, _ := v.Validate(context.Background(), provider, evidence, cache)

	if result.Attested {
		t.Error("expected attested=false for expired token")
	}
	if result.Reason != "TokenExpired" {
		t.Errorf("reason = %q, want TokenExpired", result.Reason)
	}
}

func TestK8sValidator_MatchRuleFailed(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	kid := "test-kid-4"
	server, cache := jwksServer(t, key, kid)
	defer server.Close()

	claims := map[string]interface{}{
		"iss": server.URL,
		"exp": float64(time.Now().Add(1 * time.Hour).Unix()),
		"kubernetes.io": map[string]interface{}{
			"namespace": "staging",
		},
	}
	token := signJWT(t, key, kid, claims)

	provider := &domain.TrustProvider{
		Config:     json.RawMessage(`{"oidcEndpoint":"` + server.URL + `"}`),
		MatchRules: []domain.MatchRule{{Attribute: "kubernetes.io.namespace", Value: "production"}},
	}
	evidence := &domain.ClientEvidence{
		K8s: &domain.K8sEvidence{ServiceAccountToken: token},
	}

	v := &K8sValidator{}
	result, _ := v.Validate(context.Background(), provider, evidence, cache)

	if result.Attested {
		t.Error("expected attested=false for wrong namespace")
	}
	if result.Reason != "MatchRuleFailed" {
		t.Errorf("reason = %q, want MatchRuleFailed", result.Reason)
	}
	if result.Attribute != "kubernetes.io.namespace" {
		t.Errorf("attribute = %q, want kubernetes.io.namespace", result.Attribute)
	}
	if result.ExpectedValue != "production" {
		t.Errorf("expected = %q, want production", result.ExpectedValue)
	}
	if result.ActualValue != "staging" {
		t.Errorf("actual = %q, want staging", result.ActualValue)
	}
}

func TestK8sValidator_NoToken_NoDataFound(t *testing.T) {
	provider := &domain.TrustProvider{Config: json.RawMessage(`{}`)}
	evidence := &domain.ClientEvidence{}

	cache := &JWKSCache{entries: map[string]*jwksCacheEntry{}, logger: nil}
	v := &K8sValidator{}
	result, _ := v.Validate(context.Background(), provider, evidence, cache)

	if result.Attested {
		t.Error("expected attested=false for missing token")
	}
	if result.Reason != "NoDataFound" {
		t.Errorf("reason = %q, want NoDataFound", result.Reason)
	}
}

func TestK8sValidator_IssuerMismatch(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	kid := "test-kid-5"
	server, cache := jwksServer(t, key, kid)
	defer server.Close()

	claims := map[string]interface{}{
		"iss": "https://attacker-controlled.example.com",
		"exp": float64(time.Now().Add(1 * time.Hour).Unix()),
	}
	token := signJWT(t, key, kid, claims)

	provider := &domain.TrustProvider{
		Config: json.RawMessage(`{"oidcEndpoint":"` + server.URL + `"}`),
	}
	evidence := &domain.ClientEvidence{
		K8s: &domain.K8sEvidence{ServiceAccountToken: token},
	}

	v := &K8sValidator{}
	result, _ := v.Validate(context.Background(), provider, evidence, cache)

	if result.Attested {
		t.Error("expected attested=false for issuer mismatch")
	}
	if result.Reason != "InvalidIssuer" {
		t.Errorf("reason = %q, want InvalidIssuer", result.Reason)
	}
}

func TestK8sValidator_MissingIssuerClaim(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	kid := "test-kid-6"
	server, cache := jwksServer(t, key, kid)
	defer server.Close()

	// No "iss" claim at all — must be rejected, not treated as a match.
	claims := map[string]interface{}{
		"exp": float64(time.Now().Add(1 * time.Hour).Unix()),
	}
	token := signJWT(t, key, kid, claims)

	provider := &domain.TrustProvider{
		Config: json.RawMessage(`{"oidcEndpoint":"` + server.URL + `"}`),
	}
	evidence := &domain.ClientEvidence{
		K8s: &domain.K8sEvidence{ServiceAccountToken: token},
	}

	v := &K8sValidator{}
	result, _ := v.Validate(context.Background(), provider, evidence, cache)

	if result.Attested {
		t.Error("expected attested=false for missing issuer claim")
	}
	if result.Reason != "InvalidIssuer" {
		t.Errorf("reason = %q, want InvalidIssuer", result.Reason)
	}
}

func TestK8sValidator_NotBeforeInFuture(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	kid := "test-kid-7"
	server, cache := jwksServer(t, key, kid)
	defer server.Close()

	claims := map[string]interface{}{
		"iss": server.URL,
		"exp": float64(time.Now().Add(2 * time.Hour).Unix()),
		"nbf": float64(time.Now().Add(1 * time.Hour).Unix()),
	}
	token := signJWT(t, key, kid, claims)

	provider := &domain.TrustProvider{
		Config: json.RawMessage(`{"oidcEndpoint":"` + server.URL + `"}`),
	}
	evidence := &domain.ClientEvidence{
		K8s: &domain.K8sEvidence{ServiceAccountToken: token},
	}

	v := &K8sValidator{}
	result, _ := v.Validate(context.Background(), provider, evidence, cache)

	if result.Attested {
		t.Error("expected attested=false for not-yet-valid token")
	}
	if result.Reason != "TokenNotYetValid" {
		t.Errorf("reason = %q, want TokenNotYetValid", result.Reason)
	}
}

func TestK8sValidator_NotBeforeInPast_Attested(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	kid := "test-kid-8"
	server, cache := jwksServer(t, key, kid)
	defer server.Close()

	claims := map[string]interface{}{
		"iss": server.URL,
		"exp": float64(time.Now().Add(1 * time.Hour).Unix()),
		"nbf": float64(time.Now().Add(-1 * time.Hour).Unix()),
	}
	token := signJWT(t, key, kid, claims)

	provider := &domain.TrustProvider{
		Config: json.RawMessage(`{"oidcEndpoint":"` + server.URL + `"}`),
	}
	evidence := &domain.ClientEvidence{
		K8s: &domain.K8sEvidence{ServiceAccountToken: token},
	}

	v := &K8sValidator{}
	result, err := v.Validate(context.Background(), provider, evidence, cache)

	if err != nil {
		t.Fatal(err)
	}
	if !result.Attested {
		t.Errorf("expected attested=true for nbf in the past, got reason=%s", result.Reason)
	}
}

func TestK8sValidator_IssuerTrailingSlashNormalized_Attested(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	kid := "test-kid-9"
	server, cache := jwksServer(t, key, kid)
	defer server.Close()

	// JWKSCache keys are the raw configured endpoint string (not
	// URL-normalized), so the cache must be populated under the exact
	// same key the validator will look up — the trailing-slash endpoint,
	// matching what a real customer would have configured.
	configuredEndpoint := server.URL + "/"
	cache.entries[configuredEndpoint] = cache.entries[server.URL]

	// Token's issuer has no trailing slash (typical of real issuers), but
	// the customer configured the endpoint WITH one — a common
	// copy-paste artifact. This must still attest.
	claims := map[string]interface{}{
		"iss": server.URL,
		"exp": float64(time.Now().Add(1 * time.Hour).Unix()),
	}
	token := signJWT(t, key, kid, claims)

	provider := &domain.TrustProvider{
		Config: json.RawMessage(`{"oidcEndpoint":"` + configuredEndpoint + `"}`),
	}
	evidence := &domain.ClientEvidence{
		K8s: &domain.K8sEvidence{ServiceAccountToken: token},
	}

	v := &K8sValidator{}
	result, err := v.Validate(context.Background(), provider, evidence, cache)

	if err != nil {
		t.Fatal(err)
	}
	if !result.Attested {
		t.Errorf("expected attested=true despite trailing slash in configured endpoint, got reason=%s", result.Reason)
	}
}

func TestK8sValidator_EmptyOIDCEndpoint_MissingIssuerClaim_NotAttested(t *testing.T) {
	// A misconfigured Trust Provider (empty oidcEndpoint) must never
	// attest just because the presented token also happens to lack an
	// iss claim — both trimming to "" must not be treated as a match.
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	kid := "test-kid-10"

	jwk := jose.JSONWebKey{Key: &key.PublicKey, KeyID: kid, Algorithm: "RS256", Use: "sig"}
	keySet := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{jwk}}
	cache := &JWKSCache{entries: map[string]*jwksCacheEntry{
		"": {keys: &keySet, fetchedAt: time.Now()},
	}, logger: nil}

	claims := map[string]interface{}{
		"exp": float64(time.Now().Add(1 * time.Hour).Unix()),
		// no "iss" claim
	}
	token := signJWT(t, key, kid, claims)

	provider := &domain.TrustProvider{
		Config: json.RawMessage(`{"oidcEndpoint":""}`),
	}
	evidence := &domain.ClientEvidence{
		K8s: &domain.K8sEvidence{ServiceAccountToken: token},
	}

	v := &K8sValidator{}
	result, _ := v.Validate(context.Background(), provider, evidence, cache)

	if result.Attested {
		t.Error("expected attested=false when configured oidcEndpoint is empty")
	}
	if result.Reason != "InvalidIssuer" {
		t.Errorf("reason = %q, want InvalidIssuer", result.Reason)
	}
}

func TestK8sValidator_NotBeforeWithinClockSkew_Attested(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	kid := "test-kid-11"
	server, cache := jwksServer(t, key, kid)
	defer server.Close()

	// nbf is 30s in the future — within the 180s clock-skew tolerance a
	// legitimate token can have if the issuer's clock runs slightly
	// ahead of this server's. Must still attest.
	claims := map[string]interface{}{
		"iss": server.URL,
		"exp": float64(time.Now().Add(1 * time.Hour).Unix()),
		"nbf": float64(time.Now().Add(30 * time.Second).Unix()),
	}
	token := signJWT(t, key, kid, claims)

	provider := &domain.TrustProvider{
		Config: json.RawMessage(`{"oidcEndpoint":"` + server.URL + `"}`),
	}
	evidence := &domain.ClientEvidence{
		K8s: &domain.K8sEvidence{ServiceAccountToken: token},
	}

	v := &K8sValidator{}
	result, err := v.Validate(context.Background(), provider, evidence, cache)

	if err != nil {
		t.Fatal(err)
	}
	if !result.Attested {
		t.Errorf("expected attested=true for nbf within clock skew tolerance, got reason=%s", result.Reason)
	}
}

func TestK8sValidator_AudienceMismatch(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	kid := "test-kid-12"
	server, cache := jwksServer(t, key, kid)
	defer server.Close()

	claims := map[string]interface{}{
		"iss": server.URL,
		"aud": "https://wrong-audience.dipolehq.com",
		"exp": float64(time.Now().Add(1 * time.Hour).Unix()),
	}
	token := signJWT(t, key, kid, claims)

	provider := &domain.TrustProvider{
		EdgeSDKAudience: "https://acme-id.dipolehq.com",
		Config:          json.RawMessage(`{"oidcEndpoint":"` + server.URL + `"}`),
	}
	evidence := &domain.ClientEvidence{
		K8s: &domain.K8sEvidence{ServiceAccountToken: token},
	}

	v := &K8sValidator{}
	result, _ := v.Validate(context.Background(), provider, evidence, cache)

	if result.Attested {
		t.Error("expected attested=false for audience mismatch")
	}
	if result.Reason != "InvalidAudience" {
		t.Errorf("reason = %q, want InvalidAudience", result.Reason)
	}
}

func TestK8sValidator_AudienceMatch_Attested(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	kid := "test-kid-13"
	server, cache := jwksServer(t, key, kid)
	defer server.Close()

	audience := "https://acme-id.dipolehq.com"
	claims := map[string]interface{}{
		"iss": server.URL,
		"aud": audience,
		"exp": float64(time.Now().Add(1 * time.Hour).Unix()),
	}
	token := signJWT(t, key, kid, claims)

	provider := &domain.TrustProvider{
		EdgeSDKAudience: audience,
		Config:          json.RawMessage(`{"oidcEndpoint":"` + server.URL + `"}`),
	}
	evidence := &domain.ClientEvidence{
		K8s: &domain.K8sEvidence{ServiceAccountToken: token},
	}

	v := &K8sValidator{}
	result, err := v.Validate(context.Background(), provider, evidence, cache)

	if err != nil {
		t.Fatal(err)
	}
	if !result.Attested {
		t.Errorf("expected attested=true for matching audience, got reason=%s", result.Reason)
	}
}

func TestK8sValidator_NoAudienceConfigured_NotEnforced_Attested(t *testing.T) {
	// When EdgeSDKAudience is not configured on the Trust Provider, the
	// audience check must be skipped entirely (opt-in, matching
	// GitHubValidator/OIDCValidator behavior) — a token with any aud
	// value, or none at all, must still attest.
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	kid := "test-kid-14"
	server, cache := jwksServer(t, key, kid)
	defer server.Close()

	claims := map[string]interface{}{
		"iss": server.URL,
		"exp": float64(time.Now().Add(1 * time.Hour).Unix()),
		// no "aud" claim, and provider.EdgeSDKAudience is unset below
	}
	token := signJWT(t, key, kid, claims)

	provider := &domain.TrustProvider{
		Config: json.RawMessage(`{"oidcEndpoint":"` + server.URL + `"}`),
	}
	evidence := &domain.ClientEvidence{
		K8s: &domain.K8sEvidence{ServiceAccountToken: token},
	}

	v := &K8sValidator{}
	result, err := v.Validate(context.Background(), provider, evidence, cache)

	if err != nil {
		t.Fatal(err)
	}
	if !result.Attested {
		t.Errorf("expected attested=true when no EdgeSDKAudience is configured, got reason=%s", result.Reason)
	}
}

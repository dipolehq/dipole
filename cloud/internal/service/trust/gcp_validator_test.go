package trust

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

func TestGCPValidator_ValidToken_DefaultEndpoint_Attested(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	kid := "gcp-kid-1"

	jwk := jose.JSONWebKey{Key: &key.PublicKey, KeyID: kid, Algorithm: "RS256", Use: "sig"}
	keySet := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{jwk}}
	cache := &JWKSCache{entries: map[string]*jwksCacheEntry{
		"https://accounts.google.com": {keys: &keySet, fetchedAt: time.Now()},
	}, logger: nil}

	claims := map[string]interface{}{
		"iss":   "https://accounts.google.com",
		"email": "workload@my-project.iam.gserviceaccount.com",
		"exp":   float64(time.Now().Add(1 * time.Hour).Unix()),
	}
	token := signJWT(t, key, kid, claims)

	provider := &domain.TrustProvider{
		Config:     json.RawMessage(`{}`),
		MatchRules: []domain.MatchRule{{Attribute: "email", Value: "workload@my-project.iam.gserviceaccount.com"}},
	}
	evidence := &domain.ClientEvidence{
		GCP: &domain.GCPEvidence{IdentityToken: token},
	}

	v := &GCPValidator{}
	result, err := v.Validate(context.Background(), provider, evidence, cache)

	if err != nil {
		t.Fatal(err)
	}
	if !result.Attested {
		t.Errorf("expected attested=true, got reason=%s", result.Reason)
	}
}

func TestGCPValidator_ValidToken_CustomEndpoint_Attested(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	kid := "gcp-kid-2"

	jwk := jose.JSONWebKey{Key: &key.PublicKey, KeyID: kid, Algorithm: "RS256", Use: "sig"}
	keySet := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{jwk}}
	endpoint := "https://custom-gcp-issuer.example.com"
	cache := &JWKSCache{entries: map[string]*jwksCacheEntry{
		endpoint: {keys: &keySet, fetchedAt: time.Now()},
	}, logger: nil}

	claims := map[string]interface{}{
		"iss": endpoint,
		"exp": float64(time.Now().Add(1 * time.Hour).Unix()),
	}
	token := signJWT(t, key, kid, claims)

	provider := &domain.TrustProvider{
		Config: json.RawMessage(`{"oidcEndpoint":"` + endpoint + `"}`),
	}
	evidence := &domain.ClientEvidence{
		GCP: &domain.GCPEvidence{IdentityToken: token},
	}

	v := &GCPValidator{}
	result, err := v.Validate(context.Background(), provider, evidence, cache)

	if err != nil {
		t.Fatal(err)
	}
	if !result.Attested {
		t.Errorf("expected attested=true, got reason=%s", result.Reason)
	}
}

func TestGCPValidator_IssuerMismatch(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	kid := "gcp-kid-3"

	jwk := jose.JSONWebKey{Key: &key.PublicKey, KeyID: kid, Algorithm: "RS256", Use: "sig"}
	keySet := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{jwk}}
	cache := &JWKSCache{entries: map[string]*jwksCacheEntry{
		"https://accounts.google.com": {keys: &keySet, fetchedAt: time.Now()},
	}, logger: nil}

	claims := map[string]interface{}{
		"iss": "https://attacker-controlled.example.com",
		"exp": float64(time.Now().Add(1 * time.Hour).Unix()),
	}
	token := signJWT(t, key, kid, claims)

	provider := &domain.TrustProvider{Config: json.RawMessage(`{}`)}
	evidence := &domain.ClientEvidence{
		GCP: &domain.GCPEvidence{IdentityToken: token},
	}

	v := &GCPValidator{}
	result, _ := v.Validate(context.Background(), provider, evidence, cache)

	if result.Attested {
		t.Error("expected attested=false for issuer mismatch")
	}
	if result.Reason != "InvalidIssuer" {
		t.Errorf("reason = %q, want InvalidIssuer", result.Reason)
	}
}

func TestGCPValidator_ExpiredToken(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	kid := "gcp-kid-4"

	jwk := jose.JSONWebKey{Key: &key.PublicKey, KeyID: kid, Algorithm: "RS256", Use: "sig"}
	keySet := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{jwk}}
	cache := &JWKSCache{entries: map[string]*jwksCacheEntry{
		"https://accounts.google.com": {keys: &keySet, fetchedAt: time.Now()},
	}, logger: nil}

	claims := map[string]interface{}{
		"iss": "https://accounts.google.com",
		"exp": float64(time.Now().Add(-1 * time.Hour).Unix()),
	}
	token := signJWT(t, key, kid, claims)

	provider := &domain.TrustProvider{Config: json.RawMessage(`{}`)}
	evidence := &domain.ClientEvidence{
		GCP: &domain.GCPEvidence{IdentityToken: token},
	}

	v := &GCPValidator{}
	result, _ := v.Validate(context.Background(), provider, evidence, cache)

	if result.Attested {
		t.Error("expected attested=false for expired token")
	}
	if result.Reason != "TokenExpired" {
		t.Errorf("reason = %q, want TokenExpired", result.Reason)
	}
}

func TestGCPValidator_NotBeforeInFuture(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	kid := "gcp-kid-5"

	jwk := jose.JSONWebKey{Key: &key.PublicKey, KeyID: kid, Algorithm: "RS256", Use: "sig"}
	keySet := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{jwk}}
	cache := &JWKSCache{entries: map[string]*jwksCacheEntry{
		"https://accounts.google.com": {keys: &keySet, fetchedAt: time.Now()},
	}, logger: nil}

	claims := map[string]interface{}{
		"iss": "https://accounts.google.com",
		"exp": float64(time.Now().Add(2 * time.Hour).Unix()),
		"nbf": float64(time.Now().Add(1 * time.Hour).Unix()),
	}
	token := signJWT(t, key, kid, claims)

	provider := &domain.TrustProvider{Config: json.RawMessage(`{}`)}
	evidence := &domain.ClientEvidence{
		GCP: &domain.GCPEvidence{IdentityToken: token},
	}

	v := &GCPValidator{}
	result, _ := v.Validate(context.Background(), provider, evidence, cache)

	if result.Attested {
		t.Error("expected attested=false for not-yet-valid token")
	}
	if result.Reason != "TokenNotYetValid" {
		t.Errorf("reason = %q, want TokenNotYetValid", result.Reason)
	}
}

func TestGCPValidator_MatchRuleFailed(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	kid := "gcp-kid-6"

	jwk := jose.JSONWebKey{Key: &key.PublicKey, KeyID: kid, Algorithm: "RS256", Use: "sig"}
	keySet := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{jwk}}
	cache := &JWKSCache{entries: map[string]*jwksCacheEntry{
		"https://accounts.google.com": {keys: &keySet, fetchedAt: time.Now()},
	}, logger: nil}

	claims := map[string]interface{}{
		"iss":   "https://accounts.google.com",
		"email": "other@my-project.iam.gserviceaccount.com",
		"exp":   float64(time.Now().Add(1 * time.Hour).Unix()),
	}
	token := signJWT(t, key, kid, claims)

	provider := &domain.TrustProvider{
		Config:     json.RawMessage(`{}`),
		MatchRules: []domain.MatchRule{{Attribute: "email", Value: "workload@my-project.iam.gserviceaccount.com"}},
	}
	evidence := &domain.ClientEvidence{
		GCP: &domain.GCPEvidence{IdentityToken: token},
	}

	v := &GCPValidator{}
	result, _ := v.Validate(context.Background(), provider, evidence, cache)

	if result.Attested {
		t.Error("expected attested=false for wrong email")
	}
	if result.Reason != "MatchRuleFailed" {
		t.Errorf("reason = %q, want MatchRuleFailed", result.Reason)
	}
	if result.ActualValue != "other@my-project.iam.gserviceaccount.com" {
		t.Errorf("actual = %q, want other@my-project.iam.gserviceaccount.com", result.ActualValue)
	}
}

func TestGCPValidator_WrongKey_InvalidSignature(t *testing.T) {
	signingKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	jwksKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	kid := "gcp-kid-7"

	jwk := jose.JSONWebKey{Key: &jwksKey.PublicKey, KeyID: kid, Algorithm: "RS256", Use: "sig"}
	keySet := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{jwk}}
	cache := &JWKSCache{entries: map[string]*jwksCacheEntry{
		"https://accounts.google.com": {keys: &keySet, fetchedAt: time.Now()},
	}, logger: nil}

	claims := map[string]interface{}{
		"iss": "https://accounts.google.com",
		"exp": float64(time.Now().Add(1 * time.Hour).Unix()),
	}
	token := signJWT(t, signingKey, kid, claims)

	provider := &domain.TrustProvider{Config: json.RawMessage(`{}`)}
	evidence := &domain.ClientEvidence{
		GCP: &domain.GCPEvidence{IdentityToken: token},
	}

	v := &GCPValidator{}
	result, _ := v.Validate(context.Background(), provider, evidence, cache)

	if result.Attested {
		t.Error("expected attested=false for wrong key")
	}
	if result.Reason != "InvalidSignature" {
		t.Errorf("reason = %q, want InvalidSignature", result.Reason)
	}
}

func TestGCPValidator_NoToken_NoDataFound(t *testing.T) {
	provider := &domain.TrustProvider{Config: json.RawMessage(`{}`)}
	evidence := &domain.ClientEvidence{}

	cache := &JWKSCache{entries: map[string]*jwksCacheEntry{}, logger: nil}
	v := &GCPValidator{}
	result, _ := v.Validate(context.Background(), provider, evidence, cache)

	if result.Attested {
		t.Error("expected attested=false for missing token")
	}
	if result.Reason != "NoDataFound" {
		t.Errorf("reason = %q, want NoDataFound", result.Reason)
	}
}

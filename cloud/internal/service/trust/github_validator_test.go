package trust

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

func TestGitHubValidator_ValidToken_Attested(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	kid := "gh-kid-1"

	jwk := jose.JSONWebKey{Key: &key.PublicKey, KeyID: kid, Algorithm: "RS256", Use: "sig"}
	keySet := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{jwk}}
	cache := &JWKSCache{entries: map[string]*jwksCacheEntry{
		githubOIDCEndpoint: {keys: &keySet, fetchedAt: time.Now()},
	}, logger: nil}

	audience := "https://acme.id.ap-south-1.dipolehq.com"
	claims := map[string]interface{}{
		"iss":        "https://token.actions.githubusercontent.com",
		"sub":        "repo:myorg/myrepo:ref:refs/heads/main",
		"aud":        audience,
		"repository": "myorg/myrepo",
		"actor":      "dev1",
		"workflow":   "deploy",
		"exp":        float64(time.Now().Add(1 * time.Hour).Unix()),
	}
	token := signJWT(t, key, kid, claims)

	provider := &domain.TrustProvider{
		ID:              uuid.New(),
		EdgeSDKAudience: audience,
		Config:          json.RawMessage(`{"match_rules":[{"field":"repository","value":"myorg/myrepo"}]}`),
	}
	evidence := &domain.ClientEvidence{
		GitHub: &domain.IdentityTokenEvidence{IdentityToken: token},
	}

	v := &GitHubValidator{}
	result, err := v.Validate(context.Background(), provider, evidence, cache)

	if err != nil {
		t.Fatal(err)
	}
	if !result.Attested {
		t.Errorf("expected attested=true, got reason=%s", result.Reason)
	}
}

func TestGitHubValidator_AudienceMismatch(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	kid := "gh-kid-2"

	jwk := jose.JSONWebKey{Key: &key.PublicKey, KeyID: kid, Algorithm: "RS256", Use: "sig"}
	keySet := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{jwk}}
	cache := &JWKSCache{entries: map[string]*jwksCacheEntry{
		githubOIDCEndpoint: {keys: &keySet, fetchedAt: time.Now()},
	}, logger: nil}

	claims := map[string]interface{}{
		"aud": "https://wrong-audience.dipolehq.com",
		"exp": float64(time.Now().Add(1 * time.Hour).Unix()),
	}
	token := signJWT(t, key, kid, claims)

	provider := &domain.TrustProvider{
		EdgeSDKAudience: "https://acme.id.ap-south-1.dipolehq.com",
		Config:          json.RawMessage(`{"match_rules":[]}`),
	}
	evidence := &domain.ClientEvidence{
		GitHub: &domain.IdentityTokenEvidence{IdentityToken: token},
	}

	v := &GitHubValidator{}
	result, _ := v.Validate(context.Background(), provider, evidence, cache)

	if result.Attested {
		t.Error("expected attested=false for audience mismatch")
	}
	if result.Reason != "InvalidAudience" {
		t.Errorf("reason = %q, want InvalidAudience", result.Reason)
	}
}

func TestGitHubValidator_WrongRepository_MatchRuleFailed(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	kid := "gh-kid-3"

	jwk := jose.JSONWebKey{Key: &key.PublicKey, KeyID: kid, Algorithm: "RS256", Use: "sig"}
	keySet := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{jwk}}
	cache := &JWKSCache{entries: map[string]*jwksCacheEntry{
		githubOIDCEndpoint: {keys: &keySet, fetchedAt: time.Now()},
	}, logger: nil}

	audience := "https://acme.id.ap-south-1.dipolehq.com"
	claims := map[string]interface{}{
		"aud":        audience,
		"repository": "other-org/other-repo",
		"exp":        float64(time.Now().Add(1 * time.Hour).Unix()),
	}
	token := signJWT(t, key, kid, claims)

	provider := &domain.TrustProvider{
		EdgeSDKAudience: audience,
		Config:          json.RawMessage(`{"match_rules":[{"field":"repository","value":"myorg/myrepo"}]}`),
	}
	evidence := &domain.ClientEvidence{
		GitHub: &domain.IdentityTokenEvidence{IdentityToken: token},
	}

	v := &GitHubValidator{}
	result, _ := v.Validate(context.Background(), provider, evidence, cache)

	if result.Attested {
		t.Error("expected attested=false for wrong repository")
	}
	if result.Reason != "MatchRuleFailed" {
		t.Errorf("reason = %q, want MatchRuleFailed", result.Reason)
	}
	if result.ActualValue != "other-org/other-repo" {
		t.Errorf("actual = %q, want other-org/other-repo", result.ActualValue)
	}
}

func TestGitHubValidator_NoToken_NoDataFound(t *testing.T) {
	cache := &JWKSCache{entries: map[string]*jwksCacheEntry{}, logger: nil}
	provider := &domain.TrustProvider{Config: json.RawMessage(`{}`)}
	evidence := &domain.ClientEvidence{}

	v := &GitHubValidator{}
	result, _ := v.Validate(context.Background(), provider, evidence, cache)

	if result.Reason != "NoDataFound" {
		t.Errorf("reason = %q, want NoDataFound", result.Reason)
	}
}

package trust

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
)

func TestVerifyJWS_ValidToken_CachedResolver(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	kid := "verify-kid-1"
	server, cache := jwksServer(t, key, kid)
	defer server.Close()

	claims := map[string]interface{}{"sub": "test-subject", "exp": float64(time.Now().Add(1 * time.Hour).Unix())}
	token := signJWT(t, key, kid, claims)

	got, verr := verifyJWS(context.Background(), token, cachedKeySetResolver{cache: cache, endpoint: server.URL})
	if verr != nil {
		t.Fatalf("unexpected error: %v", verr)
	}
	if got["sub"] != "test-subject" {
		t.Errorf("claims[sub] = %v, want test-subject", got["sub"])
	}
}

func TestVerifyJWS_ValidToken_StaticResolver(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	kid := "verify-kid-2"

	jwk := jose.JSONWebKey{Key: &key.PublicKey, KeyID: kid, Algorithm: "RS256", Use: "sig"}
	keySet := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{jwk}}
	jwksBytes, _ := json.Marshal(keySet)

	claims := map[string]interface{}{"sub": "static-subject"}
	token := signJWT(t, key, kid, claims)

	got, verr := verifyJWS(context.Background(), token, staticKeySetResolver{jwksJSON: string(jwksBytes)})
	if verr != nil {
		t.Fatalf("unexpected error: %v", verr)
	}
	if got["sub"] != "static-subject" {
		t.Errorf("claims[sub] = %v, want static-subject", got["sub"])
	}
}

func TestVerifyJWS_MalformedToken_ParseStage(t *testing.T) {
	cache := &JWKSCache{entries: map[string]*jwksCacheEntry{}, logger: nil}
	_, verr := verifyJWS(context.Background(), "not.a.jwt", cachedKeySetResolver{cache: cache, endpoint: "https://example.com"})
	if verr == nil {
		t.Fatal("expected error for malformed token")
	}
	if verr.stage != stageParse {
		t.Errorf("stage = %v, want stageParse", verr.stage)
	}
	if verr.keySetUnavailable() {
		t.Error("parse failure must not be reported as keySetUnavailable")
	}
}

func TestVerifyJWS_KeySetResolveFailure_IsKeySetUnavailable(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	kid := "verify-kid-3"
	claims := map[string]interface{}{"sub": "x"}
	token := signJWT(t, key, kid, claims)

	// Cache has no entry for this endpoint and FetchJWKS will fail
	// against a non-existent local port — exercises the resolver-failure
	// path without any network dependency succeeding by accident.
	cache := &JWKSCache{entries: map[string]*jwksCacheEntry{}, logger: nil}
	_, verr := verifyJWS(context.Background(), token, cachedKeySetResolver{cache: cache, endpoint: "https://localhost:1"})
	if verr == nil {
		t.Fatal("expected error for unreachable JWKS endpoint")
	}
	if !verr.keySetUnavailable() {
		t.Errorf("expected keySetUnavailable, got stage=%v", verr.stage)
	}
}

func TestVerifyJWS_KeyNotFoundForKid(t *testing.T) {
	signingKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	jwksKey, _ := rsa.GenerateKey(rand.Reader, 2048) // different key/kid entirely
	kid := "verify-kid-4"
	server, cache := jwksServer(t, jwksKey, "different-kid")
	defer server.Close()

	claims := map[string]interface{}{"sub": "x"}
	token := signJWT(t, signingKey, kid, claims)

	_, verr := verifyJWS(context.Background(), token, cachedKeySetResolver{cache: cache, endpoint: server.URL})
	if verr == nil {
		t.Fatal("expected error for missing kid")
	}
	if verr.stage != stageKeyNotFound {
		t.Errorf("stage = %v, want stageKeyNotFound", verr.stage)
	}
	if verr.keySetUnavailable() {
		t.Error("key-not-found must not be reported as keySetUnavailable")
	}
}

func TestVerifyJWS_WrongKey_VerifyStageFails(t *testing.T) {
	signingKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	jwksKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	kid := "verify-kid-5"
	server, cache := jwksServer(t, jwksKey, kid) // JWKS has the wrong key under the same kid
	defer server.Close()

	claims := map[string]interface{}{"sub": "x"}
	token := signJWT(t, signingKey, kid, claims)

	_, verr := verifyJWS(context.Background(), token, cachedKeySetResolver{cache: cache, endpoint: server.URL})
	if verr == nil {
		t.Fatal("expected error for signature verified against wrong key")
	}
	if verr.stage != stageVerify {
		t.Errorf("stage = %v, want stageVerify", verr.stage)
	}
}

func TestVerifyJWS_UploadedJWKS_Malformed(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	token := signJWT(t, key, "some-kid", map[string]interface{}{"sub": "x"})

	_, verr := verifyJWS(context.Background(), token, staticKeySetResolver{jwksJSON: "not json"})
	if verr == nil {
		t.Fatal("expected error for malformed uploaded JWKS")
	}
	if !verr.keySetUnavailable() {
		t.Errorf("malformed uploaded JWKS should be reported as keySetUnavailable, got stage=%v", verr.stage)
	}
}

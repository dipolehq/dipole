package identity

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"strings"
	"testing"
	"time"
)

func TestIssueToken_ES256_CorrectClaims(t *testing.T) {
	issuer, err := NewIdentityIssuer("acme", "ap-south-1")
	if err != nil {
		t.Fatal(err)
	}

	token, err := issuer.IssueToken(IssuanceRequest{
		Subject:  "workload-123",
		Audience: "https://api.example.com",
		Lifetime: 1 * time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}

	claims := decodePayload(t, token)

	if claims["iss"] != "https://acme-id.dipolehq.com" {
		t.Errorf("iss = %v, want https://acme-id.dipolehq.com", claims["iss"])
	}
	if claims["sub"] != "workload-123" {
		t.Errorf("sub = %v, want workload-123", claims["sub"])
	}
	if claims["aud"] != "https://api.example.com" {
		t.Errorf("aud = %v, want https://api.example.com", claims["aud"])
	}
	if claims["jti"] == nil || claims["jti"] == "" {
		t.Error("jti should be set")
	}
	if claims["nbf"] == nil {
		t.Error("nbf should be set")
	}

	exp := int64(claims["exp"].(float64))
	iat := int64(claims["iat"].(float64))
	if exp-iat != 3600 {
		t.Errorf("exp-iat = %d, want 3600", exp-iat)
	}
}

func TestIssueToken_RS256(t *testing.T) {
	issuer, err := NewIdentityIssuer("test", "us-east-1")
	if err != nil {
		t.Fatal(err)
	}

	token, err := issuer.IssueToken(IssuanceRequest{
		Subject:   "svc",
		Audience:  "target",
		Lifetime:  30 * time.Minute,
		Algorithm: "RS256",
	})
	if err != nil {
		t.Fatal(err)
	}

	header := decodeHeader(t, token)
	if header["alg"] != "RS256" {
		t.Errorf("alg = %v, want RS256", header["alg"])
	}
	if header["kid"] == "" {
		t.Error("kid should be set")
	}
}

func TestIssueToken_ES256_VerifySignature(t *testing.T) {
	issuer, err := NewIdentityIssuer("sig-test", "ap-south-1")
	if err != nil {
		t.Fatal(err)
	}

	token, err := issuer.IssueToken(IssuanceRequest{
		Subject:  "test-sub",
		Audience: "test-aud",
		Lifetime: 1 * time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}

	jwks := issuer.JWKS()
	keys := jwks["keys"].([]map[string]interface{})
	var ecKey map[string]interface{}
	for _, k := range keys {
		if k["alg"] == "ES256" {
			ecKey = k
			break
		}
	}

	xBytes, _ := base64.RawURLEncoding.DecodeString(ecKey["x"].(string))
	yBytes, _ := base64.RawURLEncoding.DecodeString(ecKey["y"].(string))
	pubKey := &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     new(big.Int).SetBytes(xBytes),
		Y:     new(big.Int).SetBytes(yBytes),
	}

	parts := strings.Split(token, ".")
	signingInput := parts[0] + "." + parts[1]
	sigBytes, _ := base64.RawURLEncoding.DecodeString(parts[2])

	r := new(big.Int).SetBytes(sigBytes[:32])
	s := new(big.Int).SetBytes(sigBytes[32:])

	hash := sha256.Sum256([]byte(signingInput))
	if !ecdsa.Verify(pubKey, hash[:], r, s) {
		t.Error("ES256 signature verification failed")
	}
}

func TestIssueToken_DynamicClaims(t *testing.T) {
	issuer, err := NewIdentityIssuer("dyn", "ap-south-1")
	if err != nil {
		t.Fatal(err)
	}

	token, err := issuer.IssueToken(IssuanceRequest{
		Subject:  "${oidc.identityToken.decode.payload.email}",
		Audience: "svc",
		Lifetime: 1 * time.Hour,
		CustomClaims: map[string]interface{}{
			"team": "${oidc.identityToken.decode.payload.team}",
		},
		TrustClaims: map[string]interface{}{
			"email": "user@test.com",
			"team":  "engineering",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	claims := decodePayload(t, token)
	if claims["sub"] != "user@test.com" {
		t.Errorf("sub = %v, want user@test.com", claims["sub"])
	}
	if claims["team"] != "engineering" {
		t.Errorf("team = %v, want engineering", claims["team"])
	}
}

func TestIssueToken_DynamicClaim_Missing_Error(t *testing.T) {
	issuer, err := NewIdentityIssuer("err", "ap-south-1")
	if err != nil {
		t.Fatal(err)
	}

	_, err = issuer.IssueToken(IssuanceRequest{
		Subject:     "${oidc.identityToken.decode.payload.nonexistent}",
		Audience:    "x",
		Lifetime:    1 * time.Hour,
		TrustClaims: map[string]interface{}{"email": "x"},
	})
	if err == nil {
		t.Error("expected error for missing dynamic claim")
	}
}

func TestDiscoveryDocument(t *testing.T) {
	issuer, err := NewIdentityIssuer("myco", "ap-south-1")
	if err != nil {
		t.Fatal(err)
	}

	doc := issuer.DiscoveryDocument()

	if doc["issuer"] != "https://myco-id.dipolehq.com" {
		t.Errorf("issuer = %v", doc["issuer"])
	}
	if doc["jwks_uri"] != "https://myco-id.dipolehq.com/.well-known/openid-configuration/jwks" {
		t.Errorf("jwks_uri = %v", doc["jwks_uri"])
	}
}

func TestJWKS_ContainsBothKeys(t *testing.T) {
	issuer, err := NewIdentityIssuer("jwks", "us-east-1")
	if err != nil {
		t.Fatal(err)
	}

	jwks := issuer.JWKS()
	keys := jwks["keys"].([]map[string]interface{})

	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}

	var hasEC, hasRSA bool
	for _, k := range keys {
		if k["kty"] == "EC" {
			hasEC = true
		}
		if k["kty"] == "RSA" {
			hasRSA = true
		}
	}
	if !hasEC {
		t.Error("missing EC key")
	}
	if !hasRSA {
		t.Error("missing RSA key")
	}
}

func decodePayload(t *testing.T, token string) map[string]interface{} {
	t.Helper()
	parts := strings.Split(token, ".")
	payload, _ := base64.RawURLEncoding.DecodeString(parts[1])
	var claims map[string]interface{}
	json.Unmarshal(payload, &claims)
	return claims
}

func decodeHeader(t *testing.T, token string) map[string]interface{} {
	t.Helper()
	parts := strings.Split(token, ".")
	header, _ := base64.RawURLEncoding.DecodeString(parts[0])
	var h map[string]interface{}
	json.Unmarshal(header, &h)
	return h
}

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
	issuer, err := NewIdentityIssuer("https://acme-id.dipolehq.com")
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
	issuer, err := NewIdentityIssuer("https://test-id.dipolehq.com")
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
	issuer, err := NewIdentityIssuer("https://sig-test-id.dipolehq.com")
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
	issuer, err := NewIdentityIssuer("https://dyn-id.dipolehq.com")
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
	issuer, err := NewIdentityIssuer("https://err-id.dipolehq.com")
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

// The discovery document must echo back the issuer URL it was asked for, not a
// process-wide one — one process serves every tenant, and OIDC requires the
// document's issuer to equal the URL the relying party fetched it from.
func TestDiscoveryDocument_EchoesRequestedIssuer(t *testing.T) {
	issuer, err := NewIdentityIssuer("https://platform-id.dipolehq.com")
	if err != nil {
		t.Fatal(err)
	}

	doc := issuer.DiscoveryDocument("https://myco-id.dipolehq.com")

	if doc["issuer"] != "https://myco-id.dipolehq.com" {
		t.Errorf("issuer = %v, want https://myco-id.dipolehq.com", doc["issuer"])
	}
	if doc["jwks_uri"] != "https://myco-id.dipolehq.com/.well-known/openid-configuration/jwks" {
		t.Errorf("jwks_uri = %v", doc["jwks_uri"])
	}

	// Deliberately absent: this issuer runs no interactive OAuth flow, and
	// /connect/token is only served on the MCP Authorization Server host. A
	// token_endpoint here would point relying parties at an endpoint that
	// rejects them.
	if _, present := doc["token_endpoint"]; present {
		t.Error("discovery document advertises a token_endpoint that this host does not serve")
	}
}

// Two tenants fetching discovery from their own hostnames must each see their
// own issuer, from the same running process.
func TestDiscoveryDocument_PerTenantIsolation(t *testing.T) {
	issuer, err := NewIdentityIssuer("https://platform-id.dipolehq.com")
	if err != nil {
		t.Fatal(err)
	}

	a := issuer.DiscoveryDocument("https://111111-id.dipolehq.com")
	b := issuer.DiscoveryDocument("https://222222-id.dipolehq.com")

	if a["issuer"] == b["issuer"] {
		t.Fatalf("both tenants got the same issuer %v", a["issuer"])
	}
	if a["issuer"] != "https://111111-id.dipolehq.com" {
		t.Errorf("tenant A issuer = %v", a["issuer"])
	}
	if b["issuer"] != "https://222222-id.dipolehq.com" {
		t.Errorf("tenant B issuer = %v", b["issuer"])
	}
}

func TestDiscoveryDocument_FallsBackToPlatformIssuer(t *testing.T) {
	issuer, err := NewIdentityIssuer("https://platform-id.dipolehq.com")
	if err != nil {
		t.Fatal(err)
	}

	if doc := issuer.DiscoveryDocument(""); doc["issuer"] != "https://platform-id.dipolehq.com" {
		t.Errorf("issuer = %v, want platform issuer", doc["issuer"])
	}
}

func TestJWKS_ContainsBothKeys(t *testing.T) {
	issuer, err := NewIdentityIssuer("https://jwks-id.dipolehq.com")
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

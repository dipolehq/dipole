package mcp

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/service/identity"
)

type mockTokenConfigResolver struct {
	config *TokenConfig
}

func (r *mockTokenConfigResolver) ResolveTokenConfig(_ uuid.UUID, _ uuid.UUID) (*TokenConfig, error) {
	return r.config, nil
}

func setupTokenService(t *testing.T) (*TokenService, *AuthCodeStore, string) {
	t.Helper()

	issuer, err := identity.NewIdentityIssuer("https://test-id.dipolehq.com")
	if err != nil {
		t.Fatal(err)
	}

	codeStore := NewAuthCodeStore()
	resolver := &mockTokenConfigResolver{
		config: &TokenConfig{
			Audience:         "https://mcp.example.com",
			Subject:          "user@example.com",
			Lifetime:         1 * time.Hour,
			Algorithm:        "ES256",
			RefreshEnabled:   true,
			RefreshAbsExpiry: 24 * time.Hour,
		},
	}

	svc := NewTokenService(codeStore, issuer, resolver)

	// Create a code with a known verifier
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	hash := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(hash[:])

	clientID := uuid.New()
	tenantID := uuid.New()
	policyID := uuid.New()

	code := codeStore.Create(clientID, tenantID, "http://localhost:7777/cb", challenge, "openid", nil, policyID)

	// Store context for tests
	t.Setenv("TEST_CLIENT_ID", clientID.String())
	t.Setenv("TEST_TENANT_ID", tenantID.String())

	return svc, codeStore, code
}

func TestTokenExchange_ValidPKCE_ReturnsToken(t *testing.T) {
	issuer, _ := identity.NewIdentityIssuer("https://test-id.dipolehq.com")
	codeStore := NewAuthCodeStore()
	resolver := &mockTokenConfigResolver{
		config: &TokenConfig{
			Audience:         "https://mcp.example.com",
			Subject:          "user@example.com",
			Lifetime:         1 * time.Hour,
			Algorithm:        "ES256",
			RefreshEnabled:   true,
			RefreshAbsExpiry: 24 * time.Hour,
		},
	}
	svc := NewTokenService(codeStore, issuer, resolver)

	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	hash := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(hash[:])

	clientID := uuid.New()
	tenantID := uuid.New()
	policyID := uuid.New()
	code := codeStore.Create(clientID, tenantID, "http://localhost:7777/cb", challenge, "openid", nil, policyID)

	resp, err := svc.Exchange(&TokenRequest{
		GrantType:    "authorization_code",
		Code:         code,
		RedirectURI:  "http://localhost:7777/cb",
		ClientID:     clientID.String(),
		CodeVerifier: verifier,
	}, tenantID)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.AccessToken == "" {
		t.Fatal("expected non-empty access_token")
	}
	if resp.TokenType != "Bearer" {
		t.Errorf("expected token_type=Bearer, got %s", resp.TokenType)
	}
	if resp.ExpiresIn != 3600 {
		t.Errorf("expected expires_in=3600, got %d", resp.ExpiresIn)
	}
	if resp.RefreshToken == "" {
		t.Fatal("expected non-empty refresh_token")
	}
}

func TestTokenExchange_WrongVerifier_Fails(t *testing.T) {
	issuer, _ := identity.NewIdentityIssuer("https://test-id.dipolehq.com")
	codeStore := NewAuthCodeStore()
	resolver := &mockTokenConfigResolver{
		config: &TokenConfig{Audience: "a", Subject: "s", Lifetime: time.Hour, Algorithm: "ES256"},
	}
	svc := NewTokenService(codeStore, issuer, resolver)

	verifier := "correct-verifier"
	hash := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(hash[:])

	clientID := uuid.New()
	tenantID := uuid.New()
	code := codeStore.Create(clientID, tenantID, "http://localhost/cb", challenge, "openid", nil, uuid.New())

	_, err := svc.Exchange(&TokenRequest{
		GrantType:    "authorization_code",
		Code:         code,
		RedirectURI:  "http://localhost/cb",
		ClientID:     clientID.String(),
		CodeVerifier: "wrong-verifier",
	}, tenantID)

	if err != ErrPKCEFailed {
		t.Fatalf("expected ErrPKCEFailed, got %v", err)
	}
}

func TestTokenExchange_InvalidCode_Fails(t *testing.T) {
	issuer, _ := identity.NewIdentityIssuer("https://test-id.dipolehq.com")
	codeStore := NewAuthCodeStore()
	resolver := &mockTokenConfigResolver{
		config: &TokenConfig{Audience: "a", Subject: "s", Lifetime: time.Hour, Algorithm: "ES256"},
	}
	svc := NewTokenService(codeStore, issuer, resolver)

	_, err := svc.Exchange(&TokenRequest{
		GrantType:    "authorization_code",
		Code:         "nonexistent-code",
		RedirectURI:  "http://localhost/cb",
		ClientID:     uuid.New().String(),
		CodeVerifier: "whatever",
	}, uuid.New())

	if err != ErrInvalidGrant {
		t.Fatalf("expected ErrInvalidGrant, got %v", err)
	}
}

func TestRefreshToken_SingleUse(t *testing.T) {
	issuer, _ := identity.NewIdentityIssuer("https://test-id.dipolehq.com")
	codeStore := NewAuthCodeStore()
	resolver := &mockTokenConfigResolver{
		config: &TokenConfig{
			Audience:         "https://mcp.example.com",
			Subject:          "user@example.com",
			Lifetime:         1 * time.Hour,
			Algorithm:        "ES256",
			RefreshEnabled:   true,
			RefreshAbsExpiry: 24 * time.Hour,
		},
	}
	svc := NewTokenService(codeStore, issuer, resolver)

	verifier := "test-verifier-string"
	hash := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(hash[:])

	clientID := uuid.New()
	tenantID := uuid.New()
	code := codeStore.Create(clientID, tenantID, "http://localhost/cb", challenge, "openid", nil, uuid.New())

	// Exchange code for token + refresh token
	resp1, err := svc.Exchange(&TokenRequest{
		GrantType:    "authorization_code",
		Code:         code,
		RedirectURI:  "http://localhost/cb",
		ClientID:     clientID.String(),
		CodeVerifier: verifier,
	}, tenantID)
	if err != nil {
		t.Fatalf("initial exchange failed: %v", err)
	}
	if resp1.RefreshToken == "" {
		t.Fatal("expected refresh_token")
	}

	// Use refresh token → should succeed
	resp2, err := svc.Exchange(&TokenRequest{
		GrantType:    "refresh_token",
		RefreshToken: resp1.RefreshToken,
		ClientID:     clientID.String(),
	}, tenantID)
	if err != nil {
		t.Fatalf("refresh exchange failed: %v", err)
	}
	if resp2.AccessToken == "" {
		t.Fatal("expected new access_token from refresh")
	}
	if resp2.RefreshToken == "" {
		t.Fatal("expected new refresh_token from refresh")
	}
	if resp2.RefreshToken == resp1.RefreshToken {
		t.Error("new refresh_token should be different from old one")
	}

	// Reuse old refresh token → should fail (single-use)
	_, err = svc.Exchange(&TokenRequest{
		GrantType:    "refresh_token",
		RefreshToken: resp1.RefreshToken,
		ClientID:     clientID.String(),
	}, tenantID)
	if err != ErrRefreshTokenInvalid {
		t.Fatalf("expected ErrRefreshTokenInvalid on reuse, got %v", err)
	}
}

func TestTokenExchange_UnsupportedGrantType(t *testing.T) {
	issuer, _ := identity.NewIdentityIssuer("https://test-id.dipolehq.com")
	codeStore := NewAuthCodeStore()
	resolver := &mockTokenConfigResolver{
		config: &TokenConfig{Audience: "a", Subject: "s", Lifetime: time.Hour, Algorithm: "ES256"},
	}
	svc := NewTokenService(codeStore, issuer, resolver)

	_, err := svc.Exchange(&TokenRequest{
		GrantType: "implicit",
	}, uuid.New())

	if err == nil || err.Error() != "unsupported_grant_type" {
		t.Fatalf("expected unsupported_grant_type error, got %v", err)
	}
}

// An authorization code belongs to the tenant that issued it. Redeeming it
// against another tenant's hostname must fail here, explicitly, rather than
// downstream when the code's policy happens not to be found under the redeeming
// tenant — that protection is incidental and would vanish if the policy lookup
// were ever cached or relaxed.
func TestTokenExchange_CodeFromAnotherTenant_Rejected(t *testing.T) {
	issuer, _ := identity.NewIdentityIssuer("https://test-id.dipolehq.com")
	codeStore := NewAuthCodeStore()
	resolver := &mockTokenConfigResolver{
		config: &TokenConfig{
			Audience:  "https://mcp.example.com",
			Subject:   "user@example.com",
			Lifetime:  1 * time.Hour,
			Algorithm: "ES256",
		},
	}
	svc := NewTokenService(codeStore, issuer, resolver)

	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	hash := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(hash[:])

	clientID := uuid.New()
	issuingTenant := uuid.New()
	redeemingTenant := uuid.New()
	code := codeStore.Create(clientID, issuingTenant, "http://localhost:7777/cb", challenge, "openid", nil, uuid.New())

	_, err := svc.Exchange(&TokenRequest{
		GrantType:    "authorization_code",
		Code:         code,
		RedirectURI:  "http://localhost:7777/cb",
		ClientID:     clientID.String(),
		CodeVerifier: verifier,
	}, redeemingTenant)

	if !errors.Is(err, ErrInvalidGrant) {
		t.Fatalf("expected ErrInvalidGrant for a cross-tenant code, got %v", err)
	}
}

// Same for refresh tokens: the entry records its tenant, and a refresh presented
// on another tenant's hostname must be refused.
func TestRefreshToken_FromAnotherTenant_Rejected(t *testing.T) {
	issuer, _ := identity.NewIdentityIssuer("https://test-id.dipolehq.com")
	codeStore := NewAuthCodeStore()
	resolver := &mockTokenConfigResolver{
		config: &TokenConfig{
			Audience:         "https://mcp.example.com",
			Subject:          "user@example.com",
			Lifetime:         1 * time.Hour,
			Algorithm:        "ES256",
			RefreshEnabled:   true,
			RefreshAbsExpiry: 24 * time.Hour,
		},
	}
	svc := NewTokenService(codeStore, issuer, resolver)

	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	hash := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(hash[:])

	clientID := uuid.New()
	issuingTenant := uuid.New()
	code := codeStore.Create(clientID, issuingTenant, "http://localhost/cb", challenge, "openid", nil, uuid.New())

	issued, err := svc.Exchange(&TokenRequest{
		GrantType:    "authorization_code",
		Code:         code,
		RedirectURI:  "http://localhost/cb",
		ClientID:     clientID.String(),
		CodeVerifier: verifier,
	}, issuingTenant)
	if err != nil {
		t.Fatalf("initial exchange failed: %v", err)
	}
	if issued.RefreshToken == "" {
		t.Fatal("expected a refresh token")
	}

	_, err = svc.Exchange(&TokenRequest{
		GrantType:    "refresh_token",
		RefreshToken: issued.RefreshToken,
		ClientID:     clientID.String(),
	}, uuid.New())

	if !errors.Is(err, ErrRefreshTokenInvalid) {
		t.Fatalf("expected ErrRefreshTokenInvalid for a cross-tenant refresh, got %v", err)
	}
}

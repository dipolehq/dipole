package mcp

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/service/identity"
)

type mockTokenConfigResolver struct {
	config *TokenConfig
}

func (r *mockTokenConfigResolver) ResolveTokenConfig(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*TokenConfig, error) {
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

	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	hash := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(hash[:])

	clientID := uuid.New()
	tenantID := uuid.New()
	policyID := uuid.New()

	code := codeStore.Create(clientID, tenantID, "http://localhost:7777/cb", challenge, "openid", nil, policyID)

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

	resp, err := svc.Exchange(context.Background(), &TokenRequest{
		GrantType:    "authorization_code",
		Code:         code,
		RedirectURI:  "http://localhost:7777/cb",
		ClientID:     clientID.String(),
		CodeVerifier: verifier,
	}, tenantID, "https://test-id.dipolehq.com")

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

	_, err := svc.Exchange(context.Background(), &TokenRequest{
		GrantType:    "authorization_code",
		Code:         code,
		RedirectURI:  "http://localhost/cb",
		ClientID:     clientID.String(),
		CodeVerifier: "wrong-verifier",
	}, tenantID, "https://test-id.dipolehq.com")

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

	_, err := svc.Exchange(context.Background(), &TokenRequest{
		GrantType:    "authorization_code",
		Code:         "nonexistent-code",
		RedirectURI:  "http://localhost/cb",
		ClientID:     uuid.New().String(),
		CodeVerifier: "whatever",
	}, uuid.New(), "https://test-id.dipolehq.com")

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

	resp1, err := svc.Exchange(context.Background(), &TokenRequest{
		GrantType:    "authorization_code",
		Code:         code,
		RedirectURI:  "http://localhost/cb",
		ClientID:     clientID.String(),
		CodeVerifier: verifier,
	}, tenantID, "https://test-id.dipolehq.com")
	if err != nil {
		t.Fatalf("initial exchange failed: %v", err)
	}
	if resp1.RefreshToken == "" {
		t.Fatal("expected refresh_token")
	}

	resp2, err := svc.Exchange(context.Background(), &TokenRequest{
		GrantType:    "refresh_token",
		RefreshToken: resp1.RefreshToken,
		ClientID:     clientID.String(),
	}, tenantID, "https://test-id.dipolehq.com")
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

	_, err = svc.Exchange(context.Background(), &TokenRequest{
		GrantType:    "refresh_token",
		RefreshToken: resp1.RefreshToken,
		ClientID:     clientID.String(),
	}, tenantID, "https://test-id.dipolehq.com")
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

	_, err := svc.Exchange(context.Background(), &TokenRequest{
		GrantType: "implicit",
	}, uuid.New(), "https://test-id.dipolehq.com")

	if err == nil || err.Error() != "unsupported_grant_type" {
		t.Fatalf("expected unsupported_grant_type error, got %v", err)
	}
}

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

	_, err := svc.Exchange(context.Background(), &TokenRequest{
		GrantType:    "authorization_code",
		Code:         code,
		RedirectURI:  "http://localhost:7777/cb",
		ClientID:     clientID.String(),
		CodeVerifier: verifier,
	}, redeemingTenant, "https://test-id.dipolehq.com")

	if !errors.Is(err, ErrInvalidGrant) {
		t.Fatalf("expected ErrInvalidGrant for a cross-tenant code, got %v", err)
	}
}

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

	issued, err := svc.Exchange(context.Background(), &TokenRequest{
		GrantType:    "authorization_code",
		Code:         code,
		RedirectURI:  "http://localhost/cb",
		ClientID:     clientID.String(),
		CodeVerifier: verifier,
	}, issuingTenant, "https://test-id.dipolehq.com")
	if err != nil {
		t.Fatalf("initial exchange failed: %v", err)
	}
	if issued.RefreshToken == "" {
		t.Fatal("expected a refresh token")
	}

	_, err = svc.Exchange(context.Background(), &TokenRequest{
		GrantType:    "refresh_token",
		RefreshToken: issued.RefreshToken,
		ClientID:     clientID.String(),
	}, uuid.New(), "https://test-id.dipolehq.com")

	if !errors.Is(err, ErrRefreshTokenInvalid) {
		t.Fatalf("expected ErrRefreshTokenInvalid for a cross-tenant refresh, got %v", err)
	}
}

func TestTokenExchange_IssuerIsPerTenant(t *testing.T) {
	issuer, _ := identity.NewIdentityIssuer("https://shared-platform-id.dipolehq.com")
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
	tenantID := uuid.New()
	code := codeStore.Create(clientID, tenantID, "http://localhost/cb", challenge, "openid", nil, uuid.New())

	perTenantIssuer := "https://mytenant-id.dipolehq.com"
	resp, err := svc.Exchange(context.Background(), &TokenRequest{
		GrantType:    "authorization_code",
		Code:         code,
		RedirectURI:  "http://localhost/cb",
		ClientID:     clientID.String(),
		CodeVerifier: verifier,
	}, tenantID, perTenantIssuer)
	if err != nil {
		t.Fatalf("exchange failed: %v", err)
	}

	parts := strings.Split(resp.AccessToken, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3-part JWT, got %d parts", len(parts))
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		t.Fatalf("unmarshal claims: %v", err)
	}
	if got := claims["iss"]; got != perTenantIssuer {
		t.Errorf("expected iss=%q (per-tenant), got %q", perTenantIssuer, got)
	}
}

func TestTokenExchange_EmbedsClientWorkloadClaims(t *testing.T) {
	issuer, _ := identity.NewIdentityIssuer("https://test-id.dipolehq.com")
	codeStore := NewAuthCodeStore()
	cwID := uuid.New()
	resolver := &mockTokenConfigResolver{
		config: &TokenConfig{
			Audience:           "https://mcp.example.com",
			Subject:            "user@example.com",
			Lifetime:           1 * time.Hour,
			Algorithm:          "ES256",
			ClientWorkloadID:   cwID,
			ClientWorkloadName: "Claude Desktop",
		},
	}
	svc := NewTokenService(codeStore, issuer, resolver)

	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	hash := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(hash[:])

	clientID := uuid.New()
	tenantID := uuid.New()
	code := codeStore.Create(clientID, tenantID, "http://localhost/cb", challenge, "openid", nil, uuid.New())

	resp, err := svc.Exchange(context.Background(), &TokenRequest{
		GrantType:    "authorization_code",
		Code:         code,
		RedirectURI:  "http://localhost/cb",
		ClientID:     clientID.String(),
		CodeVerifier: verifier,
	}, tenantID, "https://test-id.dipolehq.com")
	if err != nil {
		t.Fatalf("exchange failed: %v", err)
	}

	claims := decodeJWTClaims(t, resp.AccessToken)
	if got := claims["client_workload_name"]; got != "Claude Desktop" {
		t.Errorf("expected client_workload_name=%q, got %v", "Claude Desktop", got)
	}
	if got := claims["client_workload_id"]; got != cwID.String() {
		t.Errorf("expected client_workload_id=%q, got %v", cwID.String(), got)
	}
}

func TestTokenExchange_NoClientWorkloadName_ClaimOmitted(t *testing.T) {
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
	tenantID := uuid.New()
	code := codeStore.Create(clientID, tenantID, "http://localhost/cb", challenge, "openid", nil, uuid.New())

	resp, err := svc.Exchange(context.Background(), &TokenRequest{
		GrantType:    "authorization_code",
		Code:         code,
		RedirectURI:  "http://localhost/cb",
		ClientID:     clientID.String(),
		CodeVerifier: verifier,
	}, tenantID, "https://test-id.dipolehq.com")
	if err != nil {
		t.Fatalf("exchange failed: %v", err)
	}

	claims := decodeJWTClaims(t, resp.AccessToken)
	if _, present := claims["client_workload_name"]; present {
		t.Errorf("expected client_workload_name to be absent, got %v", claims["client_workload_name"])
	}
	if _, present := claims["client_workload_id"]; present {
		t.Errorf("expected client_workload_id to be absent, got %v", claims["client_workload_id"])
	}
}

func TestRefreshToken_PreservesClientWorkloadClaims(t *testing.T) {
	issuer, _ := identity.NewIdentityIssuer("https://test-id.dipolehq.com")
	codeStore := NewAuthCodeStore()
	cwID := uuid.New()
	resolver := &mockTokenConfigResolver{
		config: &TokenConfig{
			Audience:           "https://mcp.example.com",
			Subject:            "user@example.com",
			Lifetime:           1 * time.Hour,
			Algorithm:          "ES256",
			RefreshEnabled:     true,
			RefreshAbsExpiry:   24 * time.Hour,
			ClientWorkloadID:   cwID,
			ClientWorkloadName: "Gemini CLI",
		},
	}
	svc := NewTokenService(codeStore, issuer, resolver)

	verifier := "test-verifier-string"
	hash := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(hash[:])

	clientID := uuid.New()
	tenantID := uuid.New()
	code := codeStore.Create(clientID, tenantID, "http://localhost/cb", challenge, "openid", nil, uuid.New())

	resp1, err := svc.Exchange(context.Background(), &TokenRequest{
		GrantType:    "authorization_code",
		Code:         code,
		RedirectURI:  "http://localhost/cb",
		ClientID:     clientID.String(),
		CodeVerifier: verifier,
	}, tenantID, "https://test-id.dipolehq.com")
	if err != nil {
		t.Fatalf("initial exchange failed: %v", err)
	}

	resp2, err := svc.Exchange(context.Background(), &TokenRequest{
		GrantType:    "refresh_token",
		RefreshToken: resp1.RefreshToken,
		ClientID:     clientID.String(),
	}, tenantID, "https://test-id.dipolehq.com")
	if err != nil {
		t.Fatalf("refresh exchange failed: %v", err)
	}

	claims := decodeJWTClaims(t, resp2.AccessToken)
	if got := claims["client_workload_name"]; got != "Gemini CLI" {
		t.Errorf("expected client_workload_name=%q after refresh, got %v", "Gemini CLI", got)
	}
}

func decodeJWTClaims(t *testing.T, token string) map[string]any {
	t.Helper()
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3-part JWT, got %d parts", len(parts))
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		t.Fatalf("unmarshal claims: %v", err)
	}
	return claims
}

func TestTokenExchange_EmptyIssuerURL_Rejected(t *testing.T) {
	issuer, _ := identity.NewIdentityIssuer("https://test-id.dipolehq.com")
	codeStore := NewAuthCodeStore()
	resolver := &mockTokenConfigResolver{config: &TokenConfig{Audience: "a", Subject: "s", Lifetime: time.Hour, Algorithm: "ES256"}}
	svc := NewTokenService(codeStore, issuer, resolver)

	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	hash := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(hash[:])
	clientID := uuid.New()
	tenantID := uuid.New()
	code := codeStore.Create(clientID, tenantID, "http://localhost/cb", challenge, "openid", nil, uuid.New())

	_, err := svc.Exchange(context.Background(), &TokenRequest{
		GrantType:    "authorization_code",
		Code:         code,
		RedirectURI:  "http://localhost/cb",
		ClientID:     clientID.String(),
		CodeVerifier: verifier,
	}, tenantID, "")

	if err == nil {
		t.Fatal("expected error for empty issuerURL")
	}
}

func TestTokenService_SweepExpired_RemovesUsedAndExpiredRefreshTokens(t *testing.T) {
	issuer, _ := identity.NewIdentityIssuer("https://test-id.dipolehq.com")
	codeStore := NewAuthCodeStore()
	resolver := &mockTokenConfigResolver{config: &TokenConfig{Audience: "a", Subject: "s", Lifetime: time.Hour, Algorithm: "ES256"}}
	svc := NewTokenService(codeStore, issuer, resolver)

	usedToken := svc.createRefreshToken(uuid.New(), uuid.New(), nil, uuid.New(), "openid", time.Hour)
	svc.mu.Lock()
	svc.refreshTokens[usedToken].Used = true
	svc.mu.Unlock()

	expiredToken := svc.createRefreshToken(uuid.New(), uuid.New(), nil, uuid.New(), "openid", -time.Hour)
	liveToken := svc.createRefreshToken(uuid.New(), uuid.New(), nil, uuid.New(), "openid", time.Hour)

	svc.SweepExpired(time.Now().UTC())

	svc.mu.RLock()
	_, usedStillPresent := svc.refreshTokens[usedToken]
	_, expiredStillPresent := svc.refreshTokens[expiredToken]
	_, liveStillPresent := svc.refreshTokens[liveToken]
	svc.mu.RUnlock()

	if usedStillPresent {
		t.Error("expected used refresh token to be swept")
	}
	if expiredStillPresent {
		t.Error("expected expired refresh token to be swept")
	}
	if !liveStillPresent {
		t.Error("expected live refresh token to remain")
	}
}

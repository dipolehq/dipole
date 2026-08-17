package mcp_gateway

import (
	"testing"
	"time"

	"github.com/dipolehq/dipole/cloud/internal/service/identity"
)

const (
	testGatewayURL      = "https://test-mcpgateway.dipolehq.com"
	otherTenantGWURL    = "https://other-mcpgateway.dipolehq.com"
	testGatewayURLSlash = testGatewayURL + "/"
)

func setupAuth(t *testing.T) (*LeftSideAuth, *identity.IdentityIssuer) {
	t.Helper()
	issuer, err := identity.NewIdentityIssuer("https://test-id.dipolehq.com")
	if err != nil {
		t.Fatal(err)
	}
	return NewLeftSideAuth(issuer), issuer
}

func issueTestToken(t *testing.T, issuer *identity.IdentityIssuer, aud string, lifetime time.Duration, claims map[string]any) string {
	t.Helper()
	if claims == nil {
		claims = map[string]any{}
	}
	token, err := issuer.IssueToken(identity.IssuanceRequest{
		Subject:      "user@example.com",
		Audience:     aud,
		Lifetime:     lifetime,
		Algorithm:    "ES256",
		CustomClaims: claims,
	})
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func TestLeftAuth_ValidToken_ReturnsUser(t *testing.T) {
	auth, issuer := setupAuth(t)
	token := issueTestToken(t, issuer, testGatewayURL, 1*time.Hour, map[string]any{"email": "alice@corp.com"})

	result, err := auth.Authenticate(token, testGatewayURL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.UserID != "alice@corp.com" {
		t.Errorf("expected userId=alice@corp.com, got %s", result.UserID)
	}
}

func TestLeftAuth_TokenWithClientWorkloadClaim_ReturnsClientName(t *testing.T) {
	auth, issuer := setupAuth(t)
	token := issueTestToken(t, issuer, testGatewayURL, 1*time.Hour, map[string]any{
		"email":                "alice@corp.com",
		"client_workload_name": "Claude Desktop",
	})

	result, err := auth.Authenticate(token, testGatewayURL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ClientWorkloadName != "Claude Desktop" {
		t.Errorf("expected ClientWorkloadName=%q, got %q", "Claude Desktop", result.ClientWorkloadName)
	}
}

func TestLeftAuth_TokenWithoutClientWorkloadClaim_ReturnsEmptyName(t *testing.T) {
	auth, issuer := setupAuth(t)
	token := issueTestToken(t, issuer, testGatewayURL, 1*time.Hour, map[string]any{"email": "alice@corp.com"})

	result, err := auth.Authenticate(token, testGatewayURL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ClientWorkloadName != "" {
		t.Errorf("expected empty ClientWorkloadName, got %q", result.ClientWorkloadName)
	}
}

func TestLeftAuth_ValidToken_SubAsFallback(t *testing.T) {
	auth, issuer := setupAuth(t)
	token := issueTestToken(t, issuer, testGatewayURL, 1*time.Hour, nil)

	result, err := auth.Authenticate(token, testGatewayURL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.UserID != "user@example.com" {
		t.Errorf("expected userId=user@example.com (from sub), got %s", result.UserID)
	}
}

func TestLeftAuth_ExpiredToken_Returns401(t *testing.T) {
	auth, issuer := setupAuth(t)
	token := issueTestToken(t, issuer, testGatewayURL, -1*time.Hour, nil)

	_, err := auth.Authenticate(token, testGatewayURL)
	if err != ErrTokenExpired {
		t.Fatalf("expected ErrTokenExpired, got %v", err)
	}
}

func TestLeftAuth_WrongAudience_Returns401(t *testing.T) {
	auth, issuer := setupAuth(t)
	token := issueTestToken(t, issuer, "https://wrong.gateway.com/", 1*time.Hour, nil)

	_, err := auth.Authenticate(token, testGatewayURL)
	if err != ErrAudienceMismatch {
		t.Fatalf("expected ErrAudienceMismatch, got %v", err)
	}
}

func TestLeftAuth_TokenForAnotherTenantsGateway_Rejected(t *testing.T) {
	auth, issuer := setupAuth(t)
	token := issueTestToken(t, issuer, otherTenantGWURL, 1*time.Hour, map[string]any{"email": "alice@corp.com"})

	_, err := auth.Authenticate(token, testGatewayURL)
	if err != ErrAudienceMismatch {
		t.Fatalf("expected ErrAudienceMismatch for another tenant's token, got %v", err)
	}
}

func TestLeftAuth_InvalidToken_Returns401(t *testing.T) {
	auth, _ := setupAuth(t)

	_, err := auth.Authenticate("not.a.valid.jwt", testGatewayURL)
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestLeftAuth_EmptyToken_Returns401(t *testing.T) {
	auth, _ := setupAuth(t)

	_, err := auth.Authenticate("", testGatewayURL)
	if err != ErrMissingToken {
		t.Fatalf("expected ErrMissingToken, got %v", err)
	}
}

func TestLeftAuth_NoExpectedAudience_Rejected(t *testing.T) {
	auth, issuer := setupAuth(t)
	token := issueTestToken(t, issuer, testGatewayURL, 1*time.Hour, nil)

	_, err := auth.Authenticate(token, "")
	if err != ErrNoExpectedAudience {
		t.Fatalf("expected ErrNoExpectedAudience, got %v", err)
	}
}

func TestLeftAuth_TrailingSlashNormalised_BothDirections(t *testing.T) {
	cases := []struct {
		name        string
		tokenAud    string
		expectedAud string
	}{
		{"token without slash, expected with", testGatewayURL, testGatewayURLSlash},
		{"token with slash, expected without", testGatewayURLSlash, testGatewayURL},
		{"both without", testGatewayURL, testGatewayURL},
		{"both with", testGatewayURLSlash, testGatewayURLSlash},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			auth, issuer := setupAuth(t)
			token := issueTestToken(t, issuer, tc.tokenAud, 1*time.Hour, nil)

			result, err := auth.Authenticate(token, tc.expectedAud)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.UserID == "" {
				t.Fatal("expected user identity")
			}
		})
	}
}

func TestLeftAuth_AudienceArray(t *testing.T) {
	auth, issuer := setupAuth(t)

	tokenMatching := issueTestToken(t, issuer, "", 1*time.Hour, map[string]any{
		"aud": []any{otherTenantGWURL, testGatewayURL},
	})
	if _, err := auth.Authenticate(tokenMatching, testGatewayURL); err != nil {
		t.Fatalf("expected array audience containing the gateway URL to match, got %v", err)
	}

	tokenNonMatching := issueTestToken(t, issuer, "", 1*time.Hour, map[string]any{
		"aud": []any{otherTenantGWURL, "https://third-mcpgateway.dipolehq.com"},
	})
	if _, err := auth.Authenticate(tokenNonMatching, testGatewayURL); err != ErrAudienceMismatch {
		t.Fatalf("expected ErrAudienceMismatch, got %v", err)
	}
}

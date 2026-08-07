package mcp_gateway

import (
	"testing"
	"time"

	"github.com/dipolehq/dipole/cloud/internal/service/identity"
)

func setupAuth(t *testing.T) (*LeftSideAuth, *identity.IdentityIssuer) {
	t.Helper()
	issuer, err := identity.NewIdentityIssuer("https://test-id.dipolehq.com")
	if err != nil {
		t.Fatal(err)
	}
	auth := NewLeftSideAuth(issuer, "https://test-mcpgateway.dipolehq.com/")
	return auth, issuer
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
	token := issueTestToken(t, issuer, "https://test-mcpgateway.dipolehq.com/", 1*time.Hour, map[string]any{"email": "alice@corp.com"})

	result, err := auth.Authenticate(token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.UserID != "alice@corp.com" {
		t.Errorf("expected userId=alice@corp.com, got %s", result.UserID)
	}
}

func TestLeftAuth_ValidToken_SubAsFallback(t *testing.T) {
	auth, issuer := setupAuth(t)
	token := issueTestToken(t, issuer, "https://test-mcpgateway.dipolehq.com/", 1*time.Hour, nil)

	result, err := auth.Authenticate(token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.UserID != "user@example.com" {
		t.Errorf("expected userId=user@example.com (from sub), got %s", result.UserID)
	}
}

func TestLeftAuth_ExpiredToken_Returns401(t *testing.T) {
	auth, issuer := setupAuth(t)
	token := issueTestToken(t, issuer, "https://test-mcpgateway.dipolehq.com/", -1*time.Hour, nil)

	_, err := auth.Authenticate(token)
	if err != ErrTokenExpired {
		t.Fatalf("expected ErrTokenExpired, got %v", err)
	}
}

func TestLeftAuth_WrongAudience_Returns401(t *testing.T) {
	auth, issuer := setupAuth(t)
	token := issueTestToken(t, issuer, "https://wrong.gateway.com/", 1*time.Hour, nil)

	_, err := auth.Authenticate(token)
	if err != ErrAudienceMismatch {
		t.Fatalf("expected ErrAudienceMismatch, got %v", err)
	}
}

func TestLeftAuth_InvalidToken_Returns401(t *testing.T) {
	auth, _ := setupAuth(t)

	_, err := auth.Authenticate("not.a.valid.jwt")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestLeftAuth_EmptyToken_Returns401(t *testing.T) {
	auth, _ := setupAuth(t)

	_, err := auth.Authenticate("")
	if err != ErrMissingToken {
		t.Fatalf("expected ErrMissingToken, got %v", err)
	}
}

func TestLeftAuth_AudienceWithoutTrailingSlash_Matches(t *testing.T) {
	auth, issuer := setupAuth(t)
	// Gateway URL has trailing slash, token aud without — should still match
	token := issueTestToken(t, issuer, "https://test-mcpgateway.dipolehq.com", 1*time.Hour, nil)

	result, err := auth.Authenticate(token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.UserID == "" {
		t.Fatal("expected user identity")
	}
}

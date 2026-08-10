package mcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

// stubTrustValidator stands in for real signature/issuer verification, which is covered by OIDCValidator tests.
type stubTrustValidator struct {
	attested bool
	reason   string
}

func (v *stubTrustValidator) Validate(_ context.Context, _ *domain.TrustProvider, _ *domain.ClientEvidence) (*domain.TrustProviderResult, error) {
	if !v.attested {
		return &domain.TrustProviderResult{Attested: false, Reason: v.reason}, nil
	}
	return &domain.TrustProviderResult{Attested: true}, nil
}

// fakeIDToken builds a JWT-shaped string; only the payload segment is real.
func fakeIDToken(t *testing.T, claims map[string]any) string {
	t.Helper()
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal claims: %v", err)
	}
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	body := base64.RawURLEncoding.EncodeToString(payload)
	sig := base64.RawURLEncoding.EncodeToString([]byte("stub-signature"))
	return header + "." + body + "." + sig
}

// newTestCallbackService pre-seeds a session and fakes the IdP token endpoint so tests can call HandleOIDCCallback directly.
func newTestCallbackService(t *testing.T, idToken string, trust TrustValidator) (*IdPCallbackService, string, string) {
	t.Helper()

	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id_token": idToken}) //nolint:errcheck
	}))
	t.Cleanup(tokenServer.Close)

	idpSvc := NewIdPService(&mockIdPResolver{})
	codeStore := NewAuthCodeStore()
	callbackSvc := NewIdPCallbackService(idpSvc, codeStore, trust, nil)

	state := "test-state"
	nonce := "test-nonce-abc123"

	idpSvc.mu.Lock()
	idpSvc.sessions[state] = &IdPSession{
		OriginalRequest: &AuthorizeRequest{
			ClientID:      uuid.New().String(),
			RedirectURI:   "http://localhost:7777/callback",
			CodeChallenge: "challenge",
			Scope:         "openid",
			State:         "client-state",
			ServerBaseURL: "https://tenant-mcp.dipolehq.com",
		},
		TenantID:      uuid.New(),
		PolicyID:      uuid.New(),
		TrustProvider: &domain.TrustProvider{Type: domain.TrustProviderTypeOIDCIDToken, Config: []byte(`{}`)},
		IdPConfig: &ResolvedIdPConfig{
			ClientID:      "our-client",
			ClientSecret:  "secret",
			TokenEndpoint: tokenServer.URL,
		},
		ClientIP:  "1.2.3.4",
		CreatedAt: time.Now().UTC(),
		Nonce:     nonce,
	}
	idpSvc.mu.Unlock()

	return callbackSvc, state, nonce
}

func TestHandleOIDCCallback_MatchingNonce_IssuesCode(t *testing.T) {
	idToken := fakeIDToken(t, map[string]any{
		"iss":            "https://accounts.google.com",
		"aud":            "our-google-client-id",
		"sub":            "1234567890",
		"email":          "user@example.com",
		"email_verified": true,
		"nonce":          "test-nonce-abc123",
	})

	callbackSvc, state, _ := newTestCallbackService(t, idToken, &stubTrustValidator{attested: true})

	result, err := callbackSvc.HandleOIDCCallback(context.Background(), "authcode", state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result.RedirectURL, "error=") {
		t.Fatalf("expected success redirect, got error redirect: %s", result.RedirectURL)
	}
	parsed, perr := url.Parse(result.RedirectURL)
	if perr != nil {
		t.Fatalf("invalid redirect URL: %v", perr)
	}
	if parsed.Query().Get("code") == "" {
		t.Error("expected an authorization code in the redirect")
	}
}

func TestHandleOIDCCallback_MismatchedNonce_Rejected(t *testing.T) {
	idToken := fakeIDToken(t, map[string]any{
		"iss":            "https://accounts.google.com",
		"aud":            "our-google-client-id",
		"sub":            "1234567890",
		"email":          "user@example.com",
		"email_verified": true,
		"nonce":          "attacker-supplied-nonce",
	})

	callbackSvc, state, _ := newTestCallbackService(t, idToken, &stubTrustValidator{attested: true})

	result, err := callbackSvc.HandleOIDCCallback(context.Background(), "authcode", state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.RedirectURL, "error=access_denied") {
		t.Fatalf("expected access_denied redirect for nonce mismatch, got: %s", result.RedirectURL)
	}
	if !strings.Contains(result.RedirectURL, "nonce") {
		t.Errorf("expected error_description to mention nonce, got: %s", result.RedirectURL)
	}
}

func TestHandleOIDCCallback_MissingNonceClaim_Rejected(t *testing.T) {
	// Token has no "nonce" claim at all.
	idToken := fakeIDToken(t, map[string]any{
		"iss":            "https://accounts.google.com",
		"aud":            "our-google-client-id",
		"sub":            "1234567890",
		"email":          "user@example.com",
		"email_verified": true,
	})

	callbackSvc, state, _ := newTestCallbackService(t, idToken, &stubTrustValidator{attested: true})

	result, err := callbackSvc.HandleOIDCCallback(context.Background(), "authcode", state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.RedirectURL, "error=access_denied") {
		t.Fatalf("expected access_denied redirect for missing nonce claim, got: %s", result.RedirectURL)
	}
}

func TestHandleOIDCCallback_TrustValidationFailed_NonceNeverChecked(t *testing.T) {
	// Trust validation must fail before the nonce check runs.
	idToken := fakeIDToken(t, map[string]any{
		"nonce": "test-nonce-abc123",
	})

	callbackSvc, state, _ := newTestCallbackService(t, idToken, &stubTrustValidator{attested: false, reason: "InvalidSignature"})

	result, err := callbackSvc.HandleOIDCCallback(context.Background(), "authcode", state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.RedirectURL, "InvalidSignature") {
		t.Errorf("expected InvalidSignature reason in redirect, got: %s", result.RedirectURL)
	}
}

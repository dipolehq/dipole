package mcp

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type mockPolicyFinder struct {
	policy *domain.AccessPolicyWithRelations
}

func (f *mockPolicyFinder) FindMCPPolicy(_ context.Context, _ uuid.UUID, _ string, _ string, _ int) (*domain.AccessPolicyWithRelations, error) {
	return f.policy, nil
}

type mockConditionChecker struct {
	pass   bool
	reason string
}

func (c *mockConditionChecker) EvaluateAll(_ context.Context, _ []domain.AccessCondition, _ *domain.ClientEvidence) (bool, string, error) {
	return c.pass, c.reason, nil
}

type mockIdPResolver struct {
	cfg     *ResolvedIdPConfig
	samlCfg *ResolvedSAMLConfig
	err     error
}

func (r *mockIdPResolver) ResolveOIDCProvider(_ context.Context, _ uuid.UUID) (*ResolvedIdPConfig, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.cfg, nil
}

func (r *mockIdPResolver) ResolveSAMLProvider(_ context.Context, _ uuid.UUID) (*ResolvedSAMLConfig, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.samlCfg, nil
}

func TestAuthorize_SSOOff_ReturnsCode(t *testing.T) {
	repo := newMockClientRepo()
	dcrSvc := NewDCRService(repo)
	codeStore := NewAuthCodeStore()

	tenantID := uuid.New()
	resp, _ := dcrSvc.Register(context.Background(), tenantID, &DCRRequest{
		RedirectURIs:  []string{"http://localhost:7777/oauth/callback"},
		GrantTypes:    []string{"authorization_code"},
		ResponseTypes: []string{"code"},
	})

	policyID := uuid.New()
	finder := &mockPolicyFinder{
		policy: &domain.AccessPolicyWithRelations{
			Policy:         domain.AccessPolicy{ID: policyID, IsActive: true},
			ClientWorkload: domain.ClientWorkload{Identifiers: []domain.Identifier{{Type: "redirect_uri", Value: "http://localhost:7777/oauth/callback"}}},
			ServerWorkload: domain.ServerWorkload{AppProtocol: "MCP"},
		},
	}

	authSvc := NewAuthorizeService(dcrSvc, codeStore, finder, &mockConditionChecker{pass: true}, nil)

	result, err := authSvc.Authorize(context.Background(), tenantID, &AuthorizeRequest{
		ClientID:            resp.ClientID,
		RedirectURI:         "http://localhost:8080/oauth/callback",
		ResponseType:        "code",
		CodeChallenge:       "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
		CodeChallengeMethod: "S256",
		Scope:               "openid",
		State:               "mystate",
		Resource:            "https://mcp.example.com",
	}, "1.2.3.4")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RedirectURL == "" {
		t.Fatal("expected redirect URL with code")
	}
	if !strings.Contains(result.RedirectURL, "code=") {
		t.Error("redirect URL should contain code param")
	}
	if !strings.Contains(result.RedirectURL, "state=mystate") {
		t.Error("redirect URL should contain state param")
	}
}

func TestAuthorize_SSOOn_RedirectsToIdP(t *testing.T) {
	repo := newMockClientRepo()
	dcrSvc := NewDCRService(repo)
	codeStore := NewAuthCodeStore()

	tenantID := uuid.New()
	resp, _ := dcrSvc.Register(context.Background(), tenantID, &DCRRequest{
		RedirectURIs:  []string{"http://localhost:7777/oauth/callback"},
		GrantTypes:    []string{"authorization_code"},
		ResponseTypes: []string{"code"},
	})

	policyID := uuid.New()
	finder := &mockPolicyFinder{
		policy: &domain.AccessPolicyWithRelations{
			Policy:         domain.AccessPolicy{ID: policyID, IsActive: true},
			ClientWorkload: domain.ClientWorkload{Identifiers: []domain.Identifier{{Type: "redirect_uri", Value: "http://localhost:7777/oauth/callback"}}},
			ServerWorkload: domain.ServerWorkload{AppProtocol: "MCP"},
			TrustProviders: []domain.TrustProvider{{Type: domain.TrustProviderTypeOIDCIDToken, Config: []byte(`{}`)}},
		},
	}

	idpResolver := &mockIdPResolver{cfg: &ResolvedIdPConfig{
		ClientID:              "our-client",
		ClientSecret:          "secret",
		AuthorizationEndpoint: "https://idp.example.com/authorize",
		TokenEndpoint:         "https://idp.example.com/token",
		Scopes:                "openid profile email",
	}}
	idpSvc := NewIdPService(idpResolver)
	authSvc := NewAuthorizeService(dcrSvc, codeStore, finder, nil, idpSvc)

	result, err := authSvc.Authorize(context.Background(), tenantID, &AuthorizeRequest{
		ClientID:            resp.ClientID,
		RedirectURI:         "http://localhost:8080/oauth/callback",
		ResponseType:        "code",
		CodeChallenge:       "challenge",
		CodeChallengeMethod: "S256",
		Resource:            "https://mcp.example.com",
		ServerBaseURL:       "https://647251-mcp.dipolehq.com",
	}, "1.2.3.4")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IdPRedirect == "" {
		t.Fatal("expected IdP redirect URL")
	}
	if !strings.Contains(result.IdPRedirect, "https://idp.example.com/authorize") {
		t.Errorf("expected IdP authorization URL, got %s", result.IdPRedirect)
	}
	if !strings.Contains(result.IdPRedirect, "state=") {
		t.Error("IdP redirect should contain state param")
	}

	// The redirect_uri handed to the IdP must be built from the hostname this
	// request arrived on — that's what the customer registered with the IdP.
	// A mismatch surfaces as redirect_uri_mismatch.
	parsed, perr := url.Parse(result.IdPRedirect)
	if perr != nil {
		t.Fatalf("IdP redirect is not a valid URL: %v", perr)
	}
	if got, want := parsed.Query().Get("redirect_uri"), "https://647251-mcp.dipolehq.com/callback/oidc"; got != want {
		t.Errorf("redirect_uri = %q, want %q", got, want)
	}
	// Percent-encoded, not raw — providers treat raw "://" inconsistently.
	if strings.Contains(parsed.RawQuery, "redirect_uri=https://") {
		t.Error("redirect_uri is not percent-encoded in the query string")
	}
}

// Without a resolved hostname we'd send the IdP a malformed redirect_uri and
// fail opaquely at the provider, so this must fail loudly instead.
func TestAuthorize_SSOOn_MissingServerBaseURL_Fails(t *testing.T) {
	dcrSvc := NewDCRService(newMockClientRepo())
	codeStore := NewAuthCodeStore()

	tenantID := uuid.New()
	resp, _ := dcrSvc.Register(context.Background(), tenantID, &DCRRequest{
		RedirectURIs:  []string{"http://localhost:7777/oauth/callback"},
		GrantTypes:    []string{"authorization_code"},
		ResponseTypes: []string{"code"},
	})

	finder := &mockPolicyFinder{
		policy: &domain.AccessPolicyWithRelations{
			Policy:         domain.AccessPolicy{ID: uuid.New(), IsActive: true},
			ClientWorkload: domain.ClientWorkload{Identifiers: []domain.Identifier{{Type: "redirect_uri", Value: "http://localhost:7777/oauth/callback"}}},
			ServerWorkload: domain.ServerWorkload{AppProtocol: "MCP"},
			TrustProviders: []domain.TrustProvider{{Type: domain.TrustProviderTypeOIDCIDToken, Config: []byte(`{}`)}},
		},
	}
	idpResolver := &mockIdPResolver{cfg: &ResolvedIdPConfig{
		ClientID:              "our-client",
		AuthorizationEndpoint: "https://idp.example.com/authorize",
	}}

	authSvc := NewAuthorizeService(dcrSvc, codeStore, finder, nil, NewIdPService(idpResolver))

	_, err := authSvc.Authorize(context.Background(), tenantID, &AuthorizeRequest{
		ClientID:            resp.ClientID,
		RedirectURI:         "http://localhost:8080/oauth/callback",
		ResponseType:        "code",
		CodeChallenge:       "challenge",
		CodeChallengeMethod: "S256",
		Resource:            "https://mcp.example.com",
		// ServerBaseURL deliberately omitted.
	}, "1.2.3.4")

	if err == nil {
		t.Fatal("expected an error when the server base URL was not resolved")
	}
}

func TestAuthorize_NoPolicyFound_Error(t *testing.T) {
	repo := newMockClientRepo()
	dcrSvc := NewDCRService(repo)
	codeStore := NewAuthCodeStore()

	tenantID := uuid.New()
	resp, _ := dcrSvc.Register(context.Background(), tenantID, &DCRRequest{
		RedirectURIs:  []string{"http://localhost:7777/oauth/callback"},
		GrantTypes:    []string{"authorization_code"},
		ResponseTypes: []string{"code"},
	})

	finder := &mockPolicyFinder{policy: nil}
	authSvc := NewAuthorizeService(dcrSvc, codeStore, finder, nil, nil)

	_, err := authSvc.Authorize(context.Background(), tenantID, &AuthorizeRequest{
		ClientID:            resp.ClientID,
		RedirectURI:         "http://localhost:7777/oauth/callback",
		ResponseType:        "code",
		CodeChallenge:       "challenge",
		CodeChallengeMethod: "S256",
		Resource:            "https://mcp.example.com",
	}, "1.2.3.4")

	if err != ErrNoPolicyFound {
		t.Fatalf("expected ErrNoPolicyFound, got %v", err)
	}
}

func TestAuthorize_InvalidClientID_Error(t *testing.T) {
	repo := newMockClientRepo()
	dcrSvc := NewDCRService(repo)
	codeStore := NewAuthCodeStore()
	authSvc := NewAuthorizeService(dcrSvc, codeStore, nil, nil, nil)

	_, err := authSvc.Authorize(context.Background(), uuid.New(), &AuthorizeRequest{
		ClientID:            "not-a-uuid",
		RedirectURI:         "http://localhost/cb",
		CodeChallenge:       "ch",
		CodeChallengeMethod: "S256",
		Resource:            "https://mcp.example.com",
	}, "1.2.3.4")

	if err != ErrClientNotFound {
		t.Fatalf("expected ErrClientNotFound, got %v", err)
	}
}

func TestAuthorize_RedirectURIMismatch_Error(t *testing.T) {
	repo := newMockClientRepo()
	dcrSvc := NewDCRService(repo)
	codeStore := NewAuthCodeStore()

	tenantID := uuid.New()
	resp, _ := dcrSvc.Register(context.Background(), tenantID, &DCRRequest{
		RedirectURIs:  []string{"https://claude.ai/api/mcp/auth_callback"},
		GrantTypes:    []string{"authorization_code"},
		ResponseTypes: []string{"code"},
	})

	authSvc := NewAuthorizeService(dcrSvc, codeStore, nil, nil, nil)

	_, err := authSvc.Authorize(context.Background(), tenantID, &AuthorizeRequest{
		ClientID:            resp.ClientID,
		RedirectURI:         "https://evil.com/steal",
		CodeChallenge:       "ch",
		CodeChallengeMethod: "S256",
		Resource:            "https://mcp.example.com",
	}, "1.2.3.4")

	if err != ErrRedirectURIMismatch {
		t.Fatalf("expected ErrRedirectURIMismatch, got %v", err)
	}
}

func TestAuthorize_ConditionFailed_Error(t *testing.T) {
	repo := newMockClientRepo()
	dcrSvc := NewDCRService(repo)
	codeStore := NewAuthCodeStore()

	tenantID := uuid.New()
	resp, _ := dcrSvc.Register(context.Background(), tenantID, &DCRRequest{
		RedirectURIs:  []string{"http://localhost:7777/oauth/callback"},
		GrantTypes:    []string{"authorization_code"},
		ResponseTypes: []string{"code"},
	})

	finder := &mockPolicyFinder{
		policy: &domain.AccessPolicyWithRelations{
			Policy:           domain.AccessPolicy{ID: uuid.New(), IsActive: true},
			ClientWorkload:   domain.ClientWorkload{Identifiers: []domain.Identifier{{Type: "redirect_uri", Value: "http://localhost:7777/oauth/callback"}}},
			ServerWorkload:   domain.ServerWorkload{AppProtocol: "MCP"},
			AccessConditions: []domain.AccessCondition{{ID: uuid.New(), Type: "time_condition"}},
		},
	}

	authSvc := NewAuthorizeService(dcrSvc, codeStore, finder, &mockConditionChecker{pass: false, reason: "outside business hours"}, nil)

	_, err := authSvc.Authorize(context.Background(), tenantID, &AuthorizeRequest{
		ClientID:            resp.ClientID,
		RedirectURI:         "http://localhost:7777/oauth/callback",
		ResponseType:        "code",
		CodeChallenge:       "ch",
		CodeChallengeMethod: "S256",
		Resource:            "https://mcp.example.com",
	}, "1.2.3.4")

	if err == nil {
		t.Fatal("expected error for failed condition")
	}
	if !strings.Contains(err.Error(), "outside business hours") {
		t.Errorf("error should mention reason, got: %v", err)
	}
}

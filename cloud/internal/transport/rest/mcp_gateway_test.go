package rest

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/events"
	"github.com/dipolehq/dipole/cloud/internal/platform"
	gw "github.com/dipolehq/dipole/cloud/internal/service/mcp_gateway"
)

// recordingGatewayAuth captures the expected audience the transport derived from
// the request hostname.
type recordingGatewayAuth struct {
	called       bool
	gotAudience  string
	gotToken     string
	resultUserID string
	errToReturn  error
}

func (a *recordingGatewayAuth) Authenticate(token string, expectedAudience string) (*gw.AuthResult, error) {
	a.called = true
	a.gotToken = token
	a.gotAudience = expectedAudience
	if a.errToReturn != nil {
		return nil, a.errToReturn
	}
	return &gw.AuthResult{UserID: a.resultUserID, Claims: map[string]any{}}, nil
}

type recordingMethodHandler struct {
	called  bool
	gotCtx  *gw.RequestContext
	gotName string
}

func (h *recordingMethodHandler) HandleMethod(_ context.Context, rCtx *gw.RequestContext, method string, _ json.RawMessage) (any, error) {
	h.called = true
	h.gotCtx = rCtx
	h.gotName = method
	return map[string]any{}, nil
}

type recordingConsentHandler struct {
	called   bool
	gotState string
	gotScope gw.TenantScope
	err      error
}

func (h *recordingConsentHandler) HandleCallback(_ context.Context, _ string, state string, scope gw.TenantScope) error {
	h.called = true
	h.gotState = state
	h.gotScope = scope
	return h.err
}

const (
	gwTenantASubdomain = "647251"
	gwTenantBSubdomain = "889900"
)

func newGatewayTestServer(t *testing.T) (*MCPGatewayServer, *recordingGatewayAuth, *recordingMethodHandler, map[string]*domain.Tenant) {
	t.Helper()

	tenants := map[string]*domain.Tenant{
		gwTenantASubdomain: {ID: uuid.New(), Name: "Tenant A", Subdomain: gwTenantASubdomain},
		gwTenantBSubdomain: {ID: uuid.New(), Name: "Tenant B", Subdomain: gwTenantBSubdomain},
	}
	lookup := &fakeTenantLookup{bySubdomain: tenants}
	resolver := NewTenantHostResolver(lookup, platform.NewHosts("dipolehq.com", ""), false)

	auth := &recordingGatewayAuth{resultUserID: "alice@corp.com"}
	handler := &recordingMethodHandler{}

	srv := NewMCPGatewayServer(auth, handler, nil, nil, nil, resolver)
	return srv, auth, handler, tenants
}

func gatewayRequest(forwardedHost string, token string) *http.Request {
	body := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"ping"}`)
	req := httptest.NewRequest(http.MethodPost, "/mcp", body)
	req.Host = "internal.run.app"
	req.Header.Set("X-Forwarded-Host", forwardedHost)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req
}

func TestGatewayHandleMCP_ResolvesTenantFromHostname(t *testing.T) {
	srv, auth, handler, tenants := newGatewayTestServer(t)

	rec := httptest.NewRecorder()
	srv.HandleMCP(rec, gatewayRequest(gwTenantASubdomain+"-mcpgateway.dipolehq.com", "tok"))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}

	wantAudience := "https://" + gwTenantASubdomain + "-mcpgateway.dipolehq.com"
	if auth.gotAudience != wantAudience {
		t.Errorf("expected audience %q, got %q", wantAudience, auth.gotAudience)
	}
	if !handler.called {
		t.Fatal("expected method handler to be invoked")
	}
	if got := handler.gotCtx.Scope.TenantID; got != tenants[gwTenantASubdomain].ID {
		t.Errorf("expected scope tenant %s, got %s", tenants[gwTenantASubdomain].ID, got)
	}
	if got := handler.gotCtx.Scope.GatewayURL; got != wantAudience {
		t.Errorf("expected scope gateway URL %q, got %q", wantAudience, got)
	}
	if !handler.gotCtx.Scope.Valid() {
		t.Error("expected a valid tenant scope")
	}
}

func TestGatewayHandleMCP_ServesEveryTenantWithItsOwnScope(t *testing.T) {
	srv, auth, handler, tenants := newGatewayTestServer(t)

	for _, sub := range []string{gwTenantASubdomain, gwTenantBSubdomain} {
		rec := httptest.NewRecorder()
		srv.HandleMCP(rec, gatewayRequest(sub+"-mcpgateway.dipolehq.com", "tok"))

		if rec.Code != http.StatusOK {
			t.Fatalf("subdomain %s: expected 200, got %d", sub, rec.Code)
		}
		if got := handler.gotCtx.Scope.TenantID; got != tenants[sub].ID {
			t.Errorf("subdomain %s: expected tenant %s, got %s", sub, tenants[sub].ID, got)
		}
		wantAudience := "https://" + sub + "-mcpgateway.dipolehq.com"
		if auth.gotAudience != wantAudience {
			t.Errorf("subdomain %s: expected audience %q, got %q", sub, wantAudience, auth.gotAudience)
		}
	}
}

func TestGatewayHandleMCP_RejectsNonGatewayServiceHost(t *testing.T) {
	srv, auth, handler, _ := newGatewayTestServer(t)

	for _, host := range []string{
		gwTenantASubdomain + "-mcp.dipolehq.com",
		gwTenantASubdomain + "-id.dipolehq.com",
	} {
		rec := httptest.NewRecorder()
		req := gatewayRequest(host, "tok")
		srv.HandleMCP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("host %s: expected 404, got %d", host, rec.Code)
		}
		if _, err := srv.resolveScope(req); !errors.Is(err, ErrNotGatewayHost) {
			t.Errorf("host %s: expected ErrNotGatewayHost, got %v", host, err)
		}
	}
	if auth.called {
		t.Error("authenticator must not run for a non-Gateway hostname")
	}
	if handler.called {
		t.Error("method handler must not run for a non-Gateway hostname")
	}
}

func TestGatewayHandleMCP_UnresolvableHostsFailClosed(t *testing.T) {
	cases := map[string]string{
		"unknown tenant":  "999999-mcpgateway.dipolehq.com",
		"non-tenant host": "api.dipolehq.com",
		"foreign domain":  "647251-mcpgateway.evil.com",
		"cloud run host":  "internal.run.app",
		"no subdomain":    "-mcpgateway.dipolehq.com",
	}

	for name, host := range cases {
		t.Run(name, func(t *testing.T) {
			srv, auth, handler, _ := newGatewayTestServer(t)

			rec := httptest.NewRecorder()
			srv.HandleMCP(rec, gatewayRequest(host, "tok"))

			if rec.Code != http.StatusNotFound {
				t.Errorf("expected 404, got %d", rec.Code)
			}
			if auth.called || handler.called {
				t.Error("nothing downstream may run for an unattributable request")
			}
		})
	}
}

func TestGatewayHandleMCP_MissingTokenStillChallenges(t *testing.T) {
	srv, auth, _, _ := newGatewayTestServer(t)

	rec := httptest.NewRecorder()
	srv.HandleMCP(rec, gatewayRequest(gwTenantASubdomain+"-mcpgateway.dipolehq.com", ""))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	wwwAuth := rec.Header().Get("WWW-Authenticate")
	if wwwAuth == "" {
		t.Fatal("expected a WWW-Authenticate challenge")
	}
	wantMetadata := `resource_metadata="https://` + gwTenantASubdomain + `-mcpgateway.dipolehq.com/.well-known/oauth-protected-resource"`
	if !strings.Contains(wwwAuth, wantMetadata) {
		t.Errorf("WWW-Authenticate missing resource_metadata:\n  got:  %s\n  want substring: %s", wwwAuth, wantMetadata)
	}
	if auth.called {
		t.Error("authenticator should not be called without a token")
	}
}

func TestGatewayHandleMCP_InvalidTokenReturns401(t *testing.T) {
	srv, auth, handler, _ := newGatewayTestServer(t)
	auth.errToReturn = gw.ErrAudienceMismatch

	rec := httptest.NewRecorder()
	srv.HandleMCP(rec, gatewayRequest(gwTenantASubdomain+"-mcpgateway.dipolehq.com", "tok"))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if handler.called {
		t.Error("method handler must not run when authentication fails")
	}
}

func TestGatewayHandleMCP_TracingEntriesAreTenantScoped(t *testing.T) {
	tenants := map[string]*domain.Tenant{
		gwTenantASubdomain: {ID: uuid.New(), Name: "Tenant A", Subdomain: gwTenantASubdomain},
		gwTenantBSubdomain: {ID: uuid.New(), Name: "Tenant B", Subdomain: gwTenantBSubdomain},
	}
	lookup := &fakeTenantLookup{bySubdomain: tenants}
	resolver := NewTenantHostResolver(lookup, platform.NewHosts("dipolehq.com", ""), false)
	bus := events.NewEventBus(nil, nil)

	srv := NewMCPGatewayServer(
		&recordingGatewayAuth{resultUserID: "alice@corp.com"},
		&recordingMethodHandler{}, nil, nil, bus, resolver,
	)

	srv.HandleMCP(httptest.NewRecorder(), gatewayRequest(gwTenantASubdomain+"-mcpgateway.dipolehq.com", "tok"))
	srv.HandleMCP(httptest.NewRecorder(), gatewayRequest(gwTenantBSubdomain+"-mcpgateway.dipolehq.com", "tok"))
	srv.HandleMCP(httptest.NewRecorder(), gatewayRequest("api.dipolehq.com", "tok"))

	aEntries := bus.RecentTraces(tenants[gwTenantASubdomain].ID)
	bEntries := bus.RecentTraces(tenants[gwTenantBSubdomain].ID)

	if len(aEntries) != 1 {
		t.Fatalf("expected 1 entry for tenant A, got %d", len(aEntries))
	}
	if len(bEntries) != 1 {
		t.Fatalf("expected 1 entry for tenant B, got %d", len(bEntries))
	}
	if aEntries[0].TenantID != tenants[gwTenantASubdomain].ID {
		t.Error("tenant A's entry is attributed to the wrong tenant")
	}
	if len(bus.RecentTraces(uuid.New())) != 0 {
		t.Error("an unrelated tenant must see no entries")
	}
}

func TestGatewayConsentCallback_ResolvesTenantFromHostname(t *testing.T) {
	tenants := map[string]*domain.Tenant{
		gwTenantASubdomain: {ID: uuid.New(), Name: "Tenant A", Subdomain: gwTenantASubdomain},
	}
	lookup := &fakeTenantLookup{bySubdomain: tenants}
	resolver := NewTenantHostResolver(lookup, platform.NewHosts("dipolehq.com", ""), false)
	consent := &recordingConsentHandler{}

	srv := NewMCPGatewayServer(&recordingGatewayAuth{}, &recordingMethodHandler{}, consent, nil, nil, resolver)

	req := httptest.NewRequest(http.MethodGet, "/mcp/consent/callback?code=abc&state=xyz", nil)
	req.Host = "internal.run.app"
	req.Header.Set("X-Forwarded-Host", gwTenantASubdomain+"-mcpgateway.dipolehq.com")

	rec := httptest.NewRecorder()
	srv.HandleConsentCallback(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	if !consent.called {
		t.Fatal("expected consent handler to be invoked")
	}
	if consent.gotScope.TenantID != tenants[gwTenantASubdomain].ID {
		t.Errorf("expected scope tenant %s, got %s", tenants[gwTenantASubdomain].ID, consent.gotScope.TenantID)
	}
	if want := "https://" + gwTenantASubdomain + "-mcpgateway.dipolehq.com"; consent.gotScope.GatewayURL != want {
		t.Errorf("expected gateway URL %q, got %q", want, consent.gotScope.GatewayURL)
	}
}

func TestGatewayConsentCallback_UnresolvableHostFailsClosed(t *testing.T) {
	tenants := map[string]*domain.Tenant{
		gwTenantASubdomain: {ID: uuid.New(), Name: "Tenant A", Subdomain: gwTenantASubdomain},
	}
	lookup := &fakeTenantLookup{bySubdomain: tenants}
	resolver := NewTenantHostResolver(lookup, platform.NewHosts("dipolehq.com", ""), false)
	consent := &recordingConsentHandler{}

	srv := NewMCPGatewayServer(&recordingGatewayAuth{}, &recordingMethodHandler{}, consent, nil, nil, resolver)

	req := httptest.NewRequest(http.MethodGet, "/mcp/consent/callback?code=abc&state=xyz", nil)
	req.Host = "internal.run.app"
	req.Header.Set("X-Forwarded-Host", "api.dipolehq.com")

	rec := httptest.NewRecorder()
	srv.HandleConsentCallback(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	if consent.called {
		t.Error("consent handler must not run for an unattributable request")
	}
}

func TestGatewayProtectedResourceMetadata_ReturnsCorrectAuthServer(t *testing.T) {
	tenants := map[string]*domain.Tenant{
		gwTenantASubdomain: {ID: uuid.New(), Name: "Tenant A", Subdomain: gwTenantASubdomain},
	}
	lookup := &fakeTenantLookup{bySubdomain: tenants}
	resolver := NewTenantHostResolver(lookup, platform.NewHosts("dipolehq.com", ""), false)

	srv := NewMCPGatewayServer(&recordingGatewayAuth{}, &recordingMethodHandler{}, nil, nil, nil, resolver)

	req := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-protected-resource", nil)
	req.Host = "internal.run.app"
	req.Header.Set("X-Forwarded-Host", gwTenantASubdomain+"-mcpgateway.dipolehq.com")

	rec := httptest.NewRecorder()
	srv.HandleProtectedResourceMetadata(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}

	var meta map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &meta); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	wantResource := "https://" + gwTenantASubdomain + "-mcpgateway.dipolehq.com/mcp"
	if got := meta["resource"]; got != wantResource {
		t.Errorf("expected resource %q, got %v", wantResource, got)
	}

	servers, ok := meta["authorization_servers"].([]interface{})
	if !ok || len(servers) != 1 {
		t.Fatalf("expected exactly 1 authorization_server, got %v", meta["authorization_servers"])
	}
	wantAS := "https://" + gwTenantASubdomain + "-mcp.dipolehq.com"
	if got := servers[0]; got != wantAS {
		t.Errorf("expected authorization_server %q, got %v", wantAS, got)
	}
}

func TestGatewayProtectedResourceMetadata_RejectsNonGatewayHost(t *testing.T) {
	tenants := map[string]*domain.Tenant{
		gwTenantASubdomain: {ID: uuid.New(), Name: "Tenant A", Subdomain: gwTenantASubdomain},
	}
	lookup := &fakeTenantLookup{bySubdomain: tenants}
	resolver := NewTenantHostResolver(lookup, platform.NewHosts("dipolehq.com", ""), false)

	srv := NewMCPGatewayServer(&recordingGatewayAuth{}, &recordingMethodHandler{}, nil, nil, nil, resolver)

	req := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-protected-resource", nil)
	req.Host = "internal.run.app"
	req.Header.Set("X-Forwarded-Host", gwTenantASubdomain+"-mcp.dipolehq.com")

	rec := httptest.NewRecorder()
	srv.HandleProtectedResourceMetadata(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for non-gateway host, got %d", rec.Code)
	}
}

func TestGatewayProtectedResourceMetadata_UnknownTenantFailsClosed(t *testing.T) {
	tenants := map[string]*domain.Tenant{
		gwTenantASubdomain: {ID: uuid.New(), Name: "Tenant A", Subdomain: gwTenantASubdomain},
	}
	lookup := &fakeTenantLookup{bySubdomain: tenants}
	resolver := NewTenantHostResolver(lookup, platform.NewHosts("dipolehq.com", ""), false)

	srv := NewMCPGatewayServer(&recordingGatewayAuth{}, &recordingMethodHandler{}, nil, nil, nil, resolver)

	req := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-protected-resource", nil)
	req.Host = "internal.run.app"
	req.Header.Set("X-Forwarded-Host", "999999-mcpgateway.dipolehq.com")

	rec := httptest.NewRecorder()
	srv.HandleProtectedResourceMetadata(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for unknown tenant, got %d", rec.Code)
	}
}

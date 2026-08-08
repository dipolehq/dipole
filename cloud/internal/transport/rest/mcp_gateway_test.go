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
	"github.com/dipolehq/dipole/cloud/internal/platform"
	gw "github.com/dipolehq/dipole/cloud/internal/service/mcp_gateway"
)

// recordingGatewayAuth captures the expected audience the transport derived from
// the request hostname, which is what binds a token to one tenant.
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

// recordingMethodHandler captures the RequestContext so tests can assert the
// tenant scope handed to the business layer.
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

const (
	gwBoundSubdomain = "647251"
	gwOtherSubdomain = "889900"
)

// newGatewayTestServer builds a Gateway wired to a resolver that knows two
// tenants, so cross-tenant behaviour can be asserted.
func newGatewayTestServer(t *testing.T, boundTenantID uuid.UUID) (*MCPGatewayServer, *recordingGatewayAuth, *recordingMethodHandler, map[string]*domain.Tenant) {
	t.Helper()

	tenants := map[string]*domain.Tenant{
		gwBoundSubdomain: {ID: uuid.New(), Name: "Bound", Subdomain: gwBoundSubdomain},
		gwOtherSubdomain: {ID: uuid.New(), Name: "Other", Subdomain: gwOtherSubdomain},
	}
	lookup := &fakeTenantLookup{bySubdomain: tenants}
	resolver := NewTenantHostResolver(lookup, platform.NewHosts("dipolehq.com"), false)

	auth := &recordingGatewayAuth{resultUserID: "alice@corp.com"}
	handler := &recordingMethodHandler{}

	srv := NewMCPGatewayServer(auth, handler, nil, nil, nil, resolver, boundTenantID)
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
	srv, auth, handler, tenants := newGatewayTestServer(t, uuid.Nil)

	rec := httptest.NewRecorder()
	srv.HandleMCP(rec, gatewayRequest(gwBoundSubdomain+"-mcpgateway.dipolehq.com", "tok"))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}

	wantAudience := "https://" + gwBoundSubdomain + "-mcpgateway.dipolehq.com"
	if auth.gotAudience != wantAudience {
		t.Errorf("expected audience %q, got %q", wantAudience, auth.gotAudience)
	}
	if !handler.called {
		t.Fatal("expected method handler to be invoked")
	}
	if got := handler.gotCtx.Scope.TenantID; got != tenants[gwBoundSubdomain].ID {
		t.Errorf("expected scope tenant %s, got %s", tenants[gwBoundSubdomain].ID, got)
	}
	if got := handler.gotCtx.Scope.GatewayURL; got != wantAudience {
		t.Errorf("expected scope gateway URL %q, got %q", wantAudience, got)
	}
	if !handler.gotCtx.Scope.Valid() {
		t.Error("expected a valid tenant scope")
	}
}

// Each tenant's hostname must produce that tenant's scope — the whole point of
// per-request resolution.
func TestGatewayHandleMCP_DifferentHostnamesProduceDifferentTenants(t *testing.T) {
	srv, _, handler, tenants := newGatewayTestServer(t, uuid.Nil)

	for _, sub := range []string{gwBoundSubdomain, gwOtherSubdomain} {
		rec := httptest.NewRecorder()
		srv.HandleMCP(rec, gatewayRequest(sub+"-mcpgateway.dipolehq.com", "tok"))

		if rec.Code != http.StatusOK {
			t.Fatalf("subdomain %s: expected 200, got %d", sub, rec.Code)
		}
		if got := handler.gotCtx.Scope.TenantID; got != tenants[sub].ID {
			t.Errorf("subdomain %s: expected tenant %s, got %s", sub, tenants[sub].ID, got)
		}
	}
}

// All routes share one mux, so POST /mcp is reachable on the MCP Authorization
// Server hostname too. Without a service check the Auth Server's base URL would
// be passed to left-side auth as the expected token audience.
func TestGatewayHandleMCP_RejectsNonGatewayServiceHost(t *testing.T) {
	srv, auth, handler, _ := newGatewayTestServer(t, uuid.Nil)

	for _, host := range []string{
		gwBoundSubdomain + "-mcp.dipolehq.com",
		gwBoundSubdomain + "-id.dipolehq.com",
	} {
		rec := httptest.NewRecorder()
		req := gatewayRequest(host, "tok")
		srv.HandleMCP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("host %s: expected 404, got %d", host, rec.Code)
		}
		// The tenant resolved fine here — only the service was wrong. The error
		// must say so, so a 404 while debugging an MCP client doesn't look like
		// broken DNS or a missing tenant.
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
			srv, auth, handler, _ := newGatewayTestServer(t, uuid.Nil)

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

// While the client map, event sink, and consent URL are still per-startup
// (Fix 11 steps 5-7), the Gateway serves exactly the tenant it was provisioned
// for and refuses every other, mirroring Aembit's per-tenant provisioning.
func TestGatewayHandleMCP_BoundTenantRefusesOtherTenants(t *testing.T) {
	srv, auth, handler, tenants := newGatewayTestServer(t, uuid.Nil)
	boundID := tenants[gwBoundSubdomain].ID
	srv.boundTenantID = boundID

	rec := httptest.NewRecorder()
	srv.HandleMCP(rec, gatewayRequest(gwOtherSubdomain+"-mcpgateway.dipolehq.com", "tok"))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for a non-provisioned tenant, got %d", rec.Code)
	}
	if auth.called || handler.called {
		t.Error("nothing downstream may run for a non-provisioned tenant")
	}

	// The provisioned tenant still works.
	rec = httptest.NewRecorder()
	srv.HandleMCP(rec, gatewayRequest(gwBoundSubdomain+"-mcpgateway.dipolehq.com", "tok"))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for the provisioned tenant, got %d (%s)", rec.Code, rec.Body.String())
	}
	if handler.gotCtx.Scope.TenantID != boundID {
		t.Errorf("expected scope tenant %s, got %s", boundID, handler.gotCtx.Scope.TenantID)
	}
}

// A missing token on a valid Gateway hostname must still produce the 401 +
// WWW-Authenticate that makes an MCP client begin OAuth discovery. Resolving the
// tenant first must not turn that into a 404.
func TestGatewayHandleMCP_MissingTokenStillChallenges(t *testing.T) {
	srv, auth, _, _ := newGatewayTestServer(t, uuid.Nil)

	rec := httptest.NewRecorder()
	srv.HandleMCP(rec, gatewayRequest(gwBoundSubdomain+"-mcpgateway.dipolehq.com", ""))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if got := rec.Header().Get("WWW-Authenticate"); got == "" {
		t.Error("expected a WWW-Authenticate challenge")
	}
	if auth.called {
		t.Error("authenticator should not be called without a token")
	}
}

func TestGatewayHandleMCP_InvalidTokenReturns401(t *testing.T) {
	srv, auth, handler, _ := newGatewayTestServer(t, uuid.Nil)
	auth.errToReturn = gw.ErrAudienceMismatch

	rec := httptest.NewRecorder()
	srv.HandleMCP(rec, gatewayRequest(gwBoundSubdomain+"-mcpgateway.dipolehq.com", "tok"))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if handler.called {
		t.Error("method handler must not run when authentication fails")
	}
}

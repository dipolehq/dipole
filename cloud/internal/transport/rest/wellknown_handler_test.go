package rest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/platform"
	"github.com/dipolehq/dipole/cloud/internal/service/identity"
)

func newWellKnownTestServer(t *testing.T, tenants map[string]*domain.Tenant) *http.ServeMux {
	t.Helper()

	issuer, err := identity.NewIdentityIssuer("https://id.dipolehq.com")
	if err != nil {
		t.Fatal(err)
	}

	lookup := &fakeTenantLookup{bySubdomain: tenants}
	resolver := NewTenantHostResolver(lookup, platform.NewHosts("dipolehq.com", ""), false)

	mux := http.NewServeMux()
	NewWellKnownHandler(issuer, resolver, nil).RegisterRoutes(mux)
	return mux
}

func getJSON(t *testing.T, mux *http.ServeMux, host, path string) (int, map[string]any) {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("X-Forwarded-Host", host)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	return rec.Code, body
}

func testTenants() map[string]*domain.Tenant {
	return map[string]*domain.Tenant{
		"111111": {ID: uuid.New(), Subdomain: "111111"},
		"222222": {ID: uuid.New(), Subdomain: "222222"},
	}
}

func TestWellKnown_JWKSRouteIsNotShadowedByDiscovery(t *testing.T) {
	mux := newWellKnownTestServer(t, testTenants())

	status, body := getJSON(t, mux, "111111-id.dipolehq.com", "/.well-known/openid-configuration/jwks")
	if status != http.StatusOK {
		t.Fatalf("JWKS status = %d, want 200", status)
	}
	if _, ok := body["keys"]; !ok {
		t.Fatalf("JWKS response has no keys field: %v", body)
	}
	if _, ok := body["issuer"]; ok {
		t.Error("JWKS response contains issuer — discovery document served on the JWKS path")
	}
}

func TestWellKnown_IdentityHostAdvertisesItselfAsIssuer(t *testing.T) {
	mux := newWellKnownTestServer(t, testTenants())

	status, body := getJSON(t, mux, "111111-id.dipolehq.com", "/.well-known/openid-configuration")
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	if want := "https://111111-id.dipolehq.com"; body["issuer"] != want {
		t.Errorf("issuer = %v, want %v", body["issuer"], want)
	}
	if _, ok := body["authorization_endpoint"]; ok {
		t.Error("identity host returned OAuth authorization server metadata")
	}
}

func TestWellKnown_MCPHostAdvertisesItselfAsIssuer(t *testing.T) {
	mux := newWellKnownTestServer(t, testTenants())

	for _, path := range []string{
		"/.well-known/oauth-authorization-server",
		"/.well-known/openid-configuration",
	} {
		status, body := getJSON(t, mux, "111111-mcp.dipolehq.com", path)
		if status != http.StatusOK {
			t.Fatalf("%s status = %d, want 200", path, status)
		}
		if want := "https://111111-mcp.dipolehq.com"; body["issuer"] != want {
			t.Errorf("%s issuer = %v, want %v", path, body["issuer"], want)
		}
		if want := "https://111111-mcp.dipolehq.com/connect/authorize"; body["authorization_endpoint"] != want {
			t.Errorf("%s authorization_endpoint = %v, want %v", path, body["authorization_endpoint"], want)
		}
	}
}

func TestWellKnown_PerTenantIsolation(t *testing.T) {
	mux := newWellKnownTestServer(t, testTenants())

	_, a := getJSON(t, mux, "111111-mcp.dipolehq.com", "/.well-known/oauth-authorization-server")
	_, b := getJSON(t, mux, "222222-mcp.dipolehq.com", "/.well-known/oauth-authorization-server")

	if a["issuer"] == b["issuer"] {
		t.Fatalf("both tenants served the same issuer %v", a["issuer"])
	}
	if a["issuer"] != "https://111111-mcp.dipolehq.com" {
		t.Errorf("tenant A issuer = %v", a["issuer"])
	}
	if b["issuer"] != "https://222222-mcp.dipolehq.com" {
		t.Errorf("tenant B issuer = %v", b["issuer"])
	}
}

func TestWellKnown_GatewayHostServesNoDiscovery(t *testing.T) {
	mux := newWellKnownTestServer(t, testTenants())

	for _, path := range []string{
		"/.well-known/oauth-authorization-server",
		"/.well-known/openid-configuration",
	} {
		if status, _ := getJSON(t, mux, "111111-mcpgateway.dipolehq.com", path); status != http.StatusNotFound {
			t.Errorf("%s on gateway host = %d, want 404", path, status)
		}
	}
}

func TestWellKnown_NonTenantHostsFailClosed(t *testing.T) {
	mux := newWellKnownTestServer(t, testTenants())

	for _, host := range []string{
		"api.dipolehq.com",
		"console.dipolehq.com",
		"dipolehq.com",
		"111111-mcp.evil.com",
		"dipole-cloud-abc123-uc.a.run.app",
	} {
		for _, path := range []string{
			"/.well-known/oauth-authorization-server",
			"/.well-known/openid-configuration",
		} {
			status, body := getJSON(t, mux, host, path)
			if status != http.StatusNotFound {
				t.Errorf("%s %s = %d, want 404 (got body %v)", host, path, status, body)
			}
		}
	}
}

func TestWellKnown_UnknownTenantFailsClosed(t *testing.T) {
	mux := newWellKnownTestServer(t, testTenants())

	if status, _ := getJSON(t, mux, "999999-mcp.dipolehq.com", "/.well-known/oauth-authorization-server"); status != http.StatusNotFound {
		t.Errorf("unknown tenant = %d, want 404", status)
	}
}

func TestWellKnown_InvalidStoredSubdomainFailsClosed(t *testing.T) {
	mux := newWellKnownTestServer(t, map[string]*domain.Tenant{
		"bad_subdomain": {ID: uuid.New(), Subdomain: "bad_subdomain"},
	})

	status, body := getJSON(t, mux, "bad_subdomain-mcp.dipolehq.com", "/.well-known/oauth-authorization-server")
	if status != http.StatusNotFound {
		t.Errorf("status = %d, want 404", status)
	}
	if iss, ok := body["issuer"]; ok {
		t.Errorf("served a document with issuer %q for an unbuildable subdomain", iss)
	}
}

func TestWellKnown_JWKSServedOnAnyHost(t *testing.T) {
	mux := newWellKnownTestServer(t, testTenants())

	for _, host := range []string{
		"111111-id.dipolehq.com",
		"111111-mcp.dipolehq.com",
		"api.dipolehq.com",
	} {
		status, body := getJSON(t, mux, host, "/.well-known/openid-configuration/jwks")
		if status != http.StatusOK {
			t.Errorf("JWKS on %s = %d, want 200", host, status)
		}
		if _, ok := body["keys"]; !ok {
			t.Errorf("JWKS on %s has no keys", host)
		}
	}
}

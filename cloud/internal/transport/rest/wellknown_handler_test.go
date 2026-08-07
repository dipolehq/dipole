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

// Exercises the real ServeMux + handlers over HTTP, because the behaviour that
// matters here is a function of routing and the request hostname — neither of
// which a unit test on the handler methods would cover.
func newWellKnownTestServer(t *testing.T, tenants map[string]*domain.Tenant) *http.ServeMux {
	t.Helper()

	issuer, err := identity.NewIdentityIssuer("https://id.dipolehq.com")
	if err != nil {
		t.Fatal(err)
	}

	lookup := &fakeTenantLookup{bySubdomain: tenants}
	resolver := NewTenantHostResolver(lookup, platform.NewHosts("dipolehq.com"), false)

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

// The exact-vs-prefix distinction matters: if Go's ServeMux treated
// "/.well-known/openid-configuration" as a prefix, the JWKS route would be
// shadowed and relying parties fetching JWKS would silently receive a discovery
// document instead of keys.
func TestWellKnown_JWKSRouteIsNotShadowedByDiscovery(t *testing.T) {
	mux := newWellKnownTestServer(t, testTenants())

	status, body := getJSON(t, mux, "111111-id.dipolehq.com", "/.well-known/openid-configuration/jwks")
	if status != http.StatusOK {
		t.Fatalf("JWKS status = %d, want 200", status)
	}
	if _, ok := body["keys"]; !ok {
		t.Fatalf("JWKS response has no `keys` field — the discovery route shadowed it: %v", body)
	}
	if _, ok := body["issuer"]; ok {
		t.Error("JWKS response contains `issuer` — discovery document served on the JWKS path")
	}
}

// The -id host must advertise ITSELF as issuer. Returning the -mcp host here is
// the bug that made AWS STS / Azure Entra / GCP WIF unable to verify our tokens,
// since OIDC requires issuer == the URL discovery was fetched from.
func TestWellKnown_IdentityHostAdvertisesItselfAsIssuer(t *testing.T) {
	mux := newWellKnownTestServer(t, testTenants())

	status, body := getJSON(t, mux, "111111-id.dipolehq.com", "/.well-known/openid-configuration")
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	if want := "https://111111-id.dipolehq.com"; body["issuer"] != want {
		t.Errorf("issuer = %v, want %v", body["issuer"], want)
	}
	// Must not leak the authorization server's endpoints onto the issuer host.
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

// Two tenants hitting their own hostnames must each get their own issuer from
// the same running process. This is the core property Fix 10 exists to provide.
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

// The gateway host serves neither discovery document.
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

// Fail closed: a non-tenant hostname must never be served a discovery document
// for some default tenant.
func TestWellKnown_NonTenantHostsFailClosed(t *testing.T) {
	mux := newWellKnownTestServer(t, testTenants())

	for _, host := range []string{
		"api.dipolehq.com",
		"console.dipolehq.com",
		"dipolehq.com",
		"111111-mcp.evil.com",
		"dipole-cloud-abc123-uc.a.run.app", // Worker failed to set X-Forwarded-Host
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

// An unknown tenant must 404 rather than fall back to any configured tenant.
func TestWellKnown_UnknownTenantFailsClosed(t *testing.T) {
	mux := newWellKnownTestServer(t, testTenants())

	if status, _ := getJSON(t, mux, "999999-mcp.dipolehq.com", "/.well-known/oauth-authorization-server"); status != http.StatusNotFound {
		t.Errorf("unknown tenant = %d, want 404", status)
	}
}

// A stored subdomain that can't form a hostname must not yield a document with
// an empty issuer — resolution fails instead.
func TestWellKnown_InvalidStoredSubdomainFailsClosed(t *testing.T) {
	// ParseHost accepts the underscore (we stay liberal about what arrives), so
	// the request reaches the tenant; building its canonical URL is what fails.
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

// JWKS is intentionally host-agnostic: the signing key is shared platform-wide,
// it's public key material, and a relying party must never fail to fetch it.
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

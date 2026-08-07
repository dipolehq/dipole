package platform

import (
	"strings"
	"testing"
)

func TestBuildURLs_HyphenPattern(t *testing.T) {
	h := NewHosts("dipolehq.com")

	if got, want := h.MCPAuthServerURL("647251"), "https://647251-mcp.dipolehq.com"; got != want {
		t.Errorf("MCPAuthServerURL = %q, want %q", got, want)
	}
	if got, want := h.IdentityIssuerURL("647251"), "https://647251-id.dipolehq.com"; got != want {
		t.Errorf("IdentityIssuerURL = %q, want %q", got, want)
	}
	if got, want := h.MCPGatewayURL("647251"), "https://647251-mcpgateway.dipolehq.com"; got != want {
		t.Errorf("MCPGatewayURL = %q, want %q", got, want)
	}
}

func TestNewHosts_EmptyDomainFallsBackToDefault(t *testing.T) {
	if got := NewHosts("").Domain(); got != DefaultDomain {
		t.Errorf("Domain = %q, want %q", got, DefaultDomain)
	}
}

func TestURLForService_UnknownServiceReturnsEmpty(t *testing.T) {
	h := NewHosts("dipolehq.com")
	if got := h.URLForService("647251", Service("console")); got != "" {
		t.Errorf("URLForService(unknown) = %q, want empty", got)
	}
}

// Round-trip: anything we build must parse back to what we started with.
func TestParseHost_RoundTripsBuiltURLs(t *testing.T) {
	h := NewHosts("dipolehq.com")

	for _, svc := range []Service{ServiceMCPAuth, ServiceIdentity, ServiceGateway} {
		url := h.URLForService("647251", svc)
		host := url[len("https://"):]

		sub, gotSvc, ok := h.ParseHost(host)
		if !ok {
			t.Fatalf("ParseHost(%q) not ok", host)
		}
		if sub != "647251" {
			t.Errorf("ParseHost(%q) subdomain = %q, want 647251", host, sub)
		}
		if gotSvc != svc {
			t.Errorf("ParseHost(%q) service = %q, want %q", host, gotSvc, svc)
		}
	}
}

func TestParseHost(t *testing.T) {
	h := NewHosts("dipolehq.com")

	tests := []struct {
		name    string
		host    string
		wantSub string
		wantSvc Service
		wantOK  bool
	}{
		{"mcp auth server", "647251-mcp.dipolehq.com", "647251", ServiceMCPAuth, true},
		{"identity issuer", "647251-id.dipolehq.com", "647251", ServiceIdentity, true},
		{"gateway", "647251-mcpgateway.dipolehq.com", "647251", ServiceGateway, true},

		{"strips port", "647251-mcp.dipolehq.com:443", "647251", ServiceMCPAuth, true},
		{"case insensitive", "647251-MCP.DIPOLEHQ.COM", "647251", ServiceMCPAuth, true},
		{"trailing dot (FQDN form)", "647251-mcp.dipolehq.com.", "647251", ServiceMCPAuth, true},
		{"surrounding whitespace", "  647251-mcp.dipolehq.com  ", "647251", ServiceMCPAuth, true},

		// Subdomains may themselves contain hyphens (legacy slug-style tenants).
		// Only the LAST hyphen-delimited segment is the service.
		{"hyphenated subdomain", "acme-corp-mcp.dipolehq.com", "acme-corp", ServiceMCPAuth, true},
		{"subdomain ending in id", "foo-id-mcp.dipolehq.com", "foo-id", ServiceMCPAuth, true},

		// gateway must not be shadowed by the shorter "mcp" service
		{"gateway not parsed as mcp", "acme-mcpgateway.dipolehq.com", "acme", ServiceGateway, true},

		// Everything below must fail closed.
		{"api host is not a tenant host", "api.dipolehq.com", "", "", false},
		{"console host is not a tenant host", "console.dipolehq.com", "", "", false},
		{"staging api host", "staging-api.dipolehq.com", "", "", false},
		{"apex domain", "dipolehq.com", "", "", false},
		{"wrong domain", "647251-mcp.evil.com", "", "", false},
		{"domain as substring of attacker host", "647251-mcp.dipolehq.com.evil.com", "", "", false},
		{"multi-label prefix", "extra.647251-mcp.dipolehq.com", "", "", false},
		{"unknown service", "647251-console.dipolehq.com", "", "", false},
		{"missing subdomain", "-mcp.dipolehq.com", "", "", false},
		{"no service suffix", "647251.dipolehq.com", "", "", false},
		{"empty", "", "", "", false},
		{"cloud run internal host", "dipole-cloud-abc123-uc.a.run.app", "", "", false},
		{"localhost", "localhost:8080", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub, svc, ok := h.ParseHost(tt.host)
			if ok != tt.wantOK {
				t.Fatalf("ParseHost(%q) ok = %v, want %v", tt.host, ok, tt.wantOK)
			}
			if sub != tt.wantSub {
				t.Errorf("ParseHost(%q) subdomain = %q, want %q", tt.host, sub, tt.wantSub)
			}
			if svc != tt.wantSvc {
				t.Errorf("ParseHost(%q) service = %q, want %q", tt.host, svc, tt.wantSvc)
			}
		})
	}
}

// The platform domain is configurable; parsing must follow it.
func TestParseHost_RespectsConfiguredDomain(t *testing.T) {
	h := NewHosts("staging.dipolehq.com")

	if sub, svc, ok := h.ParseHost("647251-mcp.staging.dipolehq.com"); !ok || sub != "647251" || svc != ServiceMCPAuth {
		t.Errorf("ParseHost on configured domain = (%q, %q, %v), want (647251, mcp, true)", sub, svc, ok)
	}
	// The production domain must NOT resolve when a staging domain is configured.
	if _, _, ok := h.ParseHost("647251-mcp.dipolehq.com"); ok {
		t.Error("ParseHost matched a different domain than configured")
	}
}

func TestIsValidSubdomain(t *testing.T) {
	tests := []struct {
		name string
		sub  string
		want bool
	}{
		{"generated numeric subdomain", "647251", true},
		{"lowercase alphanumeric", "acme123", true},
		{"interior hyphen", "acme-corp", true},
		{"dev bootstrap subdomain", "dev", true},
		{"single character", "a", true},

		// Underscores are legal in DNS records generally but NOT in hostnames;
		// TLS/SNI and many HTTP clients reject them. Legacy slug-derived
		// subdomains could contain one.
		{"underscore rejected", "testing_dipole", false},

		// Hostnames are case-insensitive, but the tenant lookup is a
		// case-sensitive column comparison — so a mixed-case subdomain would
		// build a URL that then resolves to no tenant.
		{"uppercase rejected", "Acme", false},

		{"empty rejected", "", false},
		{"leading hyphen rejected", "-acme", false},
		{"trailing hyphen rejected", "acme-", false},
		{"dot rejected", "acme.corp", false},
		{"space rejected", "acme corp", false},
		{"too long rejected", strings.Repeat("a", maxSubdomainLen+1), false},
		{"at length limit accepted", strings.Repeat("a", maxSubdomainLen), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidSubdomain(tt.sub); got != tt.want {
				t.Errorf("IsValidSubdomain(%q) = %v, want %v", tt.sub, got, tt.want)
			}
		})
	}
}

// A URL we can't build correctly must come back empty so callers fail closed,
// never as a malformed hostname that fails later at the IdP or cloud provider.
func TestBuildURLs_InvalidSubdomainYieldsEmpty(t *testing.T) {
	h := NewHosts("dipolehq.com")

	for _, sub := range []string{"", "-bad", "bad-", "Acme", "testing_dipole", "has.dot"} {
		if got := h.MCPAuthServerURL(sub); got != "" {
			t.Errorf("MCPAuthServerURL(%q) = %q, want empty", sub, got)
		}
		if got := h.IdentityIssuerURL(sub); got != "" {
			t.Errorf("IdentityIssuerURL(%q) = %q, want empty", sub, got)
		}
		if got := h.MCPGatewayURL(sub); got != "" {
			t.Errorf("MCPGatewayURL(%q) = %q, want empty", sub, got)
		}
	}
}

// ParseHost stays deliberately permissive: if DNS somehow delivers a request for
// a non-standard hostname, we should still attribute it to the right tenant.
// Strictness belongs on the emitting side (urlFor), not the accepting side.
func TestParseHost_AcceptsHostsWeWouldNotEmit(t *testing.T) {
	h := NewHosts("dipolehq.com")

	sub, svc, ok := h.ParseHost("testing_dipole-mcp.dipolehq.com")
	if !ok {
		t.Fatal("ParseHost rejected a hostname that could still arrive")
	}
	if sub != "testing_dipole" || svc != ServiceMCPAuth {
		t.Errorf("ParseHost = (%q, %q), want (testing_dipole, mcp)", sub, svc)
	}
	// ...but we must refuse to advertise it.
	if h.MCPAuthServerURL(sub) != "" {
		t.Error("built a URL for a subdomain that isn't a valid hostname label")
	}
}

package platform

import (
	"strings"
	"testing"
)

func TestBuildURLs_HyphenPattern(t *testing.T) {
	h := NewHosts("dipolehq.com", "")

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

func TestBuildURLs_StagingPattern(t *testing.T) {
	h := NewHosts("dipolehq.com", "staging")

	if got, want := h.MCPAuthServerURL("647251"), "https://647251-mcp-staging.dipolehq.com"; got != want {
		t.Errorf("MCPAuthServerURL = %q, want %q", got, want)
	}
	if got, want := h.IdentityIssuerURL("647251"), "https://647251-id-staging.dipolehq.com"; got != want {
		t.Errorf("IdentityIssuerURL = %q, want %q", got, want)
	}
	if got, want := h.MCPGatewayURL("647251"), "https://647251-mcpgateway-staging.dipolehq.com"; got != want {
		t.Errorf("MCPGatewayURL = %q, want %q", got, want)
	}
}

func TestNewHosts_EmptyDomainFallsBackToDefault(t *testing.T) {
	if got := NewHosts("", "").Domain(); got != DefaultDomain {
		t.Errorf("Domain = %q, want %q", got, DefaultDomain)
	}
}

func TestURLForService_UnknownServicePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when URLForService called with unknown service")
		}
	}()
	h := NewHosts("dipolehq.com", "")
	h.URLForService("647251", Service("console"))
}

func TestParseHost_RoundTripsBuiltURLs(t *testing.T) {
	h := NewHosts("dipolehq.com", "")

	for _, svc := range []Service{ServiceMCPAuth, ServiceIdentity, ServiceGateway} {
		url := h.URLForService("647251", svc)
		host := url[len("https://"):]

		sub, gotSvc, _, ok := h.ParseHost(host)
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

func TestParseHost_RoundTripsBuiltURLs_Staging(t *testing.T) {
	h := NewHosts("dipolehq.com", "staging")

	for _, svc := range []Service{ServiceMCPAuth, ServiceIdentity, ServiceGateway} {
		url := h.URLForService("647251", svc)
		host := url[len("https://"):]

		sub, gotSvc, env, ok := h.ParseHost(host)
		if !ok {
			t.Fatalf("ParseHost(%q) not ok", host)
		}
		if sub != "647251" {
			t.Errorf("ParseHost(%q) subdomain = %q, want 647251", host, sub)
		}
		if gotSvc != svc {
			t.Errorf("ParseHost(%q) service = %q, want %q", host, gotSvc, svc)
		}
		if env != "staging" {
			t.Errorf("ParseHost(%q) env = %q, want staging", host, env)
		}
	}
}

func TestParseHost(t *testing.T) {
	h := NewHosts("dipolehq.com", "")

	tests := []struct {
		name    string
		host    string
		wantSub string
		wantSvc Service
		wantEnv string
		wantOK  bool
	}{
		{"mcp auth server", "647251-mcp.dipolehq.com", "647251", ServiceMCPAuth, "", true},
		{"identity issuer", "647251-id.dipolehq.com", "647251", ServiceIdentity, "", true},
		{"gateway", "647251-mcpgateway.dipolehq.com", "647251", ServiceGateway, "", true},

		{"strips port", "647251-mcp.dipolehq.com:443", "647251", ServiceMCPAuth, "", true},
		{"case insensitive", "647251-MCP.DIPOLEHQ.COM", "647251", ServiceMCPAuth, "", true},
		{"trailing dot (FQDN form)", "647251-mcp.dipolehq.com.", "647251", ServiceMCPAuth, "", true},
		{"surrounding whitespace", "  647251-mcp.dipolehq.com  ", "647251", ServiceMCPAuth, "", true},

		{"hyphenated subdomain", "acme-corp-mcp.dipolehq.com", "acme-corp", ServiceMCPAuth, "", true},
		{"subdomain ending in id", "foo-id-mcp.dipolehq.com", "foo-id", ServiceMCPAuth, "", true},
		{"gateway not parsed as mcp", "acme-mcpgateway.dipolehq.com", "acme", ServiceGateway, "", true},

		{"staging mcp", "647251-mcp-staging.dipolehq.com", "647251", ServiceMCPAuth, "staging", true},
		{"staging gateway", "647251-mcpgateway-staging.dipolehq.com", "647251", ServiceGateway, "staging", true},
		{"staging identity", "647251-id-staging.dipolehq.com", "647251", ServiceIdentity, "staging", true},
		{"staging hyphenated tenant", "acme-corp-mcp-staging.dipolehq.com", "acme-corp", ServiceMCPAuth, "staging", true},
		{"tenant named staging", "staging-tenant-mcpgateway-staging.dipolehq.com", "staging-tenant", ServiceGateway, "staging", true},

		{"api host is not a tenant host", "api.dipolehq.com", "", "", "", false},
		{"console host is not a tenant host", "console.dipolehq.com", "", "", "", false},
		{"staging api host", "staging-api.dipolehq.com", "", "", "", false},
		{"apex domain", "dipolehq.com", "", "", "", false},
		{"wrong domain", "647251-mcp.evil.com", "", "", "", false},
		{"domain as substring of attacker host", "647251-mcp.dipolehq.com.evil.com", "", "", "", false},
		{"multi-label prefix", "extra.647251-mcp.dipolehq.com", "", "", "", false},
		{"unknown service", "647251-console.dipolehq.com", "", "", "", false},
		{"missing subdomain", "-mcp.dipolehq.com", "", "", "", false},
		{"no service suffix", "647251.dipolehq.com", "", "", "", false},
		{"empty", "", "", "", "", false},
		{"cloud run internal host", "dipole-cloud-abc123-uc.a.run.app", "", "", "", false},
		{"localhost", "localhost:8080", "", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub, svc, env, ok := h.ParseHost(tt.host)
			if ok != tt.wantOK {
				t.Fatalf("ParseHost(%q) ok = %v, want %v", tt.host, ok, tt.wantOK)
			}
			if sub != tt.wantSub {
				t.Errorf("ParseHost(%q) subdomain = %q, want %q", tt.host, sub, tt.wantSub)
			}
			if svc != tt.wantSvc {
				t.Errorf("ParseHost(%q) service = %q, want %q", tt.host, svc, tt.wantSvc)
			}
			if env != tt.wantEnv {
				t.Errorf("ParseHost(%q) env = %q, want %q", tt.host, env, tt.wantEnv)
			}
		})
	}
}

func TestParseHost_RespectsConfiguredDomain(t *testing.T) {
	h := NewHosts("staging.dipolehq.com", "")

	if sub, svc, _, ok := h.ParseHost("647251-mcp.staging.dipolehq.com"); !ok || sub != "647251" || svc != ServiceMCPAuth {
		t.Errorf("ParseHost on configured domain = (%q, %q, %v), want (647251, mcp, true)", sub, svc, ok)
	}
	if _, _, _, ok := h.ParseHost("647251-mcp.dipolehq.com"); ok {
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
		{"underscore rejected", "testing_dipole", false},
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

func TestBuildURLs_InvalidSubdomainYieldsEmpty(t *testing.T) {
	h := NewHosts("dipolehq.com", "")

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

func TestParseHost_AcceptsHostsWeWouldNotEmit(t *testing.T) {
	h := NewHosts("dipolehq.com", "")

	sub, svc, _, ok := h.ParseHost("testing_dipole-mcp.dipolehq.com")
	if !ok {
		t.Fatal("ParseHost rejected a hostname that could still arrive")
	}
	if sub != "testing_dipole" || svc != ServiceMCPAuth {
		t.Errorf("ParseHost = (%q, %q), want (testing_dipole, mcp)", sub, svc)
	}
	if h.MCPAuthServerURL(sub) != "" {
		t.Error("built a URL for a subdomain that isn't a valid hostname label")
	}
}

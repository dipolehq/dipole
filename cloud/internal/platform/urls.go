/*
Package platform is the single source of truth for per-tenant service URLs.

Every tenant gets its own hostname per service, using a HYPHEN-separated
single label:

	https://<tenantSubdomain>-mcp.<domain>          MCP Authorization Server
	https://<tenantSubdomain>-id.<domain>           Identity Issuer (OIDC iss)
	https://<tenantSubdomain>-mcpgateway.<domain>   MCP Identity Gateway

The hyphen (rather than a dot) is deliberate: Cloudflare's free Universal SSL
only covers a single wildcard label (*.<domain>), so <tenant>.mcp.<domain> —
two labels deep — fails the TLS handshake.

This package owns BOTH directions: building a URL from a tenant subdomain, and
parsing a request hostname back into (subdomain, service). The frontend mirror
is web/src/shared/lib/platform-urls.ts — the two must stay in agreement.
*/
package platform

import "strings"

const (
	OIDCCallbackPath = "/callback/oidc"
	SAMLCallbackPath = "/callback/saml"
)

// Service identifies which per-tenant service a hostname belongs to.
type Service string

const (
	ServiceMCPAuth  Service = "mcp"
	ServiceIdentity Service = "id"
	ServiceGateway  Service = "mcpgateway"
)

var knownServices = []Service{ServiceGateway, ServiceMCPAuth, ServiceIdentity}

// DefaultDomain is used when no platform domain is configured.
const DefaultDomain = "dipolehq.com"

// Hosts builds and parses per-tenant service hostnames for one platform domain.
type Hosts struct {
	domain    string
	envSuffix string
}

func NewHosts(domain string, env string) *Hosts {
	if domain == "" {
		domain = DefaultDomain
	}
	envSuffix := ""
	if env == "staging" {
		envSuffix = "-staging"
	}
	return &Hosts{
		domain:    strings.ToLower(strings.TrimSpace(domain)),
		envSuffix: envSuffix,
	}
}

func (h *Hosts) Domain() string { return h.domain }

// MCPAuthServerURL returns the OAuth 2.1 Authorization Server base URL.
func (h *Hosts) MCPAuthServerURL(tenantSubdomain string) string {
	return h.urlFor(tenantSubdomain, ServiceMCPAuth)
}

// IdentityIssuerURL returns the OIDC issuer base URL (the iss claim in tokens).
func (h *Hosts) IdentityIssuerURL(tenantSubdomain string) string {
	return h.urlFor(tenantSubdomain, ServiceIdentity)
}

// MCPGatewayURL returns the MCP Identity Gateway base URL.
func (h *Hosts) MCPGatewayURL(tenantSubdomain string) string {
	return h.urlFor(tenantSubdomain, ServiceGateway)
}

// URLForService builds the base URL for an arbitrary service.
func (h *Hosts) URLForService(tenantSubdomain string, svc Service) string {
	for _, known := range knownServices {
		if svc == known {
			return h.urlFor(tenantSubdomain, svc)
		}
	}
	return ""
}

func (h *Hosts) urlFor(tenantSubdomain string, svc Service) string {
	if !IsValidSubdomain(tenantSubdomain) {
		return ""
	}
	return "https://" + tenantSubdomain + "-" + string(svc) + h.envSuffix + "." + h.domain
}

const maxSubdomainLen = 50

// IsValidSubdomain reports whether a tenant subdomain can legally appear in a
// DNS hostname label. Permits lowercase letters, digits, and interior hyphens.
func IsValidSubdomain(s string) bool {
	if s == "" || len(s) > maxSubdomainLen {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9':
		case c == '-' && i != 0 && i != len(s)-1:
		default:
			return false
		}
	}
	return true
}

// ParseHost turns a request hostname back into the tenant subdomain and the
// service being addressed. Returns ok=false for anything that isn't a
// well-formed per-tenant service host.
func (h *Hosts) ParseHost(host string) (tenantSubdomain string, svc Service, ok bool) {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return "", "", false
	}

	if i := strings.LastIndex(host, ":"); i != -1 && !strings.Contains(host[i:], "]") {
		host = host[:i]
	}
	host = strings.TrimSuffix(host, ".")

	suffix := "." + h.domain
	if !strings.HasSuffix(host, suffix) {
		return "", "", false
	}

	label := strings.TrimSuffix(host, suffix)
	if label == "" || strings.Contains(label, ".") {
		return "", "", false
	}

	for _, known := range knownServices {
		trimmed, found := strings.CutSuffix(label, "-"+string(known))
		if !found {
			continue
		}
		if trimmed == "" {
			return "", "", false
		}
		return trimmed, known, true
	}

	return "", "", false
}

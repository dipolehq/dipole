/*
Package platform is the single source of truth for per-tenant service URLs.

Every tenant gets its own hostname per service, using a HYPHEN-separated
single label:

	https://<tenantSubdomain>-mcp.<domain>          MCP Authorization Server
	https://<tenantSubdomain>-id.<domain>           Identity Issuer (OIDC `iss`)
	https://<tenantSubdomain>-mcpgateway.<domain>   MCP Identity Gateway

The hyphen (rather than a dot, as Aembit uses) is deliberate: Cloudflare's
free Universal SSL only covers a single wildcard label (`*.<domain>`), so
`<tenant>.mcp.<domain>` — two labels deep — fails the TLS handshake. Keeping
the tenant and service in one label stays inside that coverage.

This package deliberately owns BOTH directions:

  - building a URL from a tenant subdomain (used when we hand URLs to
    customers, embed them in tokens, or advertise them in OIDC metadata)
  - parsing a request hostname back into (subdomain, service), which is how
    every request resolves which tenant it belongs to

Keeping both here means the pattern is defined exactly once. The frontend
mirror of this file is web/src/shared/lib/platform-urls.ts — the two must
stay in agreement, because customers copy values from the UI into their IdP
and those values must match what the backend actually sends.
*/
package platform

import "strings"

// Callback paths on the MCP Authorization Server host. These are the redirect
// URIs a customer must register with their Identity Provider, so the value we
// send as redirect_uri and the value the admin console tells them to register
// have to be built from the same constants — a mismatch surfaces as
// redirect_uri_mismatch from the IdP.
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

// knownServices is ordered longest-suffix-first so that a service name which
// is a suffix of another can never shadow it. (Today none are, but ordering
// makes that safe by construction rather than by luck.)
var knownServices = []Service{ServiceGateway, ServiceMCPAuth, ServiceIdentity}

// DefaultDomain is used when no platform domain is configured.
const DefaultDomain = "dipolehq.com"

// Hosts builds and parses per-tenant service hostnames for one platform domain.
type Hosts struct {
	domain string
}

func NewHosts(domain string) *Hosts {
	if domain == "" {
		domain = DefaultDomain
	}
	return &Hosts{domain: strings.ToLower(strings.TrimSpace(domain))}
}

func (h *Hosts) Domain() string { return h.domain }

// MCPAuthServerURL returns the OAuth 2.1 Authorization Server base URL.
// This is the `issuer` in the OAuth AS metadata document (RFC 8414).
func (h *Hosts) MCPAuthServerURL(tenantSubdomain string) string {
	return h.urlFor(tenantSubdomain, ServiceMCPAuth)
}

// IdentityIssuerURL returns the OIDC issuer base URL — the value that goes in
// the `iss` claim of tokens we sign, and therefore the URL that AWS STS /
// Azure Entra / GCP WIF will fetch discovery + JWKS from.
//
// Note this is intentionally a DIFFERENT host from MCPAuthServerURL. Aembit
// documents the same split (`.id.` for the token issuer, `.mcp.` for the
// authorization server) and MCP servers must be configured with this one.
func (h *Hosts) IdentityIssuerURL(tenantSubdomain string) string {
	return h.urlFor(tenantSubdomain, ServiceIdentity)
}

// MCPGatewayURL returns the MCP Identity Gateway base URL. Used as the
// audience of tokens minted for the Gateway (left-side auth).
func (h *Hosts) MCPGatewayURL(tenantSubdomain string) string {
	return h.urlFor(tenantSubdomain, ServiceGateway)
}

// URLForService builds the base URL for an arbitrary service. Returns an empty
// string for an unrecognised service so callers can't silently produce a
// nonsense hostname.
func (h *Hosts) URLForService(tenantSubdomain string, svc Service) string {
	for _, known := range knownServices {
		if svc == known {
			return h.urlFor(tenantSubdomain, svc)
		}
	}
	return ""
}

// urlFor returns "" rather than a malformed hostname when the subdomain can't
// legally appear in one.
//
// This matters because these URLs are not cosmetic: an identity issuer URL
// becomes the `iss` claim on tokens that AWS STS / Azure Entra / GCP WIF
// verify, and a callback URL is what a customer registers with their IdP.
// Emitting `https://-id.<domain>` (empty subdomain) or a subdomain containing
// characters DNS forbids would be accepted here and then fail far away, with an
// opaque error from the cloud provider or the IdP. Callers must treat "" as
// "cannot build this URL" and fail closed.
func (h *Hosts) urlFor(tenantSubdomain string, svc Service) string {
	if !IsValidSubdomain(tenantSubdomain) {
		return ""
	}
	return "https://" + tenantSubdomain + "-" + string(svc) + "." + h.domain
}

// maxSubdomainLen keeps `<subdomain>-<service>` inside the 63-character DNS
// label limit, with room for the longest service name ("mcpgateway").
const maxSubdomainLen = 50

// IsValidSubdomain reports whether a tenant subdomain can legally appear inside
// a DNS hostname label.
//
// Permits lowercase letters, digits, and interior hyphens — the RFC 1123 rules
// for a hostname label. Notably it rejects UNDERSCORES (valid in DNS records
// generally, but not in hostnames, and rejected by TLS/SNI and many HTTP
// clients) and rejects uppercase (hostnames are case-insensitive, but the
// tenant lookup is a case-sensitive column comparison, so allowing mixed case
// would let a hostname resolve to no tenant).
//
// Subdomains generated at signup are 6 digits, so they always pass. This guards
// against subdomains introduced by any other route: the local dev bootstrap,
// data migrations, or manual SQL.
func IsValidSubdomain(s string) bool {
	if s == "" || len(s) > maxSubdomainLen {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9':
			// allowed anywhere
		case c == '-' && i != 0 && i != len(s)-1:
			// allowed only in the interior
		default:
			return false
		}
	}
	return true
}

// ParseHost turns a request hostname back into the tenant subdomain and the
// service being addressed. Any port is stripped and matching is
// case-insensitive, since hostnames are.
//
// It returns ok=false for anything that isn't a well-formed per-tenant service
// host — a different domain, a bare apex/other subdomain (api, console), a
// multi-label prefix, an unknown service, or an empty subdomain. Callers MUST
// treat ok=false as "cannot determine tenant" and fail closed rather than
// falling back to a default tenant; that fallback is precisely what allowed
// requests to be silently served from the wrong tenant.
func (h *Hosts) ParseHost(host string) (tenantSubdomain string, svc Service, ok bool) {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return "", "", false
	}

	// Strip port. Uses LastIndex so IPv6 literals (which contain colons) don't
	// get mangled — though those never carry a tenant host anyway.
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
		// Either the apex itself, or something like foo.<tenant>-mcp.<domain>
		// which is not a hostname we serve.
		return "", "", false
	}

	for _, known := range knownServices {
		trimmed, found := strings.CutSuffix(label, "-"+string(known))
		if !found {
			continue
		}
		if trimmed == "" {
			// "-mcp.<domain>" — no tenant subdomain present.
			return "", "", false
		}
		return trimmed, known, true
	}

	return "", "", false
}

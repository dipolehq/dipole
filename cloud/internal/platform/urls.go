// Package platform builds and parses per-tenant service URLs.
package platform

import "strings"

const (
	OIDCCallbackPath = "/callback/oidc"
	SAMLCallbackPath = "/callback/saml"
)

type Service string

const (
	ServiceMCPAuth  Service = "mcp"
	ServiceIdentity Service = "id"
	ServiceGateway  Service = "mcpgateway"
)

var knownServices = []Service{ServiceGateway, ServiceMCPAuth, ServiceIdentity}

const DefaultDomain = "dipolehq.com"

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

func (h *Hosts) MCPAuthServerURL(tenantSubdomain string) string {
	return h.urlFor(tenantSubdomain, ServiceMCPAuth)
}

func (h *Hosts) IdentityIssuerURL(tenantSubdomain string) string {
	return h.urlFor(tenantSubdomain, ServiceIdentity)
}

func (h *Hosts) MCPGatewayURL(tenantSubdomain string) string {
	return h.urlFor(tenantSubdomain, ServiceGateway)
}

func (h *Hosts) URLForService(tenantSubdomain string, svc Service) string {
	switch svc {
	case ServiceMCPAuth, ServiceIdentity, ServiceGateway:
		return h.urlFor(tenantSubdomain, svc)
	default:
		panic("platform: unknown service " + string(svc))
	}
}

func (h *Hosts) urlFor(tenantSubdomain string, svc Service) string {
	if !IsValidSubdomain(tenantSubdomain) {
		return ""
	}
	return "https://" + tenantSubdomain + "-" + string(svc) + h.envSuffix + "." + h.domain
}

const maxSubdomainLen = 50

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

func (h *Hosts) ParseHost(host string) (tenantSubdomain string, svc Service, env string, ok bool) {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return "", "", "", false
	}

	if i := strings.LastIndex(host, ":"); i != -1 && !strings.Contains(host[i:], "]") {
		host = host[:i]
	}
	host = strings.TrimSuffix(host, ".")

	suffix := "." + h.domain
	if !strings.HasSuffix(host, suffix) {
		return "", "", "", false
	}

	label := strings.TrimSuffix(host, suffix)
	if label == "" || strings.Contains(label, ".") {
		return "", "", "", false
	}

	parsedEnv := ""
	if strings.HasSuffix(label, "-staging") {
		label = strings.TrimSuffix(label, "-staging")
		parsedEnv = "staging"
	}

	for _, known := range knownServices {
		trimmed, found := strings.CutSuffix(label, "-"+string(known))
		if !found {
			continue
		}
		if trimmed == "" {
			return "", "", "", false
		}
		return trimmed, known, parsedEnv, true
	}

	return "", "", "", false
}

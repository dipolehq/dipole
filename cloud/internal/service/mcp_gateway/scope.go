/*
scope.go defines the per-request tenant scope for MCP Identity Gateway traffic.

One Cloud process serves the Gateway for every tenant, so nothing in this
package may capture a tenant at construction time. Anything that did was served
as whichever tenant happened to be configured at startup, regardless of which
hostname the request arrived on.

TenantScope is the resolved identity of the tenant a single request belongs to.
It is derived from the request hostname by rest.TenantHostResolver and threaded
down through every call that needs it.
*/
package mcp_gateway

import "github.com/google/uuid"

// TenantScope identifies the tenant an in-flight Gateway request operates in.
//
// TenantID and GatewayURL deliberately travel as one value. Right-side policy
// lookup needs both: the tenant to scope the query, and the Gateway URL to
// recognise the Gateway itself as the Client Workload of that policy (matched
// against an OIDC ID Token Audience identifier). Pairing them in a struct means
// a caller cannot accidentally combine one tenant's ID with another tenant's
// URL, which two positional string/UUID parameters would allow.
type TenantScope struct {
	TenantID uuid.UUID

	// TenantSubdomain is the tenant's DNS-safe subdomain (e.g. "647251" or
	// "testing-dipole"). Carried here so downstream code that needs to build
	// URLs for other services of the same tenant (like the MCP Auth Server URL
	// for Protected Resource Metadata) does not have to re-resolve the tenant.
	TenantSubdomain string

	// GatewayURL is the canonical Gateway base URL for this tenant, e.g.
	// https://647251-mcpgateway.dipolehq.com. Comparisons against it normalise
	// a trailing slash (see audienceMatches, matchesGatewayClient), so either
	// form is accepted from tokens and stored identifiers.
	GatewayURL string
}

// Valid reports whether this scope is safe to act on.
//
// Callers must fail closed when it is not. An empty GatewayURL would match a
// Client Workload whose audience identifier is also empty, and a nil TenantID
// would scope a query to no tenant — both turn "could not determine the tenant"
// into "silently used the wrong one".
func (s TenantScope) Valid() bool {
	return s.TenantID != uuid.Nil && s.GatewayURL != "" && s.TenantSubdomain != ""
}

// Subdomain returns the tenant's DNS-safe subdomain.
func (s TenantScope) Subdomain() string {
	return s.TenantSubdomain
}

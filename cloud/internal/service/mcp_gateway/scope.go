// scope.go defines TenantScope, the per-request tenant identity threaded through every MCP Identity Gateway call.
package mcp_gateway

import "github.com/google/uuid"

type TenantScope struct {
	TenantID        uuid.UUID
	TenantSubdomain string
	GatewayURL      string
}

func (s TenantScope) Valid() bool {
	return s.TenantID != uuid.Nil && s.GatewayURL != "" && s.TenantSubdomain != ""
}

func (s TenantScope) Subdomain() string {
	return s.TenantSubdomain
}

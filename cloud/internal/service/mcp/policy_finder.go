/*
policy_finder.go implements MCP-specific policy matching.

MCPPolicyFinder finds Access Policies where the Client Workload has a
redirect_uri identifier matching the MCP client's redirect URI, AND the
Server Workload matches the MCP server's host+port with protocol=MCP.
*/
package mcp

import (
	"context"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
)

type MCPPolicyFinder struct {
	repo repository.AccessPolicyRepository
}

func NewMCPPolicyFinder(repo repository.AccessPolicyRepository) *MCPPolicyFinder {
	return &MCPPolicyFinder{repo: repo}
}

func (f *MCPPolicyFinder) FindMCPPolicy(ctx context.Context, tenantID uuid.UUID, redirectURI string, host string, port int) (*domain.AccessPolicyWithRelations, error) {
	policies, err := f.repo.FindByServerTarget(ctx, tenantID, host, port, nil)
	if err != nil {
		return nil, err
	}

	for i := range policies {
		p := &policies[i]

		if !p.Policy.IsActive {
			continue
		}

		if p.ServerWorkload.AppProtocol != domain.AppProtocolMCP {
			continue
		}

		if !clientHasRedirectURI(p.ClientWorkload, redirectURI) {
			continue
		}

		return p, nil
	}

	return nil, nil
}

func clientHasRedirectURI(cw domain.ClientWorkload, redirectURI string) bool {
	for _, id := range cw.Identifiers {
		if id.Type == domain.IdentifierTypeRedirectURI {
			if MatchRedirectURI(redirectURI, id.Value) {
				return true
			}
		}
	}
	return false
}

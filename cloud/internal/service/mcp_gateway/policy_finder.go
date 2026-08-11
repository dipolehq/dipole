/*
policy_finder.go implements the RightSidePolicyFinder interface using the
AccessPolicyRepository to find MCP policies where the Gateway is the client.
*/
package mcp_gateway

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
)

type DBRightSidePolicyFinder struct {
	policyRepo repository.AccessPolicyRepository
}

func NewDBRightSidePolicyFinder(policyRepo repository.AccessPolicyRepository) *DBRightSidePolicyFinder {
	return &DBRightSidePolicyFinder{policyRepo: policyRepo}
}

func (f *DBRightSidePolicyFinder) FindGatewayPolicies(ctx context.Context, tenantID uuid.UUID, gatewayURL string) ([]RightSidePolicy, error) {
	policies, err := f.policyRepo.FindByAppProtocol(ctx, tenantID, domain.AppProtocolMCP)
	if err != nil {
		return nil, err
	}

	var results []RightSidePolicy
	for _, p := range policies {
		if !matchesGatewayClient(p.ClientWorkload, gatewayURL) {
			continue
		}
		results = append(results, mapToRightSidePolicy(&p))
	}
	return results, nil
}

func (f *DBRightSidePolicyFinder) FindPolicyForServer(ctx context.Context, tenantID uuid.UUID, gatewayURL string, serverHost string, serverPort int) (*RightSidePolicy, error) {
	policies, err := f.policyRepo.FindByServerTarget(ctx, tenantID, serverHost, serverPort, nil)
	if err != nil {
		return nil, err
	}

	for _, p := range policies {
		if p.ServerWorkload.AppProtocol != domain.AppProtocolMCP {
			continue
		}
		if !matchesGatewayClient(p.ClientWorkload, gatewayURL) {
			continue
		}
		rsp := mapToRightSidePolicy(&p)
		return &rsp, nil
	}
	return nil, nil
}

func matchesGatewayClient(cw domain.ClientWorkload, gatewayURL string) bool {
	normalized := strings.TrimSuffix(gatewayURL, "/")
	for _, id := range cw.Identifiers {
		if id.Type != domain.IdentifierTypeOIDCIDTokenAudience {
			continue
		}
		if strings.TrimSuffix(id.Value, "/") == normalized {
			return true
		}
	}
	return false
}

func mapToRightSidePolicy(p *domain.AccessPolicyWithRelations) RightSidePolicy {
	rsp := RightSidePolicy{
		PolicyID: p.Policy.ID,
		ServerWorkload: ServerInfo{
			ID:   p.ServerWorkload.ID,
			Name: p.ServerWorkload.Name,
			Host: p.ServerWorkload.Host,
			Port: p.ServerWorkload.Port,
			Path: p.ServerWorkload.URLPath,
			TLS:  p.ServerWorkload.TLS,
		},
	}

	if len(p.Policy.CredentialProviderIDs) > 0 {
		rsp.CredentialProviderID = p.Policy.CredentialProviderIDs[0].CredentialProviderID
	}

	if len(p.CredentialProviders) > 0 {
		rsp.CredentialType = p.CredentialProviders[0].Type
		rsp.CredentialConfigEncrypted = p.CredentialProviders[0].ConfigEncrypted
	}

	return rsp
}

// consent_config_resolver.go implements CPConsentConfigResolver, which resolves OAuth configuration for the consent flow from a Credential Provider's encrypted config.
package mcp_gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/crypto"
	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
)

type CPConsentConfigResolver struct {
	policyRepo repository.AccessPolicyRepository
	cpRepo     repository.CredentialProviderRepository
	encKey     []byte
}

func NewCPConsentConfigResolver(policyRepo repository.AccessPolicyRepository, cpRepo repository.CredentialProviderRepository, encKey []byte) *CPConsentConfigResolver {
	return &CPConsentConfigResolver{
		policyRepo: policyRepo,
		cpRepo:     cpRepo,
		encKey:     encKey,
	}
}

func (r *CPConsentConfigResolver) GetConsentConfig(ctx context.Context, tenantID uuid.UUID, serverWorkloadID uuid.UUID, gatewayURL string) (*ConsentConfig, error) {
	if gatewayURL == "" {
		return nil, ErrNoGatewayURL
	}

	policies, err := r.policyRepo.FindByAppProtocol(ctx, tenantID, domain.AppProtocolMCP)
	if err != nil {
		return nil, fmt.Errorf("find MCP policies: %w", err)
	}

	var cpID uuid.UUID
	for _, p := range policies {
		if p.ServerWorkload.ID == serverWorkloadID && len(p.Policy.CredentialProviderIDs) > 0 {
			cpID = p.Policy.CredentialProviderIDs[0].CredentialProviderID
			break
		}
	}

	if cpID == uuid.Nil {
		return nil, fmt.Errorf("no credential provider found for server workload %s", serverWorkloadID)
	}

	cp, err := r.cpRepo.GetByID(ctx, tenantID, cpID)
	if err != nil || cp == nil {
		return nil, fmt.Errorf("load credential provider: %w", err)
	}

	var plaintext []byte
	if len(r.encKey) > 0 && len(cp.ConfigEncrypted) > 0 {
		plaintext, err = crypto.Decrypt(cp.ConfigEncrypted, r.encKey)
		if err != nil {
			return nil, fmt.Errorf("decrypt credential provider config: %w", err)
		}
	} else {
		plaintext = cp.ConfigEncrypted
	}

	var oauthCfg struct {
		AuthorizationEndpoint string `json:"authorization_endpoint"`
		TokenEndpoint         string `json:"token_endpoint"`
		ClientID              string `json:"client_id"`
		ClientSecret          string `json:"client_secret"`
		Scopes                string `json:"scopes"`
	}
	if err := json.Unmarshal(plaintext, &oauthCfg); err != nil {
		return nil, fmt.Errorf("parse credential provider config: %w", err)
	}

	callbackURL := strings.TrimSuffix(gatewayURL, "/") + ConsentCallbackPath

	return &ConsentConfig{
		AuthorizationEndpoint: oauthCfg.AuthorizationEndpoint,
		TokenEndpoint:         oauthCfg.TokenEndpoint,
		ClientID:              oauthCfg.ClientID,
		ClientSecret:          oauthCfg.ClientSecret,
		Scopes:                oauthCfg.Scopes,
		GatewayCallbackURL:    callbackURL,
	}, nil
}

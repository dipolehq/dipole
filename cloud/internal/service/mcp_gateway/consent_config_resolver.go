/*
consent_config_resolver.go implements ConsentConfigResolver using the
AccessPolicyRepository and CredentialProviderRepository.

CPConsentConfigResolver finds the credential provider config for a given
MCP server workload by looking up the right-side policy, loading the
credential provider, decrypting its config, and extracting the OAuth fields.
*/
package mcp_gateway

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
)

// CPConsentConfigResolver holds no Gateway URL. The callback URL it produces must
// belong to the tenant handling the request, so the URL is supplied per call from
// that request's TenantScope — a captured URL sent every tenant's users to one
// tenant's callback.
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
		plaintext, err = decryptAESGCM(cp.ConfigEncrypted, r.encKey)
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

	// Tolerates either form of the supplied URL: this value is also what the
	// customer registers as the redirect URI with the downstream provider, and
	// the token exchange must send back a byte-identical string.
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

func decryptAESGCM(ciphertext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, ct := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ct, nil)
}

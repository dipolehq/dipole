/*
token_config_resolver.go resolves token configuration from Access Policy
Credential Providers for MCP token issuance.

PolicyTokenConfigResolver loads the Access Policy by ID, finds the associated
Credential Provider (typically OIDC ID Token type for MCP flows), and extracts
the audience, subject, lifetime, algorithm, and refresh settings needed to
issue the access token.
*/
package mcp

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
)

type PolicyTokenConfigResolver struct {
	policyRepo repository.AccessPolicyRepository
	cpRepo     repository.CredentialProviderRepository
	encKey     []byte
}

func NewPolicyTokenConfigResolver(policyRepo repository.AccessPolicyRepository, cpRepo repository.CredentialProviderRepository, encKey []byte) *PolicyTokenConfigResolver {
	return &PolicyTokenConfigResolver{
		policyRepo: policyRepo,
		cpRepo:     cpRepo,
		encKey:     encKey,
	}
}

type oidcCPConfig struct {
	Audience         string         `json:"audience"`
	Subject          string         `json:"subject"`
	LifetimeSeconds  int            `json:"lifetime_seconds"`
	Algorithm        string         `json:"algorithm"`
	CustomClaims     map[string]any `json:"custom_claims"`
	RefreshEnabled   bool           `json:"refresh_enabled"`
	RefreshAbsExpiry int            `json:"refresh_absolute_expiry_seconds"`
}

func (r *PolicyTokenConfigResolver) ResolveTokenConfig(policyID uuid.UUID, tenantID uuid.UUID) (*TokenConfig, error) {
	ctx := context.Background()

	policy, err := r.policyRepo.GetByID(ctx, tenantID, policyID)
	if err != nil {
		return nil, fmt.Errorf("load policy: %w", err)
	}

	if len(policy.CredentialProviderIDs) == 0 {
		return nil, fmt.Errorf("policy has no credential provider configured")
	}

	cpID := policy.CredentialProviderIDs[0].CredentialProviderID
	cp, err := r.cpRepo.GetByID(ctx, tenantID, cpID)
	if err != nil {
		return nil, fmt.Errorf("load credential provider: %w", err)
	}

	cfg, err := r.decryptAndParse(cp)
	if err != nil {
		return nil, err
	}

	lifetime := time.Duration(cfg.LifetimeSeconds) * time.Second
	if lifetime == 0 {
		lifetime = 1 * time.Hour
	}

	algorithm := cfg.Algorithm
	if algorithm == "" {
		algorithm = "ES256"
	}

	refreshAbsExpiry := time.Duration(cfg.RefreshAbsExpiry) * time.Second
	if refreshAbsExpiry == 0 && cfg.RefreshEnabled {
		refreshAbsExpiry = 24 * time.Hour
	}

	return &TokenConfig{
		Audience:         cfg.Audience,
		Subject:          cfg.Subject,
		Lifetime:         lifetime,
		Algorithm:        algorithm,
		CustomClaims:     cfg.CustomClaims,
		RefreshEnabled:   cfg.RefreshEnabled,
		RefreshAbsExpiry: refreshAbsExpiry,
	}, nil
}

func (r *PolicyTokenConfigResolver) decryptAndParse(cp *domain.CredentialProvider) (*oidcCPConfig, error) {
	if cp.Type != domain.CredentialProviderTypeOIDCIDToken &&
		cp.Type != domain.CredentialProviderTypeDipoleAccessToken {
		return nil, fmt.Errorf("unsupported credential provider type for MCP: %s", cp.Type)
	}

	var plaintext []byte
	if len(r.encKey) > 0 && len(cp.ConfigEncrypted) > 0 {
		var err error
		plaintext, err = decryptAESGCM(cp.ConfigEncrypted, r.encKey)
		if err != nil {
			return nil, fmt.Errorf("decrypt credential provider config: %w", err)
		}
	} else {
		plaintext = cp.ConfigEncrypted
	}

	var cfg oidcCPConfig
	if err := json.Unmarshal(plaintext, &cfg); err != nil {
		return nil, fmt.Errorf("parse credential provider config: %w", err)
	}
	return &cfg, nil
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

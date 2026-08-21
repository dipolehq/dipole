// oidc_id_token.go issues OIDC ID Tokens with configurable subject, audience, lifetime, signing algorithm, and custom claims.
package credentials

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/service/identity"
)

type OIDCIDTokenHandler struct {
	issuer *identity.IdentityIssuer
}

func NewOIDCIDTokenHandler(issuer *identity.IdentityIssuer) *OIDCIDTokenHandler {
	return &OIDCIDTokenHandler{issuer: issuer}
}

type oidcIDTokenConfig struct {
	Subject          string            `json:"subject"`
	Audience         string            `json:"audience"`
	LifetimeSeconds  int               `json:"lifetime_seconds"`
	SigningAlgorithm string            `json:"signing_algorithm"`
	CustomClaims     map[string]string `json:"custom_claims"`
}

func (h *OIDCIDTokenHandler) Retrieve(_ context.Context, configJSON []byte, issuerURL string, evidence *domain.ClientEvidence, trustClaims map[string]interface{}) (*domain.Credential, error) {
	var cfg oidcIDTokenConfig
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		return nil, fmt.Errorf("parse oidc id token config: %w", err)
	}

	lifetime := time.Duration(cfg.LifetimeSeconds) * time.Second
	if lifetime == 0 {
		lifetime = 1 * time.Hour
	}

	customClaims := make(map[string]interface{}, len(cfg.CustomClaims))
	for k, v := range cfg.CustomClaims {
		customClaims[k] = v
	}

	token, err := h.issuer.IssueToken(identity.IssuanceRequest{
		IssuerURL:    issuerURL,
		Subject:      cfg.Subject,
		Audience:     cfg.Audience,
		Lifetime:     lifetime,
		Algorithm:    cfg.SigningAlgorithm,
		CustomClaims: customClaims,
		TrustClaims:  trustClaims,
	})
	if err != nil {
		return nil, fmt.Errorf("issue oidc id token: %w", err)
	}

	return &domain.Credential{Type: "bearer", Value: token}, nil
}

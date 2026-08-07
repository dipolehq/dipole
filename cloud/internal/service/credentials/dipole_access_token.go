/*
dipole_access_token.go issues a short-lived platform access token using our
OIDC Identity Issuance Service. Used for authenticating to the Dipole API
itself or as a generic bearer token.
*/
package credentials

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/service/identity"
)

type DipoleAccessTokenHandler struct {
	issuer *identity.IdentityIssuer
}

func NewDipoleAccessTokenHandler(issuer *identity.IdentityIssuer) *DipoleAccessTokenHandler {
	return &DipoleAccessTokenHandler{issuer: issuer}
}

type dipoleAccessTokenConfig struct {
	LifetimeSeconds int `json:"lifetime_seconds"`
}

// issuerURL is the requesting tenant's identity host, used as the token's `iss`.
func (h *DipoleAccessTokenHandler) Retrieve(_ context.Context, configJSON []byte, issuerURL string, evidence *domain.ClientEvidence) (*domain.Credential, error) {
	var cfg dipoleAccessTokenConfig
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		return nil, fmt.Errorf("parse dipole access token config: %w", err)
	}

	lifetime := time.Duration(cfg.LifetimeSeconds) * time.Second
	if lifetime == 0 {
		lifetime = 1 * time.Hour
	}

	token, err := h.issuer.IssueToken(identity.IssuanceRequest{
		IssuerURL: issuerURL,
		Subject:   evidence.DipoleClientID,
		Audience:  evidence.Hostname,
		Lifetime:  lifetime,
	})
	if err != nil {
		return nil, fmt.Errorf("issue dipole access token: %w", err)
	}

	return &domain.Credential{Type: "bearer", Value: token}, nil
}

// jwt_svid.go issues SPIFFE JWT-SVID identity documents, subject set to a SPIFFE ID and audience to the intended recipient's SPIFFE ID.
package credentials

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/service/identity"
)

type JWTSVIDHandler struct {
	issuer *identity.IdentityIssuer
}

func NewJWTSVIDHandler(issuer *identity.IdentityIssuer) *JWTSVIDHandler {
	return &JWTSVIDHandler{issuer: issuer}
}

type jwtSVIDConfig struct {
	SpiffeID        string   `json:"spiffe_id"`
	Audience        []string `json:"audience"`
	LifetimeSeconds int      `json:"lifetime_seconds"`
}

func (h *JWTSVIDHandler) Retrieve(_ context.Context, configJSON []byte, issuerURL string, _ *domain.ClientEvidence, _ map[string]interface{}) (*domain.Credential, error) {
	var cfg jwtSVIDConfig
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		return nil, fmt.Errorf("parse jwt-svid config: %w", err)
	}

	lifetime := time.Duration(cfg.LifetimeSeconds) * time.Second
	if lifetime == 0 {
		lifetime = 5 * time.Minute
	}

	audience := ""
	if len(cfg.Audience) > 0 {
		audience = cfg.Audience[0]
	}

	token, err := h.issuer.IssueToken(identity.IssuanceRequest{
		IssuerURL: issuerURL,
		Subject:   cfg.SpiffeID,
		Audience:  audience,
		Lifetime:  lifetime,
	})
	if err != nil {
		return nil, fmt.Errorf("issue jwt-svid: %w", err)
	}

	return &domain.Credential{Type: "bearer", Value: token}, nil
}

/*
jwt.go issues custom JSON Web Tokens with arbitrary claims. More flexible than
OIDC ID Token — allows any claim structure without OIDC constraints.
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

type JWTHandler struct {
	issuer *identity.IdentityIssuer
}

func NewJWTHandler(issuer *identity.IdentityIssuer) *JWTHandler {
	return &JWTHandler{issuer: issuer}
}

type jwtConfig struct {
	Claims           map[string]string `json:"claims"`
	LifetimeSeconds  int               `json:"lifetime_seconds"`
	SigningAlgorithm string            `json:"signing_algorithm"`
}

func (h *JWTHandler) Retrieve(_ context.Context, configJSON []byte, _ *domain.ClientEvidence, trustClaims map[string]interface{}) (*domain.Credential, error) {
	var cfg jwtConfig
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		return nil, fmt.Errorf("parse jwt config: %w", err)
	}

	lifetime := time.Duration(cfg.LifetimeSeconds) * time.Second
	if lifetime == 0 {
		lifetime = 1 * time.Hour
	}

	customClaims := make(map[string]interface{}, len(cfg.Claims))
	for k, v := range cfg.Claims {
		customClaims[k] = v
	}

	subject := ""
	if s, ok := cfg.Claims["sub"]; ok {
		subject = s
		delete(customClaims, "sub")
	}

	audience := ""
	if a, ok := cfg.Claims["aud"]; ok {
		audience = a
		delete(customClaims, "aud")
	}

	token, err := h.issuer.IssueToken(identity.IssuanceRequest{
		Subject:      subject,
		Audience:     audience,
		Lifetime:     lifetime,
		Algorithm:    cfg.SigningAlgorithm,
		CustomClaims: customClaims,
		TrustClaims:  trustClaims,
	})
	if err != nil {
		return nil, fmt.Errorf("issue jwt: %w", err)
	}

	return &domain.Credential{Type: "bearer", Value: token}, nil
}

// Package service — auth.go implements API token authentication: hashing,
// generation, and resolving a Bearer token to its owning tenant.
package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/repository"
)

const APITokenPrefix = "dipole_pat_"

type APITokenAuthenticator struct {
	tenants repository.TenantRepository
}

func NewAPITokenAuthenticator(tenants repository.TenantRepository) *APITokenAuthenticator {
	return &APITokenAuthenticator{tenants: tenants}
}

func (a *APITokenAuthenticator) ValidateToken(ctx context.Context, token string) (uuid.UUID, string, error) {
	if token == "" {
		return uuid.Nil, "", fmt.Errorf("empty token")
	}

	tenant, err := a.tenants.GetByAPITokenHash(ctx, HashAPIToken(token))
	if err != nil {
		return uuid.Nil, "", err
	}
	if tenant == nil {
		return uuid.Nil, "", fmt.Errorf("invalid token")
	}
	return tenant.ID, "apitoken@" + tenant.Subdomain, nil
}

// ValidateAPIToken implements rest.TenantAuthenticator for the Dipole MCP Server.
func (a *APITokenAuthenticator) ValidateAPIToken(token string) (uuid.UUID, error) {
	tenantID, _, err := a.ValidateToken(context.Background(), token)
	return tenantID, err
}

func HashAPIToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func GenerateAPIToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return APITokenPrefix + base64.RawURLEncoding.EncodeToString(buf), nil
}

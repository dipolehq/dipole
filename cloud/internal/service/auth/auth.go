// auth.go implements API token authentication: hashing, generation, and resolving a Bearer token to its owning tenant.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/repository"
	"github.com/dipolehq/dipole/cloud/internal/service/identity"
)

const APITokenPrefix = "dipole_pat_"

const minPasswordLength = 8

type APITokenAuthenticator struct {
	tenants   repository.TenantRepository
	apiTokens repository.APITokenRepository
	issuer    *identity.IdentityIssuer
}

func NewAPITokenAuthenticator(tenants repository.TenantRepository) *APITokenAuthenticator {
	return &APITokenAuthenticator{tenants: tenants}
}

func (a *APITokenAuthenticator) SetIssuer(issuer *identity.IdentityIssuer) {
	a.issuer = issuer
}

func (a *APITokenAuthenticator) SetAPITokenRepo(repo repository.APITokenRepository) {
	a.apiTokens = repo
}

func (a *APITokenAuthenticator) ValidateToken(ctx context.Context, token string) (uuid.UUID, string, *uuid.UUID, error) {
	if token == "" {
		return uuid.Nil, "", nil, fmt.Errorf("empty token")
	}

	if tenantID, email, resourceSetID, matched, err := a.tryJWTSession(token); matched {
		return tenantID, email, resourceSetID, err
	}

	hash := HashAPIToken(token)

	if tenantID, email, resourceSetID, matched, err := a.tryScopedAPIToken(ctx, hash); matched {
		return tenantID, email, resourceSetID, err
	}

	return a.tryLegacyTenantToken(ctx, hash)
}

func (a *APITokenAuthenticator) tryJWTSession(token string) (uuid.UUID, string, *uuid.UUID, bool, error) {
	if a.issuer == nil || !isJWT(token) {
		return uuid.Nil, "", nil, false, nil
	}
	tenantID, email, err := a.validateJWTSession(token)
	if err != nil {
		return uuid.Nil, "", nil, false, nil
	}
	return tenantID, email, nil, true, nil
}

func (a *APITokenAuthenticator) tryScopedAPIToken(ctx context.Context, hash string) (uuid.UUID, string, *uuid.UUID, bool, error) {
	if a.apiTokens == nil {
		return uuid.Nil, "", nil, false, nil
	}
	apiToken, err := a.apiTokens.GetByHash(ctx, hash)
	if err != nil || apiToken == nil {
		return uuid.Nil, "", nil, false, nil
	}
	return apiToken.TenantID, "apitoken@scoped", apiToken.ResourceSetID, true, nil
}

func (a *APITokenAuthenticator) tryLegacyTenantToken(ctx context.Context, hash string) (uuid.UUID, string, *uuid.UUID, error) {
	tenant, err := a.tenants.GetByAPITokenHash(ctx, hash)
	if err != nil {
		return uuid.Nil, "", nil, err
	}
	if tenant == nil {
		return uuid.Nil, "", nil, fmt.Errorf("invalid token")
	}
	return tenant.ID, "apitoken@" + tenant.Subdomain, nil, nil
}

func (a *APITokenAuthenticator) validateJWTSession(token string) (uuid.UUID, string, error) {
	if err := a.issuer.VerifyTokenSignature(token); err != nil {
		return uuid.Nil, "", err
	}

	raw, err := decodeJWTPayload(token)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("invalid JWT")
	}

	var claims struct {
		TenantID string `json:"tenant_id"`
		Email    string `json:"email"`
		Exp      int64  `json:"exp"`
	}
	if err := json.Unmarshal(raw, &claims); err != nil {
		return uuid.Nil, "", fmt.Errorf("parse JWT claims: %w", err)
	}

	if claims.Exp > 0 && time.Now().Unix() > claims.Exp {
		return uuid.Nil, "", fmt.Errorf("token expired")
	}

	tenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("invalid tenant_id in token")
	}

	return tenantID, claims.Email, nil
}

func isJWT(token string) bool {
	return strings.Count(token, ".") == 2
}

func (a *APITokenAuthenticator) ValidateAPIToken(ctx context.Context, token string) (uuid.UUID, error) {
	tenantID, _, _, err := a.ValidateToken(ctx, token)
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

/*
Package auth implements admin and API authentication: API token hashing and
validation, email+password login, SSO, MFA, password reset, sign-on policy
enforcement, and tenant signup.

auth.go implements API token authentication: hashing, generation, and
resolving a Bearer token to its owning tenant.
*/
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

	// Try JWT session token first (issued by login endpoint)
	if a.issuer != nil && isJWT(token) {
		tenantID, email, err := a.validateJWTSession(token)
		if err == nil {
			return tenantID, email, nil, nil
		}
	}

	hash := HashAPIToken(token)

	// Try scoped api_tokens table first.
	if a.apiTokens != nil {
		apiToken, err := a.apiTokens.GetByHash(ctx, hash)
		if err == nil && apiToken != nil {
			return apiToken.TenantID, "apitoken@scoped", apiToken.ResourceSetID, nil
		}
	}

	// Fall back to legacy tenant-level token.
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

	// Decode claims
	parts := strings.SplitN(token, ".", 3)
	if len(parts) < 2 {
		return uuid.Nil, "", fmt.Errorf("invalid JWT")
	}
	payload := parts[1]
	if pad := len(payload) % 4; pad != 0 {
		payload += strings.Repeat("=", 4-pad)
	}
	raw, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("decode JWT payload: %w", err)
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

// ValidateAPIToken implements rest.TenantAuthenticator for the Dipole MCP Server.
func (a *APITokenAuthenticator) ValidateAPIToken(token string) (uuid.UUID, error) {
	tenantID, _, _, err := a.ValidateToken(context.Background(), token)
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

/*
token.go implements the OAuth token exchange for MCP authorization.

TokenService validates authorization codes (with PKCE verification),
issues JWT access tokens via the IdentityIssuer, and manages refresh tokens.
Supports both authorization_code and refresh_token grant types.
*/
package mcp

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/service/identity"
)

var (
	ErrInvalidGrant         = errors.New("invalid_grant")
	ErrPKCEFailed           = errors.New("invalid_grant - code_verifier doesn't match code_challenge")
	ErrRedirectURIMismatchT = errors.New("invalid_grant - redirect_uri mismatch")
	ErrClientIDMismatchT    = errors.New("invalid_grant - client_id mismatch")
	ErrRefreshTokenInvalid  = errors.New("invalid_grant - refresh token invalid or expired")
)

type TokenRequest struct {
	GrantType    string
	Code         string
	RedirectURI  string
	ClientID     string
	CodeVerifier string
	RefreshToken string
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

type TokenConfig struct {
	Audience         string
	Subject          string
	Lifetime         time.Duration
	Algorithm        string
	CustomClaims     map[string]any
	RefreshEnabled   bool
	RefreshAbsExpiry time.Duration
}

type RefreshTokenEntry struct {
	Token             string
	ClientID          uuid.UUID
	TenantID          uuid.UUID
	UserClaims        map[string]any
	PolicyID          uuid.UUID
	Scope             string
	AbsoluteExpiresAt time.Time
	Used              bool
}

type TokenService struct {
	codeStore     *AuthCodeStore
	issuer        *identity.IdentityIssuer
	tokenConfig   TokenConfigResolver
	refreshTokens map[string]*RefreshTokenEntry
	mu            sync.RWMutex
}

type TokenConfigResolver interface {
	ResolveTokenConfig(policyID uuid.UUID, tenantID uuid.UUID) (*TokenConfig, error)
}

func NewTokenService(codeStore *AuthCodeStore, issuer *identity.IdentityIssuer, configResolver TokenConfigResolver) *TokenService {
	return &TokenService{
		codeStore:     codeStore,
		issuer:        issuer,
		tokenConfig:   configResolver,
		refreshTokens: make(map[string]*RefreshTokenEntry),
	}
}

func (s *TokenService) Exchange(req *TokenRequest, tenantID uuid.UUID) (*TokenResponse, error) {
	switch req.GrantType {
	case "authorization_code":
		return s.exchangeCode(req, tenantID)
	case "refresh_token":
		return s.exchangeRefresh(req, tenantID)
	default:
		return nil, errors.New("unsupported_grant_type")
	}
}

func (s *TokenService) exchangeCode(req *TokenRequest, tenantID uuid.UUID) (*TokenResponse, error) {
	pending, err := s.codeStore.Consume(req.Code)
	if err != nil {
		return nil, ErrInvalidGrant
	}

	clientUUID, err := uuid.Parse(req.ClientID)
	if err != nil || clientUUID != pending.ClientID {
		return nil, ErrClientIDMismatchT
	}

	if req.RedirectURI != pending.RedirectURI {
		return nil, ErrRedirectURIMismatchT
	}

	if !verifyPKCE(req.CodeVerifier, pending.CodeChallenge) {
		return nil, ErrPKCEFailed
	}

	cfg, err := s.tokenConfig.ResolveTokenConfig(pending.PolicyID, tenantID)
	if err != nil {
		return nil, err
	}

	customClaims := make(map[string]any)
	for k, v := range cfg.CustomClaims {
		customClaims[k] = v
	}
	customClaims["client_id"] = req.ClientID
	customClaims["scope"] = pending.Scope

	token, err := s.issuer.IssueToken(identity.IssuanceRequest{
		Subject:      cfg.Subject,
		Audience:     cfg.Audience,
		Lifetime:     cfg.Lifetime,
		Algorithm:    cfg.Algorithm,
		CustomClaims: customClaims,
		TrustClaims:  pending.UserClaims,
	})
	if err != nil {
		return nil, err
	}

	resp := &TokenResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   int(cfg.Lifetime.Seconds()),
	}

	if cfg.RefreshEnabled {
		rt := s.createRefreshToken(pending.ClientID, tenantID, pending.UserClaims, pending.PolicyID, pending.Scope, cfg.RefreshAbsExpiry)
		resp.RefreshToken = rt
	}

	return resp, nil
}

func (s *TokenService) exchangeRefresh(req *TokenRequest, tenantID uuid.UUID) (*TokenResponse, error) {
	s.mu.Lock()
	entry, ok := s.refreshTokens[req.RefreshToken]
	if !ok || entry.Used || time.Now().UTC().After(entry.AbsoluteExpiresAt) {
		s.mu.Unlock()
		return nil, ErrRefreshTokenInvalid
	}
	entry.Used = true
	s.mu.Unlock()

	clientUUID, err := uuid.Parse(req.ClientID)
	if err != nil || clientUUID != entry.ClientID {
		return nil, ErrClientIDMismatchT
	}

	cfg, err := s.tokenConfig.ResolveTokenConfig(entry.PolicyID, tenantID)
	if err != nil {
		return nil, err
	}

	customClaims := make(map[string]any)
	for k, v := range cfg.CustomClaims {
		customClaims[k] = v
	}
	customClaims["client_id"] = req.ClientID
	customClaims["scope"] = entry.Scope

	token, err := s.issuer.IssueToken(identity.IssuanceRequest{
		Subject:      cfg.Subject,
		Audience:     cfg.Audience,
		Lifetime:     cfg.Lifetime,
		Algorithm:    cfg.Algorithm,
		CustomClaims: customClaims,
		TrustClaims:  entry.UserClaims,
	})
	if err != nil {
		return nil, err
	}

	// New refresh token inherits the SAME absolute expiry (does NOT reset)
	newRT := s.createRefreshTokenWithExpiry(entry.ClientID, tenantID, entry.UserClaims, entry.PolicyID, entry.Scope, entry.AbsoluteExpiresAt)

	return &TokenResponse{
		AccessToken:  token,
		TokenType:    "Bearer",
		ExpiresIn:    int(cfg.Lifetime.Seconds()),
		RefreshToken: newRT,
	}, nil
}

func (s *TokenService) createRefreshToken(clientID, tenantID uuid.UUID, userClaims map[string]any, policyID uuid.UUID, scope string, absExpiry time.Duration) string {
	expiresAt := time.Now().UTC().Add(absExpiry)
	return s.createRefreshTokenWithExpiry(clientID, tenantID, userClaims, policyID, scope, expiresAt)
}

func (s *TokenService) createRefreshTokenWithExpiry(clientID, tenantID uuid.UUID, userClaims map[string]any, policyID uuid.UUID, scope string, expiresAt time.Time) string {
	token := generateCode()

	s.mu.Lock()
	s.refreshTokens[token] = &RefreshTokenEntry{
		Token:             token,
		ClientID:          clientID,
		TenantID:          tenantID,
		UserClaims:        userClaims,
		PolicyID:          policyID,
		Scope:             scope,
		AbsoluteExpiresAt: expiresAt,
		Used:              false,
	}
	s.mu.Unlock()

	return token
}

func verifyPKCE(codeVerifier, codeChallenge string) bool {
	if codeVerifier == "" {
		return false
	}
	hash := sha256.Sum256([]byte(codeVerifier))
	computed := base64.RawURLEncoding.EncodeToString(hash[:])
	return subtle.ConstantTimeCompare([]byte(computed), []byte(codeChallenge)) == 1
}

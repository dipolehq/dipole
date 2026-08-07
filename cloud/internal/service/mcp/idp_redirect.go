/*
idp_redirect.go handles Identity Provider integration for MCP Enforce SSO flows.

IdPService builds redirect URLs using the tenant's configured Identity Provider
(from Administration → Identity Providers) and manages session state that maps
the OAuth state param back to the original MCP authorization request.

The Identity Provider provides OAuth credentials (client_id, secret, discovery URL).
The Trust Provider provides token validation rules (match rules, attestation method).
These are separate concerns following Aembit's architecture.
*/
package mcp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type IdPSession struct {
	OriginalRequest  *AuthorizeRequest
	TenantID         uuid.UUID
	PolicyID         uuid.UUID
	TrustProvider    *domain.TrustProvider
	IdPConfig        *ResolvedIdPConfig
	AccessConditions []domain.AccessCondition
	ClientIP         string
	CreatedAt        time.Time
}

type IdPRedirectResult struct {
	RedirectURL string
	State       string
}

type ResolvedIdPConfig struct {
	ClientID              string
	ClientSecret          string
	AuthorizationEndpoint string
	TokenEndpoint         string
	Scopes                string
}

type IdentityProviderResolver interface {
	ResolveOIDCProvider(ctx context.Context, tenantID uuid.UUID) (*ResolvedIdPConfig, error)
}

type IdPService struct {
	callbackBaseURL string
	idpResolver     IdentityProviderResolver
	sessions        map[string]*IdPSession
	mu              sync.RWMutex
}

func NewIdPService(callbackBaseURL string, idpResolver IdentityProviderResolver) *IdPService {
	return &IdPService{
		callbackBaseURL: callbackBaseURL,
		idpResolver:     idpResolver,
		sessions:        make(map[string]*IdPSession),
	}
}

func (s *IdPService) BuildOIDCRedirect(ctx context.Context, req *AuthorizeRequest, tenantID uuid.UUID, policyID uuid.UUID, tp *domain.TrustProvider, conditions []domain.AccessCondition, clientIP string) (*IdPRedirectResult, error) {
	idpCfg, err := s.idpResolver.ResolveOIDCProvider(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("resolve identity provider: %w", err)
	}

	state := generateState()
	nonce := generateState()

	s.mu.Lock()
	s.sessions[state] = &IdPSession{
		OriginalRequest:  req,
		TenantID:         tenantID,
		PolicyID:         policyID,
		TrustProvider:    tp,
		IdPConfig:        idpCfg,
		AccessConditions: conditions,
		ClientIP:         clientIP,
		CreatedAt:        time.Now().UTC(),
	}
	s.mu.Unlock()

	scopes := idpCfg.Scopes
	if scopes == "" {
		scopes = "openid profile email"
	}

	redirectURL := idpCfg.AuthorizationEndpoint +
		"?response_type=code" +
		"&client_id=" + idpCfg.ClientID +
		"&redirect_uri=" + s.callbackBaseURL + "/callback/oidc" +
		"&scope=" + strings.ReplaceAll(scopes, " ", "+") +
		"&state=" + state +
		"&nonce=" + nonce

	return &IdPRedirectResult{
		RedirectURL: redirectURL,
		State:       state,
	}, nil
}

func (s *IdPService) BuildSAMLRedirect(ctx context.Context, req *AuthorizeRequest, tenantID uuid.UUID, policyID uuid.UUID, tp *domain.TrustProvider, conditions []domain.AccessCondition, clientIP string) (*IdPRedirectResult, error) {
	var samlCfg struct {
		SSOUrl string `json:"sso_url"`
	}
	if err := json.Unmarshal(tp.Config, &samlCfg); err != nil {
		return nil, err
	}

	state := generateState()

	s.mu.Lock()
	s.sessions[state] = &IdPSession{
		OriginalRequest:  req,
		TenantID:         tenantID,
		PolicyID:         policyID,
		TrustProvider:    tp,
		AccessConditions: conditions,
		ClientIP:         clientIP,
		CreatedAt:        time.Now().UTC(),
	}
	s.mu.Unlock()

	redirectURL := samlCfg.SSOUrl + "?RelayState=" + state

	return &IdPRedirectResult{
		RedirectURL: redirectURL,
		State:       state,
	}, nil
}

func (s *IdPService) GetSession(state string) (*IdPSession, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[state]
	if !ok {
		return nil, false
	}
	if time.Now().UTC().After(session.CreatedAt.Add(5 * time.Minute)) {
		return nil, false
	}
	return session, true
}

func (s *IdPService) ConsumeSession(state string) (*IdPSession, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[state]
	if !ok {
		return nil, false
	}
	if time.Now().UTC().After(session.CreatedAt.Add(5 * time.Minute)) {
		delete(s.sessions, state)
		return nil, false
	}
	delete(s.sessions, state)
	return session, true
}

func generateState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

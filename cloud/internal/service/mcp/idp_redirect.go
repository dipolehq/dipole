// idp_redirect.go implements IdPService, which builds Identity Provider redirect URLs for MCP Enforce SSO flows and owns the pending-session map.
package mcp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/platform"
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
	Nonce            string
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
	ResolveSAMLProvider(ctx context.Context, tenantID uuid.UUID) (*ResolvedSAMLConfig, error)
}

type IdPService struct {
	idpResolver IdentityProviderResolver
	sessions    map[string]*IdPSession
	mu          sync.RWMutex
}

func NewIdPService(idpResolver IdentityProviderResolver) *IdPService {
	return &IdPService{
		idpResolver: idpResolver,
		sessions:    make(map[string]*IdPSession),
	}
}

func (s *IdPService) BuildOIDCRedirect(ctx context.Context, req *AuthorizeRequest, tenantID uuid.UUID, policyID uuid.UUID, tp *domain.TrustProvider, conditions []domain.AccessCondition, clientIP string) (*IdPRedirectResult, error) {
	if req.ServerBaseURL == "" {
		return nil, fmt.Errorf("cannot build IdP redirect: server base URL not resolved from request")
	}

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
		Nonce:            nonce,
	}
	s.mu.Unlock()

	scopes := idpCfg.Scopes
	if scopes == "" {
		scopes = "openid profile email"
	}

	params := url.Values{
		"response_type": {"code"},
		"client_id":     {idpCfg.ClientID},
		"redirect_uri":  {req.ServerBaseURL + platform.OIDCCallbackPath},
		"scope":         {scopes},
		"state":         {state},
		"nonce":         {nonce},
	}

	separator := "?"
	if strings.Contains(idpCfg.AuthorizationEndpoint, "?") {
		separator = "&"
	}

	return &IdPRedirectResult{
		RedirectURL: idpCfg.AuthorizationEndpoint + separator + params.Encode(),
		State:       state,
	}, nil
}

func (s *IdPService) BuildSAMLRedirect(ctx context.Context, req *AuthorizeRequest, tenantID uuid.UUID, policyID uuid.UUID, tp *domain.TrustProvider, conditions []domain.AccessCondition, clientIP string) (*IdPRedirectResult, error) {
	samlCfg, err := s.idpResolver.ResolveSAMLProvider(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("resolve SAML identity provider: %w", err)
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

	separator := "?"
	if strings.Contains(samlCfg.SSOURL, "?") {
		separator = "&"
	}
	params := url.Values{"RelayState": {state}}

	return &IdPRedirectResult{
		RedirectURL: samlCfg.SSOURL + separator + params.Encode(),
		State:       state,
	}, nil
}

func (s *IdPService) SweepExpired(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for state, session := range s.sessions {
		if now.After(session.CreatedAt.Add(5 * time.Minute)) {
			delete(s.sessions, state)
		}
	}
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

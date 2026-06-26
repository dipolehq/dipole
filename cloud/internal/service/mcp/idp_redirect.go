/*
idp_redirect.go handles Identity Provider integration for MCP Enforce SSO flows.

IdPService builds redirect URLs for OIDC/SAML identity providers and manages
session state that maps the OAuth state param back to the original MCP
authorization request. After user authenticates at the IdP, the callback
handler uses this state to resume the authorization flow.
*/
package mcp

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
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
	AccessConditions []domain.AccessCondition
	ClientIP         string
	CreatedAt        time.Time
}

type IdPRedirectResult struct {
	RedirectURL string
	State       string
}

type OIDCIdPConfig struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	ClientID              string `json:"client_id"`
	ClientSecret          string `json:"client_secret"`
}

type IdPService struct {
	callbackBaseURL string
	sessions        map[string]*IdPSession
	mu              sync.RWMutex
}

func NewIdPService(callbackBaseURL string) *IdPService {
	return &IdPService{
		callbackBaseURL: callbackBaseURL,
		sessions:        make(map[string]*IdPSession),
	}
}

func (s *IdPService) BuildOIDCRedirect(req *AuthorizeRequest, tenantID uuid.UUID, policyID uuid.UUID, tp *domain.TrustProvider, conditions []domain.AccessCondition, clientIP string) (*IdPRedirectResult, error) {
	var idpCfg OIDCIdPConfig
	if err := json.Unmarshal(tp.Config, &idpCfg); err != nil {
		return nil, err
	}

	state := generateState()
	nonce := generateState()

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

	redirectURL := idpCfg.AuthorizationEndpoint +
		"?response_type=code" +
		"&client_id=" + idpCfg.ClientID +
		"&redirect_uri=" + s.callbackBaseURL + "/callback/oidc" +
		"&scope=openid+profile+email" +
		"&state=" + state +
		"&nonce=" + nonce

	return &IdPRedirectResult{
		RedirectURL: redirectURL,
		State:       state,
	}, nil
}

func (s *IdPService) BuildSAMLRedirect(req *AuthorizeRequest, tenantID uuid.UUID, policyID uuid.UUID, tp *domain.TrustProvider, conditions []domain.AccessCondition, clientIP string) (*IdPRedirectResult, error) {
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

	// SAML AuthnRequest — simplified redirect binding
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

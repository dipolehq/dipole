/*
idp_callback.go processes Identity Provider callbacks after user authentication.

OIDCCallbackHandler exchanges the IdP authorization code for an id_token,
validates it using the Trust Provider (OIDC validator), extracts user claims,
and resumes the MCP authorization flow by issuing an authorization code.

SAMLCallbackHandler validates the SAML assertion and does the same.
*/
package mcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/google/uuid"
)

type TrustValidator interface {
	Validate(ctx context.Context, provider *domain.TrustProvider, evidence *domain.ClientEvidence) (*domain.TrustProviderResult, error)
}

type IdPCallbackService struct {
	idpService *IdPService
	codeStore  *AuthCodeStore
	trust      TrustValidator
	conditions ConditionChecker
	httpClient *http.Client
}

func NewIdPCallbackService(idpService *IdPService, codeStore *AuthCodeStore, trust TrustValidator, conditions ConditionChecker) *IdPCallbackService {
	return &IdPCallbackService{
		idpService: idpService,
		codeStore:  codeStore,
		trust:      trust,
		conditions: conditions,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

type CallbackResult struct {
	RedirectURL string
	Error       string
}

func (s *IdPCallbackService) HandleOIDCCallback(ctx context.Context, code, state string) (*CallbackResult, error) {
	session, ok := s.idpService.ConsumeSession(state)
	if !ok {
		return nil, fmt.Errorf("invalid or expired session state")
	}

	var idpCfg OIDCIdPConfig
	if err := json.Unmarshal(session.TrustProvider.Config, &idpCfg); err != nil {
		return nil, err
	}

	idToken, err := s.exchangeCodeForToken(ctx, code, idpCfg, s.idpService.callbackBaseURL+"/callback/oidc")
	if err != nil {
		return &CallbackResult{
			RedirectURL: buildErrorRedirect(session.OriginalRequest.RedirectURI, "access_denied", "IdP token exchange failed", session.OriginalRequest.State),
		}, nil
	}

	evidence := &domain.ClientEvidence{
		SourceIP: session.ClientIP,
		Environment: map[string]string{
			"OIDC_ID_TOKEN": idToken,
		},
	}

	trustResult, err := s.trust.Validate(ctx, session.TrustProvider, evidence)
	if err != nil || !trustResult.Attested {
		reason := "Trust Provider validation failed"
		if trustResult != nil && trustResult.Reason != "" {
			reason = trustResult.Reason
		}
		return &CallbackResult{
			RedirectURL: buildErrorRedirect(session.OriginalRequest.RedirectURI, "access_denied", reason, session.OriginalRequest.State),
		}, nil
	}

	userClaims := extractUserClaims(idToken)

	return s.issueCodeAfterAuth(ctx, session, userClaims)
}

func (s *IdPCallbackService) HandleSAMLCallback(ctx context.Context, samlResponse, relayState string) (*CallbackResult, error) {
	session, ok := s.idpService.ConsumeSession(relayState)
	if !ok {
		return nil, fmt.Errorf("invalid or expired session state")
	}

	evidence := &domain.ClientEvidence{
		SourceIP: session.ClientIP,
		Environment: map[string]string{
			"SAML_RESPONSE": samlResponse,
		},
	}

	trustResult, err := s.trust.Validate(ctx, session.TrustProvider, evidence)
	if err != nil || !trustResult.Attested {
		reason := "SAML validation failed"
		if trustResult != nil && trustResult.Reason != "" {
			reason = trustResult.Reason
		}
		return &CallbackResult{
			RedirectURL: buildErrorRedirect(session.OriginalRequest.RedirectURI, "access_denied", reason, session.OriginalRequest.State),
		}, nil
	}

	userClaims := extractSAMLClaims(samlResponse)

	return s.issueCodeAfterAuth(ctx, session, userClaims)
}

func (s *IdPCallbackService) issueCodeAfterAuth(ctx context.Context, session *IdPSession, userClaims map[string]any) (*CallbackResult, error) {
	if len(session.AccessConditions) > 0 && s.conditions != nil {
		evidence := &domain.ClientEvidence{SourceIP: session.ClientIP}
		passed, reason, err := s.conditions.EvaluateAll(ctx, session.AccessConditions, evidence)
		if err != nil {
			return &CallbackResult{
				RedirectURL: buildErrorRedirect(session.OriginalRequest.RedirectURI, "access_denied", "condition evaluation error", session.OriginalRequest.State),
			}, nil
		}
		if !passed {
			return &CallbackResult{
				RedirectURL: buildErrorRedirect(session.OriginalRequest.RedirectURI, "access_denied", reason, session.OriginalRequest.State),
			}, nil
		}
	}

	scope := session.OriginalRequest.Scope
	if scope == "" {
		scope = "openid"
	}

	code := s.codeStore.Create(
		mustParseUUID(session.OriginalRequest.ClientID),
		session.TenantID,
		session.OriginalRequest.RedirectURI,
		session.OriginalRequest.CodeChallenge,
		scope,
		userClaims,
		session.PolicyID,
	)

	redirectURL := buildAuthCodeRedirect(session.OriginalRequest.RedirectURI, code, session.OriginalRequest.State)
	return &CallbackResult{RedirectURL: redirectURL}, nil
}

func (s *IdPCallbackService) exchangeCodeForToken(ctx context.Context, code string, cfg OIDCIdPConfig, redirectURI string) (string, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"client_id":     {cfg.ClientID},
		"client_secret": {cfg.ClientSecret},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.TokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token exchange failed: %s", string(body))
	}

	var tokenResp struct {
		IDToken     string `json:"id_token"`
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", err
	}

	if tokenResp.IDToken != "" {
		return tokenResp.IDToken, nil
	}
	return tokenResp.AccessToken, nil
}

func extractUserClaims(idToken string) map[string]any {
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return nil
	}
	// Decode payload (middle part) — no signature verification needed here
	// (already validated by Trust Provider)
	payload, err := decodeBase64URL(parts[1])
	if err != nil {
		return nil
	}
	var claims map[string]any
	json.Unmarshal(payload, &claims)
	return claims
}

func extractSAMLClaims(samlResponse string) map[string]any {
	// SAML claims are extracted during Trust Provider validation
	// Return minimal claims — the Trust Provider already validated the assertion
	return map[string]any{
		"auth_method": "saml",
	}
}

func decodeBase64URL(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}

func buildErrorRedirect(redirectURI, errCode, description, state string) string {
	u, _ := url.Parse(redirectURI)
	q := u.Query()
	q.Set("error", errCode)
	q.Set("error_description", description)
	if state != "" {
		q.Set("state", state)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func mustParseUUID(s string) uuid.UUID {
	id, _ := uuid.Parse(s)
	return id
}

// consent_callback.go implements ConsentCallbackService, which handles the OAuth callback from downstream MCP servers and stores the resulting per-user credentials.
package mcp_gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ConsentCallbackService struct {
	tokenStore     UserTokenStore
	configResolver ConsentConfigResolver
	httpClient     *http.Client
}

func NewConsentCallbackService(tokenStore UserTokenStore, configResolver ConsentConfigResolver) *ConsentCallbackService {
	return &ConsentCallbackService{
		tokenStore:     tokenStore,
		configResolver: configResolver,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

var ErrConsentTenantMismatch = errors.New("consent state tenant does not match the request hostname")

func (s *ConsentCallbackService) HandleCallback(ctx context.Context, code string, state string, scope TenantScope) error {
	if !scope.Valid() {
		return ErrInvalidTenantScope
	}

	tenantID, userID, serverWorkloadID, err := parseConsentState(state)
	if err != nil {
		return fmt.Errorf("parse consent state: %w", err)
	}

	if tenantID != scope.TenantID {
		return ErrConsentTenantMismatch
	}

	cfg, err := s.configResolver.GetConsentConfig(ctx, tenantID, serverWorkloadID, scope.GatewayURL)
	if err != nil {
		return fmt.Errorf("get consent config: %w", err)
	}

	accessToken, refreshToken, expiresIn, err := s.exchangeCode(ctx, code, cfg)
	if err != nil {
		return fmt.Errorf("exchange authorization code: %w", err)
	}

	expiresAt := time.Now().UTC().Add(time.Duration(expiresIn) * time.Second)
	entry := &UserTokenEntry{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
	}

	return s.tokenStore.Store(ctx, tenantID, userID, serverWorkloadID, entry)
}

func (s *ConsentCallbackService) exchangeCode(ctx context.Context, code string, cfg *ConsentConfig) (accessToken, refreshToken string, expiresIn int, err error) {
	form := url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {code},
		"redirect_uri": {cfg.GatewayCallbackURL},
	}
	if cfg.ClientID != "" {
		form.Set("client_id", cfg.ClientID)
	}
	if cfg.ClientSecret != "" {
		form.Set("client_secret", cfg.ClientSecret)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.TokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", "", 0, fmt.Errorf("create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", "", 0, fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", 0, fmt.Errorf("read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", "", 0, fmt.Errorf("token endpoint returned status %d", resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", "", 0, fmt.Errorf("parse token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return "", "", 0, fmt.Errorf("token response missing access_token")
	}

	expiresIn = tokenResp.ExpiresIn
	if expiresIn == 0 {
		expiresIn = 3600
	}

	return tokenResp.AccessToken, tokenResp.RefreshToken, expiresIn, nil
}

func parseConsentState(state string) (tenantID uuid.UUID, userID string, serverWorkloadID uuid.UUID, err error) {
	parts := strings.SplitN(state, ":", 3)
	if len(parts) != 3 {
		return uuid.Nil, "", uuid.Nil, fmt.Errorf("invalid state format: expected tenant:user:server")
	}

	tenantID, err = uuid.Parse(parts[0])
	if err != nil {
		return uuid.Nil, "", uuid.Nil, fmt.Errorf("invalid tenant ID in state: %w", err)
	}

	userID = parts[1]
	if userID == "" {
		return uuid.Nil, "", uuid.Nil, fmt.Errorf("empty user ID in state")
	}

	serverWorkloadID, err = uuid.Parse(parts[2])
	if err != nil {
		return uuid.Nil, "", uuid.Nil, fmt.Errorf("invalid server workload ID in state: %w", err)
	}

	return tenantID, userID, serverWorkloadID, nil
}

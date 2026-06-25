/*
oauth_auth_code.go handles OAuth 2.0 Authorization Code credential retrieval.
The admin pre-authorizes the flow (stores refresh_token in encrypted config).
On each request, returns the cached access_token if valid, or exchanges the
refresh_token for a new one. Returns error if refresh_token is expired.
*/
package credentials

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type OAuthAuthCodeHandler struct {
	httpClient *http.Client
}

func NewOAuthAuthCodeHandler() *OAuthAuthCodeHandler {
	return &OAuthAuthCodeHandler{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

type oauthAuthCodeConfig struct {
	TokenURL     string `json:"token_url"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Scopes       string `json:"scopes"`
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
	ExpiresAt    string `json:"expires_at"`
}

func (h *OAuthAuthCodeHandler) Retrieve(ctx context.Context, configJSON []byte, _ *domain.ClientEvidence, _ map[string]interface{}) (*domain.Credential, error) {
	var cfg oauthAuthCodeConfig
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		return nil, fmt.Errorf("parse oauth auth code config: %w", err)
	}

	if cfg.AccessToken != "" && cfg.ExpiresAt != "" {
		expiresAt, err := time.Parse(time.RFC3339, cfg.ExpiresAt)
		if err == nil && time.Now().Before(expiresAt) {
			return &domain.Credential{Type: "bearer", Value: cfg.AccessToken}, nil
		}
	}

	if cfg.RefreshToken == "" {
		return nil, fmt.Errorf("no refresh token available — admin must re-authorize")
	}

	token, err := h.exchangeRefreshToken(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("refresh token exchange failed: %w", err)
	}

	return &domain.Credential{Type: "bearer", Value: token}, nil
}

func (h *OAuthAuthCodeHandler) exchangeRefreshToken(ctx context.Context, cfg oauthAuthCodeConfig) (string, error) {
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {cfg.ClientID},
		"client_secret": {cfg.ClientSecret},
		"refresh_token": {cfg.RefreshToken},
	}
	if cfg.Scopes != "" {
		form.Set("scope", cfg.Scopes)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("parse token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("token response missing access_token")
	}

	return tokenResp.AccessToken, nil
}

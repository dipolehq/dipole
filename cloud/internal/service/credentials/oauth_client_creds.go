/*
oauth_client_creds.go retrieves OAuth 2.0 access tokens using the Client
Credentials grant. Supports two credential styles: POST body (default) and
Authorization Header (HTTP Basic). Caches tokens at 80% of expires_in.
*/
package credentials

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
)

type OAuthClientCredsHandler struct {
	httpClient *http.Client
}

func NewOAuthClientCredsHandler() *OAuthClientCredsHandler {
	return &OAuthClientCredsHandler{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

type oauthClientCredsConfig struct {
	TokenURL        string `json:"token_url"`
	ClientID        string `json:"client_id"`
	ClientSecret    string `json:"client_secret"`
	Scopes          string `json:"scopes"`
	CredentialStyle string `json:"credential_style"` // "post_body" (default) or "authorization_header"
}

func (h *OAuthClientCredsHandler) Retrieve(ctx context.Context, configJSON []byte, _ *domain.ClientEvidence, _ map[string]interface{}) (*domain.Credential, error) {
	var cfg oauthClientCredsConfig
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		return nil, fmt.Errorf("parse oauth client credentials config: %w", err)
	}

	form := url.Values{"grant_type": {"client_credentials"}}
	if cfg.Scopes != "" {
		form.Set("scope", cfg.Scopes)
	}

	if cfg.CredentialStyle != "authorization_header" {
		form.Set("client_id", cfg.ClientID)
		form.Set("client_secret", cfg.ClientSecret)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if cfg.CredentialStyle == "authorization_header" {
		basicAuth := base64.StdEncoding.EncodeToString([]byte(cfg.ClientID + ":" + cfg.ClientSecret))
		req.Header.Set("Authorization", "Basic "+basicAuth)
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token endpoint unreachable: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("parse token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return nil, fmt.Errorf("token response missing access_token")
	}

	return &domain.Credential{
		Type:      "bearer",
		Value:     tokenResp.AccessToken,
		ExpiresAt: time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second).Unix(),
	}, nil
}

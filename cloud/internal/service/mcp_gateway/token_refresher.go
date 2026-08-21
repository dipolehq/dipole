// token_refresher.go implements OAuthTokenRefresher, which refreshes expired per-user OAuth tokens against a downstream MCP server's token endpoint.
package mcp_gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

type OAuthTokenRefresher struct {
	httpClient     *http.Client
	configResolver ConsentConfigResolver
}

func NewOAuthTokenRefresher(configResolver ConsentConfigResolver) *OAuthTokenRefresher {
	return &OAuthTokenRefresher{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		configResolver: configResolver,
	}
}

func (r *OAuthTokenRefresher) Refresh(ctx context.Context, scope TenantScope, refreshToken string, serverWorkloadID uuid.UUID) (*UserTokenEntry, error) {
	cfg, err := r.configResolver.GetConsentConfig(ctx, scope.TenantID, serverWorkloadID, scope.GatewayURL)
	if err != nil {
		return nil, fmt.Errorf("resolve token endpoint: %w", err)
	}
	if cfg.TokenEndpoint == "" {
		return nil, fmt.Errorf("no token endpoint configured for server %s", serverWorkloadID)
	}

	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	}
	if cfg.ClientID != "" {
		form.Set("client_id", cfg.ClientID)
	}
	if cfg.ClientSecret != "" {
		form.Set("client_secret", cfg.ClientSecret)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.TokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("refresh token request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read refresh response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("refresh failed with status %d", resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("parse refresh response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return nil, fmt.Errorf("refresh response missing access_token")
	}

	expiresAt := time.Now().UTC().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	if tokenResp.ExpiresIn == 0 {
		expiresAt = time.Now().UTC().Add(1 * time.Hour)
	}

	newRefreshToken := tokenResp.RefreshToken
	if newRefreshToken == "" {
		newRefreshToken = refreshToken
	}

	return &UserTokenEntry{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: newRefreshToken,
		ExpiresAt:    expiresAt,
	}, nil
}

/*
token_refresher.go implements OAuth token refresh for per-user MCP credentials.

OAuthTokenRefresher calls downstream MCP servers' OAuth token endpoints with
grant_type=refresh_token to obtain new access tokens when existing ones expire.
*/
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
	tokenEndpoints map[uuid.UUID]string
}

func NewOAuthTokenRefresher(tokenEndpoints map[uuid.UUID]string) *OAuthTokenRefresher {
	return &OAuthTokenRefresher{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		tokenEndpoints: tokenEndpoints,
	}
}

func (r *OAuthTokenRefresher) Refresh(ctx context.Context, refreshToken string, serverWorkloadID uuid.UUID) (*UserTokenEntry, error) {
	endpoint, ok := r.tokenEndpoints[serverWorkloadID]
	if !ok || endpoint == "" {
		return nil, fmt.Errorf("no token endpoint configured for server %s", serverWorkloadID)
	}

	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
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

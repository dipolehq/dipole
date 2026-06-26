/*
user_consent.go handles the per-user OAuth consent flow for MCP servers.

When a user accesses an MCP server for the first time (no stored credential),
the Gateway returns an error with a consent URL. The user visits this URL in
their browser, completes the OAuth flow with the MCP server, and we store the
resulting tokens. Subsequent requests use the stored token.
*/
package mcp_gateway

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"
)

type ConsentConfig struct {
	AuthorizationEndpoint string
	ClientID              string
	Scopes                string
	GatewayCallbackURL    string
}

type ConsentResult struct {
	ConsentURL string
}

type ConsentConfigResolver interface {
	GetConsentConfig(ctx context.Context, tenantID uuid.UUID, serverWorkloadID uuid.UUID) (*ConsentConfig, error)
}

type UserConsentService struct {
	tokenStore     UserTokenStore
	configResolver ConsentConfigResolver
	gatewayURL     string
}

func NewUserConsentService(tokenStore UserTokenStore, configResolver ConsentConfigResolver, gatewayURL string) *UserConsentService {
	return &UserConsentService{
		tokenStore:     tokenStore,
		configResolver: configResolver,
		gatewayURL:     gatewayURL,
	}
}

func (s *UserConsentService) BuildConsentURL(ctx context.Context, tenantID uuid.UUID, userID string, serverWorkloadID uuid.UUID) (*ConsentResult, error) {
	cfg, err := s.configResolver.GetConsentConfig(ctx, tenantID, serverWorkloadID)
	if err != nil {
		return nil, err
	}

	state := fmt.Sprintf("%s:%s:%s", tenantID.String(), userID, serverWorkloadID.String())

	consentURL := cfg.AuthorizationEndpoint +
		"?response_type=code" +
		"&client_id=" + url.QueryEscape(cfg.ClientID) +
		"&redirect_uri=" + url.QueryEscape(cfg.GatewayCallbackURL) +
		"&scope=" + url.QueryEscape(cfg.Scopes) +
		"&state=" + url.QueryEscape(state)

	return &ConsentResult{ConsentURL: consentURL}, nil
}

func (s *UserConsentService) StoreToken(ctx context.Context, tenantID uuid.UUID, userID string, serverWorkloadID uuid.UUID, accessToken, refreshToken string, expiresIn int) error {
	entry := &UserTokenEntry{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().UTC().Add(time.Duration(expiresIn) * time.Second),
	}
	return s.tokenStore.Store(ctx, tenantID, userID, serverWorkloadID, entry)
}

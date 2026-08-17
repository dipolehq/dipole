// user_consent.go implements UserConsentService, which builds per-user OAuth consent URLs for first-time access to an MCP server.
package mcp_gateway

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"
)

type ConsentConfig struct {
	AuthorizationEndpoint string
	TokenEndpoint         string
	ClientID              string
	ClientSecret          string
	Scopes                string
	GatewayCallbackURL    string
}

type ConsentResult struct {
	ConsentURL string
}

const ConsentCallbackPath = "/mcp/consent/callback"

var ErrNoGatewayURL = errors.New("no gateway URL supplied for this request")

type ConsentConfigResolver interface {
	GetConsentConfig(ctx context.Context, tenantID uuid.UUID, serverWorkloadID uuid.UUID, gatewayURL string) (*ConsentConfig, error)
}

type UserConsentService struct {
	tokenStore     UserTokenStore
	configResolver ConsentConfigResolver
}

func NewUserConsentService(tokenStore UserTokenStore, configResolver ConsentConfigResolver) *UserConsentService {
	return &UserConsentService{
		tokenStore:     tokenStore,
		configResolver: configResolver,
	}
}

func (s *UserConsentService) BuildConsentURL(ctx context.Context, scope TenantScope, userID string, serverWorkloadID uuid.UUID) (*ConsentResult, error) {
	if !scope.Valid() {
		return nil, ErrInvalidTenantScope
	}
	tenantID := scope.TenantID

	cfg, err := s.configResolver.GetConsentConfig(ctx, tenantID, serverWorkloadID, scope.GatewayURL)
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

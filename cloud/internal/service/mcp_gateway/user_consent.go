// user_consent.go implements UserConsentService, which builds per-user OAuth consent URLs for first-time access to an MCP server.
package mcp_gateway

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/google/uuid"
)

type ConsentConfig struct {
	AuthorizationEndpoint string
	TokenEndpoint         string
	ClientID              string
	ClientSecret          string
	Scopes                string
	GatewayCallbackURL    string
	// CredentialProviderID is the Credential Provider this OAuth config
	// was resolved from, recorded alongside the token it's used to fetch
	// (see UserTokenStore.Store).
	CredentialProviderID uuid.UUID
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
	configResolver ConsentConfigResolver
}

func NewUserConsentService(configResolver ConsentConfigResolver) *UserConsentService {
	return &UserConsentService{
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

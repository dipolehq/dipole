// admin_sso.go implements OIDC SSO login for admin users, including IdP callback handling and auto-provisioning.
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
)

type IdentityProviderConfig struct {
	ID              uuid.UUID
	TenantID        uuid.UUID
	Name            string
	Type            string
	Config          json.RawMessage
	AutoCreateUsers bool
	DefaultRoleID   *uuid.UUID
	CreatedAt       time.Time
}

type OIDCConfig struct {
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	DiscoveryURL string `json:"baseUrl"`
	Scopes       string `json:"scopes"`
}

type SSOResult struct {
	UserID  uuid.UUID
	Email   string
	Name    string
	Created bool
}

type OIDCClient interface {
	FetchDiscovery(ctx context.Context, discoveryURL string) (*OIDCDiscoveryDoc, error)
	ExchangeCode(ctx context.Context, tokenEndpoint string, clientID, clientSecret, code, redirectURI string) (*OIDCTokenResponse, error)
}

type OIDCDiscoveryDoc struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	JWKSURI               string `json:"jwks_uri"`
	Issuer                string `json:"issuer"`
}

type OIDCTokenResponse struct {
	IDToken     string `json:"id_token"`
	AccessToken string `json:"access_token"`
}

type AdminSSOService struct {
	userRepo   repository.UserRepository
	oidcClient OIDCClient
}

func NewAdminSSOService(userRepo repository.UserRepository, oidcClient OIDCClient) *AdminSSOService {
	return &AdminSSOService{
		userRepo:   userRepo,
		oidcClient: oidcClient,
	}
}

func (s *AdminSSOService) BuildOIDCLoginURL(ctx context.Context, idp *IdentityProviderConfig, callbackURL, state string) (string, error) {
	var cfg OIDCConfig
	if err := json.Unmarshal(idp.Config, &cfg); err != nil {
		return "", err
	}

	discovery, err := s.oidcClient.FetchDiscovery(ctx, cfg.DiscoveryURL)
	if err != nil {
		return "", fmt.Errorf("OIDC discovery failed: %w", err)
	}

	scopes := cfg.Scopes
	if scopes == "" {
		scopes = "openid profile email"
	}

	return discovery.AuthorizationEndpoint +
		"?response_type=code" +
		"&client_id=" + cfg.ClientID +
		"&redirect_uri=" + callbackURL +
		"&scope=" + strings.ReplaceAll(scopes, " ", "+") +
		"&state=" + state, nil
}

func (s *AdminSSOService) HandleOIDCCallback(ctx context.Context, idp *IdentityProviderConfig, code, callbackURL string) (*SSOResult, error) {
	var cfg OIDCConfig
	if err := json.Unmarshal(idp.Config, &cfg); err != nil {
		return nil, err
	}

	discovery, err := s.oidcClient.FetchDiscovery(ctx, cfg.DiscoveryURL)
	if err != nil {
		return nil, fmt.Errorf("OIDC discovery failed: %w", err)
	}

	tokenResp, err := s.oidcClient.ExchangeCode(ctx, discovery.TokenEndpoint, cfg.ClientID, cfg.ClientSecret, code, callbackURL)
	if err != nil {
		return nil, fmt.Errorf("OIDC code exchange failed: %w", err)
	}

	claims, err := decodeIDToken(tokenResp.IDToken)
	if err != nil {
		return nil, fmt.Errorf("invalid id_token: %w", err)
	}

	email, _ := claims["email"].(string)
	name, _ := claims["name"].(string)
	if email == "" {
		return nil, fmt.Errorf("id_token missing email claim")
	}

	user, err := s.userRepo.GetByEmail(ctx, idp.TenantID, email)
	if err != nil || user == nil {
		if !idp.AutoCreateUsers {
			return nil, fmt.Errorf("user not found and auto-creation disabled")
		}
		user = &domain.User{
			ID:       uuid.New(),
			TenantID: idp.TenantID,
			Email:    email,
			Name:     name,
			RoleID:   idp.DefaultRoleID,
		}
		if err := s.userRepo.Create(ctx, idp.TenantID, user); err != nil {
			return nil, fmt.Errorf("auto-create user failed: %w", err)
		}
		return &SSOResult{UserID: user.ID, Email: email, Name: name, Created: true}, nil
	}

	return &SSOResult{UserID: user.ID, Email: user.Email, Name: user.Name, Created: false}, nil
}

func decodeIDToken(token string) (map[string]any, error) {
	decoded, err := decodeJWTPayload(token)
	if err != nil {
		return nil, err
	}

	var claims map[string]any
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return nil, err
	}
	return claims, nil
}

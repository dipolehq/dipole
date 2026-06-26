/*
admin_sso.go implements Identity Provider management and SSO login flows.

AdminSSOService handles OIDC and SAML SSO for admin users. On first login via
SSO, users are auto-created with a configured default role (if enabled).
Super Admin can always bypass SSO via native login.
*/
package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	DiscoveryURL string `json:"discovery_url"`
	Scopes       string `json:"scopes"`
}

type SAMLConfig struct {
	MetadataURL string `json:"metadata_url"`
	EntityID    string `json:"entity_id"`
	ACSURL      string `json:"acs_url"`
}

type SSOResult struct {
	UserID  uuid.UUID
	Email   string
	Name    string
	Created bool
}

type AdminSSOService struct {
	userRepo       repository.UserRepository
	httpClient     *http.Client
	discoveryCache map[string]*cachedDiscovery
}

type cachedDiscovery struct {
	doc       *oidcDiscovery
	fetchedAt time.Time
}

func NewAdminSSOService(userRepo repository.UserRepository) *AdminSSOService {
	return &AdminSSOService{
		userRepo:       userRepo,
		httpClient:     &http.Client{Timeout: 10 * time.Second},
		discoveryCache: make(map[string]*cachedDiscovery),
	}
}

func (s *AdminSSOService) BuildOIDCLoginURL(idp *IdentityProviderConfig, callbackURL, state string) (string, error) {
	var cfg OIDCConfig
	if err := json.Unmarshal(idp.Config, &cfg); err != nil {
		return "", err
	}

	discovery, err := s.fetchOIDCDiscovery(cfg.DiscoveryURL)
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

	discovery, err := s.fetchOIDCDiscovery(cfg.DiscoveryURL)
	if err != nil {
		return nil, fmt.Errorf("OIDC discovery failed: %w", err)
	}

	tokenResp, err := s.exchangeOIDCCode(ctx, cfg, discovery.TokenEndpoint, code, callbackURL)
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

type oidcTokenResponse struct {
	IDToken     string `json:"id_token"`
	AccessToken string `json:"access_token"`
}

func (s *AdminSSOService) exchangeOIDCCode(ctx context.Context, cfg OIDCConfig, tokenEndpoint, code, redirectURI string) (*oidcTokenResponse, error) {
	data := fmt.Sprintf("grant_type=authorization_code&code=%s&redirect_uri=%s&client_id=%s&client_secret=%s",
		code, redirectURI, cfg.ClientID, cfg.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenEndpoint, strings.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp oidcTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, err
	}
	return &tokenResp, nil
}

type oidcDiscovery struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	JWKSURI               string `json:"jwks_uri"`
	Issuer                string `json:"issuer"`
}

func (s *AdminSSOService) fetchOIDCDiscovery(discoveryURL string) (*oidcDiscovery, error) {
	if cached, ok := s.discoveryCache[discoveryURL]; ok {
		if time.Since(cached.fetchedAt) < 1*time.Hour {
			return cached.doc, nil
		}
	}

	url := strings.TrimSuffix(discoveryURL, "/") + "/.well-known/openid-configuration"
	resp, err := s.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discovery endpoint returned %d", resp.StatusCode)
	}

	var doc oidcDiscovery
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, err
	}

	s.discoveryCache[discoveryURL] = &cachedDiscovery{doc: &doc, fetchedAt: time.Now()}
	return &doc, nil
}

func decodeIDToken(token string) (map[string]any, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format")
	}

	decoded, err := base64RawURLDecodeSSO(parts[1])
	if err != nil {
		return nil, err
	}

	var claims map[string]any
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return nil, err
	}
	return claims, nil
}

func base64RawURLDecodeSSO(s string) ([]byte, error) {
	// base64url without padding
	return base64.RawURLEncoding.DecodeString(s)
}

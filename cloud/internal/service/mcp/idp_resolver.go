/*
idp_resolver.go resolves the tenant's Identity Provider configuration for
MCP SSO flows. Fetches the OIDC discovery document to dynamically resolve
authorization and token endpoints (same pattern as admin SSO).
*/
package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
)

type OIDCDiscovery struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	JWKSURI               string `json:"jwks_uri"`
	Issuer                string `json:"issuer"`
}

type OIDCDiscoveryClient interface {
	FetchDiscovery(ctx context.Context, discoveryURL string) (*OIDCDiscovery, error)
}

type DBIdentityProviderResolver struct {
	idpRepo    repository.IdentityProviderRepository
	oidcClient OIDCDiscoveryClient
}

func NewDBIdentityProviderResolver(idpRepo repository.IdentityProviderRepository, oidcClient OIDCDiscoveryClient) *DBIdentityProviderResolver {
	return &DBIdentityProviderResolver{
		idpRepo:    idpRepo,
		oidcClient: oidcClient,
	}
}

func (r *DBIdentityProviderResolver) ResolveOIDCProvider(ctx context.Context, tenantID uuid.UUID) (*ResolvedIdPConfig, error) {
	idps, _, err := r.idpRepo.List(ctx, tenantID, domain.ListOptions{Page: 1, PerPage: 10})
	if err != nil {
		return nil, fmt.Errorf("list identity providers: %w", err)
	}

	for _, idp := range idps {
		if idp.Type != "oidc_1.0" {
			continue
		}

		var cfg struct {
			ClientID     string `json:"client_id"`
			ClientSecret string `json:"client_secret"`
			DiscoveryURL string `json:"discovery_url"`
			Scopes       string `json:"scopes"`
		}
		if err := json.Unmarshal(idp.Config, &cfg); err != nil {
			continue
		}
		if cfg.ClientID == "" || cfg.DiscoveryURL == "" {
			continue
		}

		discovery, err := r.oidcClient.FetchDiscovery(ctx, cfg.DiscoveryURL)
		if err != nil {
			return nil, fmt.Errorf("OIDC discovery for %q: %w", idp.Name, err)
		}

		return &ResolvedIdPConfig{
			ClientID:              cfg.ClientID,
			ClientSecret:          cfg.ClientSecret,
			AuthorizationEndpoint: discovery.AuthorizationEndpoint,
			TokenEndpoint:         discovery.TokenEndpoint,
			Scopes:                cfg.Scopes,
		}, nil
	}

	return nil, fmt.Errorf("no active OIDC identity provider configured for this tenant")
}

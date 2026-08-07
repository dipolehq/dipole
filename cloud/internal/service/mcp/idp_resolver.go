/*
idp_resolver.go resolves the tenant's Identity Provider configuration for
MCP SSO flows. Fetches the OIDC discovery document or parses SAML metadata
to dynamically resolve IdP endpoints.
*/
package mcp

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"

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

type ResolvedSAMLConfig struct {
	SSOURL   string
	EntityID string
}

type DBIdentityProviderResolver struct {
	idpRepo    repository.IdentityProviderRepository
	oidcClient OIDCDiscoveryClient
	httpClient *http.Client
}

func NewDBIdentityProviderResolver(idpRepo repository.IdentityProviderRepository, oidcClient OIDCDiscoveryClient) *DBIdentityProviderResolver {
	return &DBIdentityProviderResolver{
		idpRepo:    idpRepo,
		oidcClient: oidcClient,
		httpClient: &http.Client{Timeout: 10 * time.Second},
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

func (r *DBIdentityProviderResolver) ResolveSAMLProvider(ctx context.Context, tenantID uuid.UUID) (*ResolvedSAMLConfig, error) {
	idps, _, err := r.idpRepo.List(ctx, tenantID, domain.ListOptions{Page: 1, PerPage: 10})
	if err != nil {
		return nil, fmt.Errorf("list identity providers: %w", err)
	}

	for _, idp := range idps {
		if idp.Type != "saml_2.0" {
			continue
		}

		var cfg struct {
			MetadataURL string `json:"metadataUrl"`
			MetadataXML string `json:"metadataXml"`
		}
		if err := json.Unmarshal(idp.Config, &cfg); err != nil {
			continue
		}

		var metadataBytes []byte
		if cfg.MetadataURL != "" {
			metadataBytes, err = r.fetchSAMLMetadata(ctx, cfg.MetadataURL)
			if err != nil {
				return nil, fmt.Errorf("fetch SAML metadata for %q: %w", idp.Name, err)
			}
		} else if cfg.MetadataXML != "" {
			metadataBytes = []byte(cfg.MetadataXML)
		} else {
			continue
		}

		ssoURL, entityID, err := parseSAMLMetadata(metadataBytes)
		if err != nil {
			return nil, fmt.Errorf("parse SAML metadata for %q: %w", idp.Name, err)
		}

		return &ResolvedSAMLConfig{
			SSOURL:   ssoURL,
			EntityID: entityID,
		}, nil
	}

	return nil, fmt.Errorf("no active SAML identity provider configured for this tenant")
}

func (r *DBIdentityProviderResolver) fetchSAMLMetadata(ctx context.Context, metadataURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, metadataURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("metadata endpoint returned %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func parseSAMLMetadata(data []byte) (ssoURL, entityID string, err error) {
	var descriptor struct {
		EntityID         string `xml:"entityID,attr"`
		IDPSSODescriptor struct {
			SingleSignOnServices []struct {
				Binding  string `xml:"Binding,attr"`
				Location string `xml:"Location,attr"`
			} `xml:"SingleSignOnService"`
		} `xml:"IDPSSODescriptor"`
	}
	if err := xml.Unmarshal(data, &descriptor); err != nil {
		return "", "", fmt.Errorf("unmarshal SAML metadata: %w", err)
	}

	entityID = descriptor.EntityID

	for _, svc := range descriptor.IDPSSODescriptor.SingleSignOnServices {
		if svc.Binding == "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" {
			return svc.Location, entityID, nil
		}
	}

	if len(descriptor.IDPSSODescriptor.SingleSignOnServices) > 0 {
		return descriptor.IDPSSODescriptor.SingleSignOnServices[0].Location, entityID, nil
	}

	return "", "", fmt.Errorf("no SingleSignOnService found in SAML metadata")
}

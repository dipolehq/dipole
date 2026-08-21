package mcp

import (
	"context"

	"github.com/dipolehq/dipole/cloud/internal/service/auth"
)

type HTTPOIDCDiscoveryAdapter struct {
	client *auth.HTTPOIDCClient
}

func NewHTTPOIDCDiscoveryAdapter(client *auth.HTTPOIDCClient) *HTTPOIDCDiscoveryAdapter {
	return &HTTPOIDCDiscoveryAdapter{client: client}
}

func (a *HTTPOIDCDiscoveryAdapter) FetchDiscovery(ctx context.Context, discoveryURL string) (*OIDCDiscovery, error) {
	doc, err := a.client.FetchDiscovery(ctx, discoveryURL)
	if err != nil {
		return nil, err
	}
	return &OIDCDiscovery{
		AuthorizationEndpoint: doc.AuthorizationEndpoint,
		TokenEndpoint:         doc.TokenEndpoint,
		JWKSURI:               doc.JWKSURI,
		Issuer:                doc.Issuer,
	}, nil
}

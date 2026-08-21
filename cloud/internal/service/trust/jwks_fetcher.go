// jwks_fetcher.go fetches JWKS from OIDC provider endpoints via HTTPS-only, SSRF-safe, size-bounded requests.
package trust

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-jose/go-jose/v4"

	"github.com/dipolehq/dipole/cloud/internal/service/trust/ssrf"
)

const maxJWKSResponseBytes = 1 << 20 // 1 MiB

var ssrfSafeTransport = func() *http.Transport {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.DialContext = ssrf.SafeDialContext(&net.Dialer{Timeout: 5 * time.Second})
	return t
}()

var httpClient = &http.Client{
	Timeout:   10 * time.Second,
	Transport: ssrfSafeTransport,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 2 {
			return fmt.Errorf("too many redirects")
		}
		if req.URL.Scheme != "https" {
			return fmt.Errorf("redirect to non-HTTPS URL: %s", req.URL)
		}
		return nil
	},
}

func FetchJWKS(ctx context.Context, oidcEndpoint string) (*jose.JSONWebKeySet, error) {
	if !strings.HasPrefix(oidcEndpoint, "https://") {
		return nil, fmt.Errorf("JWKS endpoint must be HTTPS: %s", oidcEndpoint)
	}

	jwksURI, err := fetchJWKSURI(ctx, oidcEndpoint)
	if err != nil {
		return nil, err
	}

	if !strings.HasPrefix(jwksURI, "https://") {
		return nil, fmt.Errorf("jwks_uri must be HTTPS: %s", jwksURI)
	}

	return fetchKeySet(ctx, jwksURI)
}

func fetchJWKSURI(ctx context.Context, oidcEndpoint string) (string, error) {
	discoveryURL := strings.TrimSuffix(oidcEndpoint, "/") + "/.well-known/openid-configuration"

	req, err := http.NewRequestWithContext(ctx, "GET", discoveryURL, nil)
	if err != nil {
		return "", fmt.Errorf("create discovery request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch discovery %s: %w", discoveryURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("discovery %s returned %d", discoveryURL, resp.StatusCode)
	}

	var config struct {
		JWKSURI string `json:"jwks_uri"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxJWKSResponseBytes)).Decode(&config); err != nil {
		return "", fmt.Errorf("parse discovery response: %w", err)
	}
	if config.JWKSURI == "" {
		return "", fmt.Errorf("no jwks_uri in discovery response from %s", discoveryURL)
	}

	return config.JWKSURI, nil
}

func fetchKeySet(ctx context.Context, jwksURI string) (*jose.JSONWebKeySet, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", jwksURI, nil)
	if err != nil {
		return nil, fmt.Errorf("create JWKS request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch JWKS %s: %w", jwksURI, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JWKS %s returned %d", jwksURI, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxJWKSResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("read JWKS body: %w", err)
	}

	var keySet jose.JSONWebKeySet
	if err := json.Unmarshal(body, &keySet); err != nil {
		return nil, fmt.Errorf("parse JWKS: %w", err)
	}

	return &keySet, nil
}

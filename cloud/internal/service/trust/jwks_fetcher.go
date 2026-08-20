/*
jwks_fetcher.go fetches JWKS from OIDC provider endpoints.

FetchJWKS performs OIDC Discovery (fetches .well-known/openid-configuration)
to extract the jwks_uri, then fetches and parses the JWKS. HTTPS only, 10s
timeout, system CA trust store.

Trust Provider configs (oidcEndpoint) are tenant-supplied and untrusted:
a malicious or misconfigured value could point at cloud metadata services
or internal-only infrastructure. httpClient's transport dials through
ssrf.SafeDialContext, which validates the resolved IP — not just the
literal hostname — before every connection the client makes, including
ones reached via redirect. Response bodies are also capped to prevent an
attacker-controlled endpoint from exhausting memory.
*/
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

// maxJWKSResponseBytes bounds how much of a discovery or JWKS response
// body is read. A legitimate JWKS document is a few KB at most; this
// limit exists to stop an attacker-controlled or compromised endpoint
// from exhausting memory with an oversized response.
const maxJWKSResponseBytes = 1 << 20 // 1 MiB

// ssrfSafeTransport is a copy of http.DefaultTransport with DialContext
// swapped for the SSRF-safe dialer. Cloning DefaultTransport's fields
// (rather than starting from a bare &http.Transport{}) preserves behavior
// the original zero-value http.Client{} got implicitly by falling back to
// http.DefaultTransport: proxy support via HTTP_PROXY/HTTPS_PROXY/NO_PROXY
// (relevant if this service is deployed behind an egress proxy) and
// connection-pooling defaults (idle connection reuse, TLS handshake
// timeout). Losing proxy support silently would be a deployment-specific
// outage risk, not just a performance regression.
//
// Note: if an egress proxy IS configured, the proxy — not this process —
// makes the final connection to the destination, so SafeDialContext only
// validates the hop to the proxy itself. That's expected: in a proxied
// topology the proxy is the enforced egress boundary, and SSRF policy
// belongs there.
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

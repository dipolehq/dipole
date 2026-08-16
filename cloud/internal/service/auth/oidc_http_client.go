/*
oidc_http_client.go provides the concrete HTTP implementation of the OIDCClient
interface used by AdminSSOService. Handles OIDC discovery with 1-hour caching
and authorization code exchange.
*/
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

type HTTPOIDCClient struct {
	httpClient *http.Client
	cache      map[string]*cachedDiscovery
	mu         sync.RWMutex
}

type cachedDiscovery struct {
	doc       *OIDCDiscoveryDoc
	fetchedAt time.Time
}

func NewHTTPOIDCClient() *HTTPOIDCClient {
	return &HTTPOIDCClient{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		cache:      make(map[string]*cachedDiscovery),
	}
}

func (c *HTTPOIDCClient) FetchDiscovery(ctx context.Context, discoveryURL string) (*OIDCDiscoveryDoc, error) {
	c.mu.RLock()
	if cached, ok := c.cache[discoveryURL]; ok {
		if time.Since(cached.fetchedAt) < 1*time.Hour {
			c.mu.RUnlock()
			return cached.doc, nil
		}
	}
	c.mu.RUnlock()

	url := strings.TrimSuffix(discoveryURL, "/") + "/.well-known/openid-configuration"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discovery endpoint returned %d", resp.StatusCode)
	}

	var doc OIDCDiscoveryDoc
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.cache[discoveryURL] = &cachedDiscovery{doc: &doc, fetchedAt: time.Now()}
	c.mu.Unlock()

	return &doc, nil
}

func (c *HTTPOIDCClient) ExchangeCode(ctx context.Context, tokenEndpoint string, clientID, clientSecret, code, redirectURI string) (*OIDCTokenResponse, error) {
	data := fmt.Sprintf("grant_type=authorization_code&code=%s&redirect_uri=%s&client_id=%s&client_secret=%s",
		code, redirectURI, clientID, clientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenEndpoint, strings.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp OIDCTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, err
	}
	return &tokenResp, nil
}

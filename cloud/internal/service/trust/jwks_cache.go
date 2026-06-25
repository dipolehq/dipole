/*
jwks_cache.go caches JWKS by OIDC endpoint URL with 1-hour TTL.

JWKSCache serves cached keys immediately on hit, triggers background refresh
when stale, fetches synchronously on miss, and fails closed (returns error)
if no cache exists and fetch fails.
*/
package trust

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/go-jose/go-jose/v4"
)

const jwksTTL = 1 * time.Hour

type jwksCacheEntry struct {
	keys      *jose.JSONWebKeySet
	fetchedAt time.Time
}

type JWKSCache struct {
	mu      sync.RWMutex
	entries map[string]*jwksCacheEntry
	logger  *slog.Logger
}

func NewJWKSCache(logger *slog.Logger) *JWKSCache {
	c := &JWKSCache{
		entries: make(map[string]*jwksCacheEntry),
		logger:  logger,
	}
	go c.preemptiveRefreshLoop()
	return c
}

func (c *JWKSCache) Get(ctx context.Context, oidcEndpoint string) (*jose.JSONWebKeySet, error) {
	c.mu.RLock()
	entry, exists := c.entries[oidcEndpoint]
	if exists && time.Since(entry.fetchedAt) < jwksTTL {
		c.mu.RUnlock()
		return entry.keys, nil
	}
	if exists {
		// Stale — return immediately, refresh in background
		c.mu.RUnlock()
		go c.refresh(oidcEndpoint)
		return entry.keys, nil
	}
	c.mu.RUnlock()

	// Cache miss — fetch synchronously
	keys, err := FetchJWKS(ctx, oidcEndpoint)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.entries[oidcEndpoint] = &jwksCacheEntry{keys: keys, fetchedAt: time.Now()}
	c.mu.Unlock()

	return keys, nil
}

func (c *JWKSCache) refresh(oidcEndpoint string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	keys, err := FetchJWKS(ctx, oidcEndpoint)
	if err != nil {
		c.logger.Warn("JWKS background refresh failed, serving stale", "endpoint", oidcEndpoint, "error", err)
		return
	}

	c.mu.Lock()
	c.entries[oidcEndpoint] = &jwksCacheEntry{keys: keys, fetchedAt: time.Now()}
	c.mu.Unlock()
}

func (c *JWKSCache) preemptiveRefreshLoop() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		c.mu.RLock()
		var stale []string
		for endpoint, entry := range c.entries {
			if time.Since(entry.fetchedAt) > 45*time.Minute {
				stale = append(stale, endpoint)
			}
		}
		c.mu.RUnlock()

		for _, endpoint := range stale {
			c.refresh(endpoint)
		}
	}
}

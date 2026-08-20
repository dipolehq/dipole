// jwks_cache.go caches JWKS by OIDC endpoint URL with 1-hour TTL, serve-stale-while-revalidate, and bounded LRU eviction.
package trust

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-jose/go-jose/v4"
)

const jwksTTL = 1 * time.Hour

const maxJWKSCacheEntries = 1000

type jwksCacheEntry struct {
	keys      *jose.JSONWebKeySet
	fetchedAt time.Time

	lastAccessedAt atomic.Int64
}

func (e *jwksCacheEntry) touch() {
	e.lastAccessedAt.Store(time.Now().UnixNano())
}

func (e *jwksCacheEntry) lastAccess() time.Time {
	return time.Unix(0, e.lastAccessedAt.Load())
}

type JWKSCache struct {
	mu      sync.RWMutex
	entries map[string]*jwksCacheEntry
	logger  *slog.Logger

	maxEntries int
}

func NewJWKSCache(logger *slog.Logger) *JWKSCache {
	c := &JWKSCache{
		entries:    make(map[string]*jwksCacheEntry),
		logger:     logger,
		maxEntries: maxJWKSCacheEntries,
	}
	go c.preemptiveRefreshLoop()
	return c
}

func (c *JWKSCache) Get(ctx context.Context, oidcEndpoint string) (*jose.JSONWebKeySet, error) {
	c.mu.RLock()
	entry, exists := c.entries[oidcEndpoint]
	if exists && time.Since(entry.fetchedAt) < jwksTTL {
		entry.touch()
		c.mu.RUnlock()
		return entry.keys, nil
	}
	if exists {
		entry.touch()
		c.mu.RUnlock()
		go c.refresh(oidcEndpoint)
		return entry.keys, nil
	}
	c.mu.RUnlock()

	keys, err := FetchJWKS(ctx, oidcEndpoint)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.insertLocked(oidcEndpoint, keys)
	c.mu.Unlock()

	return keys, nil
}

func (c *JWKSCache) insertLocked(oidcEndpoint string, keys *jose.JSONWebKeySet) {
	entry := &jwksCacheEntry{keys: keys, fetchedAt: time.Now()}
	entry.touch()
	c.entries[oidcEndpoint] = entry

	if c.maxEntries <= 0 {
		return
	}
	for len(c.entries) > c.maxEntries {
		oldestEndpoint := ""
		var oldestAccess time.Time
		for endpoint, e := range c.entries {
			access := e.lastAccess()
			if oldestEndpoint == "" || access.Before(oldestAccess) {
				oldestEndpoint = endpoint
				oldestAccess = access
			}
		}
		if oldestEndpoint == "" {
			return
		}
		delete(c.entries, oldestEndpoint)
	}
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
	c.insertLocked(oidcEndpoint, keys)
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

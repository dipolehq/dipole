/*
jwks_cache.go caches JWKS by OIDC endpoint URL with 1-hour TTL.

JWKSCache serves cached keys immediately on hit, triggers background refresh
when stale, fetches synchronously on miss, and fails closed (returns error)
if no cache exists and fetch fails.

Cache size is bounded (maxJWKSCacheEntries, LRU eviction) because entries
are keyed by whatever OIDC endpoint string each configured Trust Provider
supplies — in a multi-tenant deployment with many distinct customer
endpoints, an unbounded map would grow for the lifetime of the process
with no way to shrink it, even for endpoints no tenant references anymore.
*/
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

// maxJWKSCacheEntries bounds how many distinct OIDC endpoints JWKSCache
// holds at once. Chosen generously relative to realistic usage (each
// Trust Provider typically configures one of a small handful of
// well-known or per-cluster endpoints; even hundreds of tenants each
// configuring several distinct endpoints stays well under this) while
// still guaranteeing the cache cannot grow without bound over the life
// of a long-running process.
const maxJWKSCacheEntries = 1000

type jwksCacheEntry struct {
	keys      *jose.JSONWebKeySet
	fetchedAt time.Time

	// lastAccessedAt is updated on every Get() call that observes this
	// entry (hit or served-stale), independent of fetchedAt (which only
	// changes when the underlying JWKS is actually re-fetched). It's an
	// atomic int64 of UnixNano, not a plain time.Time, so reads on the
	// Get() hot path can bump it without acquiring the cache's write
	// lock — Get() only needs a read lock to look up the entry, and
	// updating a plain time.Time field concurrently from multiple
	// readers under only an RLock would itself be a data race.
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

	// maxEntries bounds the cache size; see insertLocked. Left unset
	// (zero) by test call sites that construct JWKSCache directly via
	// struct literal rather than NewJWKSCache — a zero value disables
	// eviction rather than evicting down to a limit of zero, which would
	// otherwise silently discard every entry on every insert in tests
	// that never call NewJWKSCache.
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
		// Stale — return immediately, refresh in background
		entry.touch()
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
	c.insertLocked(oidcEndpoint, keys)
	c.mu.Unlock()

	return keys, nil
}

// insertLocked stores a freshly fetched key set and evicts the
// least-recently-accessed entry (repeatedly, in the rare case a single
// eviction isn't enough — e.g. maxEntries was lowered after entries were
// already inserted) until the cache is back within maxEntries. Must be
// called with c.mu held for writing.
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
	// Refreshing an endpoint that's already cached should never itself
	// trigger LRU eviction of some *other* entry — replacing an existing
	// key does not grow len(c.entries), so insertLocked's eviction loop
	// is a no-op here in practice, but routing through the same helper
	// keeps there being exactly one place that decides how entries are
	// stored and touched.
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

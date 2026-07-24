/*
Package credcache provides an in-memory credential cache for policy evaluation
results. Keyed by target host:port, entries expire based on TTL from Cloud or
a configurable default. Thread-safe via sync.RWMutex. Serves cached credentials
during brief Cloud outages (static stability).
*/
package credcache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dipolehq/dipole/edge/internal/injection"
)

type CacheEntry struct {
	Allowed          bool
	Credential       *injection.Credential
	InjectionConfig  *injection.InjectionConfig
	PolicyID         string
	ClientWorkloadID string
	ServerWorkloadID string
	ExpiresAt        time.Time
}

type CredentialCache struct {
	mu         sync.RWMutex
	entries    map[string]*CacheEntry
	defaultTTL time.Duration
}

func NewCredentialCache(defaultTTL time.Duration) *CredentialCache {
	return &CredentialCache{
		entries:    make(map[string]*CacheEntry),
		defaultTTL: defaultTTL,
	}
}

func (c *CredentialCache) Get(host string, port int) (*CacheEntry, bool) {
	key := cacheKey(host, port)

	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()

	if !ok {
		return nil, false
	}

	if time.Now().After(entry.ExpiresAt) {
		return nil, false
	}

	return entry, true
}

func (c *CredentialCache) Set(host string, port int, entry *CacheEntry) {
	key := cacheKey(host, port)

	c.mu.Lock()
	c.entries[key] = entry
	c.mu.Unlock()
}

func (c *CredentialCache) DefaultTTL() time.Duration {
	return c.defaultTTL
}

func (c *CredentialCache) Size() int {
	c.mu.RLock()
	n := len(c.entries)
	c.mu.RUnlock()
	return n
}

func (c *CredentialCache) Cleanup() {
	now := time.Now()

	c.mu.Lock()
	for key, entry := range c.entries {
		if now.After(entry.ExpiresAt) {
			delete(c.entries, key)
		}
	}
	c.mu.Unlock()
}

func (c *CredentialCache) StartCleanupLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.Cleanup()
		}
	}
}

func cacheKey(host string, port int) string {
	return fmt.Sprintf("%s:%d", host, port)
}

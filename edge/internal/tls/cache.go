/*
cache.go provides a thread-safe in-memory certificate cache keyed by hostname.

CertCache stores generated leaf certs with a 24-hour TTL. GetOrGenerate uses
a RWMutex with double-check pattern so concurrent requests for the same hostname
result in exactly one generation. A background goroutine evicts expired entries
every 10 minutes.
*/
package tls

import (
	"crypto/tls"
	"sync"
	"time"
)

type cacheEntry struct {
	cert      *tls.Certificate
	expiresAt time.Time
}

type CertCache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry
	ca      *CertificateAuthority
	ttl     time.Duration
}

func NewCertCache(ca *CertificateAuthority) *CertCache {
	c := &CertCache{
		entries: make(map[string]*cacheEntry),
		ca:      ca,
		ttl:     24 * time.Hour,
	}
	go c.cleanupLoop()
	return c
}

func (c *CertCache) GetOrGenerate(hostname string) (*tls.Certificate, error) {
	c.mu.RLock()
	if entry, ok := c.entries[hostname]; ok && time.Now().Before(entry.expiresAt) {
		c.mu.RUnlock()
		return entry.cert, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	if entry, ok := c.entries[hostname]; ok && time.Now().Before(entry.expiresAt) {
		return entry.cert, nil
	}

	cert, err := GenerateLeafCert(hostname, c.ca)
	if err != nil {
		return nil, err
	}

	c.entries[hostname] = &cacheEntry{
		cert:      cert,
		expiresAt: time.Now().Add(c.ttl),
	}
	return cert, nil
}

func (c *CertCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

func (c *CertCache) cleanupLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		c.evictExpired()
	}
}

func (c *CertCache) evictExpired() {
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	for host, entry := range c.entries {
		if now.After(entry.expiresAt) {
			delete(c.entries, host)
		}
	}
}

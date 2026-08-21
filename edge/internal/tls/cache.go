// cache.go provides a thread-safe, singleflight in-memory leaf certificate cache keyed by hostname.
package tls

import (
	"context"
	"crypto/tls"
	"sync"
	"time"

	"github.com/dipolehq/dipole/edge/internal/metrics"
)

type cacheEntry struct {
	cert      *tls.Certificate
	expiresAt time.Time
}

// CertCache stores leaf certs signed by Cloud, keyed by hostname, honoring
// the lifetime Cloud assigned to each certificate (falling back to a
// conservative default only if Cloud did not return one). inflight
// deduplicates concurrent cache misses for the same hostname to exactly one
// SignLeafCertificate round-trip: a network call is too slow to do the
// cheap RWMutex double-check this cache previously relied on when leaf
// generation was pure in-memory crypto.
type CertCache struct {
	mu       sync.RWMutex
	entries  map[string]*cacheEntry
	inflight map[string]*inflightSign
	anchor   *TrustAnchor
	signer   LeafSigner
	// fallbackTTL bounds how long a cert is cached if Cloud did not return
	// a lifetime; it is not a leaf certificate's actual validity period,
	// which is always determined by Cloud.
	fallbackTTL time.Duration
}

type inflightSign struct {
	done chan struct{}
	cert *tls.Certificate
	err  error
}

func NewCertCache(anchor *TrustAnchor, signer LeafSigner) *CertCache {
	c := &CertCache{
		entries:     make(map[string]*cacheEntry),
		inflight:    make(map[string]*inflightSign),
		anchor:      anchor,
		signer:      signer,
		fallbackTTL: 1 * time.Hour,
	}
	go c.cleanupLoop()
	return c
}

// signTimeout bounds the SignLeafCertificate round-trip that backs a
// singleflight sign operation. It is intentionally not derived from any
// individual caller's context: several connections may be waiting on the
// same inflight sign, and the operation itself must keep running to
// completion (or fail closed on its own schedule) regardless of whether
// the specific caller who happened to trigger it disconnects or times out
// first.
const signTimeout = 10 * time.Second

func (c *CertCache) GetOrGenerate(ctx context.Context, hostname string) (*tls.Certificate, error) {
	if cert, ok := c.getCached(hostname); ok {
		return cert, nil
	}

	c.mu.Lock()
	if cert, ok := c.getCachedLocked(hostname); ok {
		c.mu.Unlock()
		return cert, nil
	}

	if inflight, ok := c.inflight[hostname]; ok {
		c.mu.Unlock()
		return waitForInflight(ctx, inflight)
	}

	inflight := &inflightSign{done: make(chan struct{})}
	c.inflight[hostname] = inflight
	c.mu.Unlock()

	// The sign runs in its own goroutine on a detached, self-bounded
	// context so it is not tied to (and cannot be prematurely canceled by)
	// whichever caller happens to trigger it. Every caller — including
	// this one — waits on it via waitForInflight, which independently
	// honors each caller's own context.
	go c.runSign(hostname, inflight)

	return waitForInflight(ctx, inflight)
}

func (c *CertCache) runSign(hostname string, inflight *inflightSign) {
	signCtx, cancel := context.WithTimeout(context.Background(), signTimeout)
	defer cancel()
	cert, notAfter, err := GenerateLeafCert(signCtx, hostname, c.anchor, c.signer)

	c.mu.Lock()
	delete(c.inflight, hostname)
	if err == nil {
		expiresAt := notAfter
		if expiresAt.IsZero() {
			expiresAt = time.Now().Add(c.fallbackTTL)
		}
		c.entries[hostname] = &cacheEntry{cert: cert, expiresAt: expiresAt}
		metrics.TLSCertsGenerated.Inc()
		metrics.TLSCertCacheSize.Set(float64(len(c.entries)))
	}
	c.mu.Unlock()

	inflight.cert = cert
	inflight.err = err
	close(inflight.done)
}

// waitForInflight waits for a singleflight sign to complete, but gives up
// early if the caller's own context is done first. Other callers waiting
// on the same inflight sign are unaffected — the sign itself keeps running
// on its own detached context (see runSign) and will still populate the
// cache for them.
func waitForInflight(ctx context.Context, inflight *inflightSign) (*tls.Certificate, error) {
	select {
	case <-inflight.done:
		return inflight.cert, inflight.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *CertCache) getCached(hostname string) (*tls.Certificate, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.getCachedLocked(hostname)
}

func (c *CertCache) getCachedLocked(hostname string) (*tls.Certificate, bool) {
	entry, ok := c.entries[hostname]
	if !ok || !time.Now().Before(entry.expiresAt) {
		return nil, false
	}
	return entry.cert, true
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
	metrics.TLSCertCacheSize.Set(float64(len(c.entries)))
}

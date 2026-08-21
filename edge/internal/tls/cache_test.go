package tls

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// countingSigner wraps a fakeSigner and counts how many times SignLeaf was
// actually invoked, so tests can verify the singleflight/cache behavior
// (that a hostname miss triggers exactly one network round-trip even under
// concurrent access).
type countingSigner struct {
	inner LeafSigner
	calls atomic.Int64
}

func (c *countingSigner) SignLeaf(ctx context.Context, csrDER []byte, hostname string) ([]byte, time.Time, error) {
	c.calls.Add(1)
	return c.inner.SignLeaf(ctx, csrDER, hostname)
}

func newTestCache(t *testing.T) (*CertCache, *countingSigner) {
	ca := newTestCA(t)
	anchor, err := NewTrustAnchor(ca.certPEM())
	if err != nil {
		t.Fatalf("NewTrustAnchor: %v", err)
	}
	signer := &countingSigner{inner: &fakeSigner{ca: ca}}
	return NewCertCache(anchor, signer), signer
}

func TestCertCache_HitAndMiss(t *testing.T) {
	cache, signer := newTestCache(t)

	cert1, err := cache.GetOrGenerate(context.Background(), "a.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if cache.Size() != 1 {
		t.Errorf("size = %d, want 1", cache.Size())
	}

	cert2, _ := cache.GetOrGenerate(context.Background(), "a.example.com")
	if cert1 != cert2 {
		t.Error("cache miss on second call (expected hit)")
	}
	if signer.calls.Load() != 1 {
		t.Errorf("signer called %d times, want 1 (second call should be a cache hit)", signer.calls.Load())
	}

	cert3, _ := cache.GetOrGenerate(context.Background(), "b.example.com")
	if cert3 == cert1 {
		t.Error("different hostname returned same cert")
	}
	if cache.Size() != 2 {
		t.Errorf("size = %d, want 2", cache.Size())
	}
}

func TestCertCache_Expiry(t *testing.T) {
	ca := newTestCA(t)
	anchor, err := NewTrustAnchor(ca.certPEM())
	if err != nil {
		t.Fatalf("NewTrustAnchor: %v", err)
	}
	signer := &fakeSigner{ca: ca, notAfter: time.Now().Add(1 * time.Millisecond)}
	cache := NewCertCache(anchor, signer)

	cert1, err := cache.GetOrGenerate(context.Background(), "expire.example.com")
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)

	cert2, err := cache.GetOrGenerate(context.Background(), "expire.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if cert1 == cert2 {
		t.Error("expired entry was served (expected regeneration)")
	}
}

func TestCertCache_FallbackTTLOnZeroNotAfter(t *testing.T) {
	ca := newTestCA(t)
	anchor, err := NewTrustAnchor(ca.certPEM())
	if err != nil {
		t.Fatalf("NewTrustAnchor: %v", err)
	}
	// notAfter left zero: signer returns no lifetime, cache must fall back
	// to fallbackTTL rather than treating the cert as immediately expired.
	signer := &fakeSigner{ca: ca}
	cache := NewCertCache(anchor, signer)
	cache.fallbackTTL = 1 * time.Hour

	cert1, err := cache.GetOrGenerate(context.Background(), "fallback.example.com")
	if err != nil {
		t.Fatal(err)
	}
	cert2, err := cache.GetOrGenerate(context.Background(), "fallback.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if cert1 != cert2 {
		t.Error("cert with fallback TTL should still be cached, got a miss")
	}
}

func TestCertCache_SignerErrorNotCached(t *testing.T) {
	ca := newTestCA(t)
	anchor, err := NewTrustAnchor(ca.certPEM())
	if err != nil {
		t.Fatalf("NewTrustAnchor: %v", err)
	}
	signer := &fakeSigner{ca: ca, err: errors.New("cloud unavailable")}
	cache := NewCertCache(anchor, signer)

	_, err = cache.GetOrGenerate(context.Background(), "denied.example.com")
	if err == nil {
		t.Fatal("expected error from failing signer, got nil")
	}
	if cache.Size() != 0 {
		t.Errorf("size = %d, want 0 (failed sign must not be cached)", cache.Size())
	}
}

// slowSigner blocks until release is closed, letting tests control exactly
// when a signing operation completes relative to a waiter's cancellation.
type slowSigner struct {
	inner   LeafSigner
	release chan struct{}
}

func (s *slowSigner) SignLeaf(ctx context.Context, csrDER []byte, hostname string) ([]byte, time.Time, error) {
	select {
	case <-s.release:
	case <-ctx.Done():
		return nil, time.Time{}, ctx.Err()
	}
	return s.inner.SignLeaf(ctx, csrDER, hostname)
}

// TestCertCache_WaiterCancellationDoesNotFailOthers is a regression test:
// previously, the singleflight sign ran on the context of whichever caller
// happened to trigger it, so that caller's cancellation would abort the
// shared operation and fail every other concurrent waiter for the same
// hostname. The sign must now run on its own detached context so it keeps
// going (and still populates the cache) for the waiters that stay.
func TestCertCache_WaiterCancellationDoesNotFailOthers(t *testing.T) {
	ca := newTestCA(t)
	anchor, err := NewTrustAnchor(ca.certPEM())
	if err != nil {
		t.Fatalf("NewTrustAnchor: %v", err)
	}
	signer := &slowSigner{inner: &fakeSigner{ca: ca}, release: make(chan struct{})}
	cache := NewCertCache(anchor, signer)

	cancelCtx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	patientResults := make(chan error, 5)

	// One caller triggers the sign and will have its own context canceled
	// mid-flight.
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := cache.GetOrGenerate(cancelCtx, "shared.example.com")
		if err == nil {
			t.Error("canceled waiter expected an error, got nil")
		}
	}()

	// Several other callers wait on the same inflight sign with contexts
	// that are never canceled.
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := cache.GetOrGenerate(context.Background(), "shared.example.com")
			patientResults <- err
		}()
	}

	// Give all goroutines time to register as waiters, then cancel only
	// the first caller's context.
	time.Sleep(50 * time.Millisecond)
	cancel()
	time.Sleep(50 * time.Millisecond)

	// Now let the underlying sign actually complete.
	close(signer.release)
	wg.Wait()
	close(patientResults)

	for err := range patientResults {
		if err != nil {
			t.Errorf("uncanceled waiter failed due to another caller's cancellation: %v", err)
		}
	}

	if cache.Size() != 1 {
		t.Errorf("size = %d, want 1 (sign should have completed and been cached despite one waiter canceling)", cache.Size())
	}
}

func TestCertCache_Concurrency(t *testing.T) {
	cache, signer := newTestCache(t)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := cache.GetOrGenerate(context.Background(), "concurrent.example.com")
			if err != nil {
				t.Errorf("concurrent get: %v", err)
			}
		}()
	}
	wg.Wait()

	if cache.Size() != 1 {
		t.Errorf("size = %d after 100 concurrent calls, want 1", cache.Size())
	}
	if signer.calls.Load() != 1 {
		t.Errorf("signer called %d times under 100 concurrent misses, want exactly 1 (singleflight)", signer.calls.Load())
	}
}

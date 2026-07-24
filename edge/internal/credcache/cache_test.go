package credcache

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/dipolehq/dipole/edge/internal/injection"
)

func TestGet_Miss_EmptyCache(t *testing.T) {
	c := NewCredentialCache(60 * time.Second)

	entry, ok := c.Get("api.stripe.com", 443)
	if ok {
		t.Fatal("expected miss on empty cache")
	}
	if entry != nil {
		t.Fatal("expected nil entry on miss")
	}
}

func TestGet_Hit_ValidEntry(t *testing.T) {
	c := NewCredentialCache(60 * time.Second)

	cred := &injection.Credential{Type: "http_bearer", Value: "token123"}
	injCfg := &injection.InjectionConfig{Method: "http_bearer", HeaderName: "Authorization", Prefix: "Bearer "}

	c.Set("api.stripe.com", 443, &CacheEntry{
		Allowed:         true,
		Credential:      cred,
		InjectionConfig: injCfg,
		ExpiresAt:       time.Now().Add(5 * time.Minute),
	})

	entry, ok := c.Get("api.stripe.com", 443)
	if !ok {
		t.Fatal("expected hit")
	}
	if !entry.Allowed {
		t.Fatal("expected Allowed=true")
	}
	if entry.Credential.Value != "token123" {
		t.Fatalf("credential value = %q, want %q", entry.Credential.Value, "token123")
	}
	if entry.InjectionConfig.Method != "http_bearer" {
		t.Fatalf("injection method = %q, want %q", entry.InjectionConfig.Method, "http_bearer")
	}
}

func TestGet_Miss_ExpiredEntry(t *testing.T) {
	c := NewCredentialCache(60 * time.Second)

	c.Set("api.stripe.com", 443, &CacheEntry{
		Allowed:    true,
		Credential: &injection.Credential{Value: "expired-token"},
		ExpiresAt:  time.Now().Add(-1 * time.Second),
	})

	entry, ok := c.Get("api.stripe.com", 443)
	if ok {
		t.Fatal("expected miss for expired entry")
	}
	if entry != nil {
		t.Fatal("expected nil entry for expired")
	}
}

func TestGet_DifferentHostPort(t *testing.T) {
	c := NewCredentialCache(60 * time.Second)

	c.Set("api.stripe.com", 443, &CacheEntry{
		Allowed:   true,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	})

	_, ok := c.Get("api.stripe.com", 8443)
	if ok {
		t.Fatal("expected miss for different port")
	}

	_, ok = c.Get("other.host.com", 443)
	if ok {
		t.Fatal("expected miss for different host")
	}
}

func TestSet_OverwritesExisting(t *testing.T) {
	c := NewCredentialCache(60 * time.Second)

	c.Set("api.stripe.com", 443, &CacheEntry{
		Allowed:    true,
		Credential: &injection.Credential{Value: "old-token"},
		ExpiresAt:  time.Now().Add(5 * time.Minute),
	})

	c.Set("api.stripe.com", 443, &CacheEntry{
		Allowed:    true,
		Credential: &injection.Credential{Value: "new-token"},
		ExpiresAt:  time.Now().Add(10 * time.Minute),
	})

	entry, ok := c.Get("api.stripe.com", 443)
	if !ok {
		t.Fatal("expected hit after overwrite")
	}
	if entry.Credential.Value != "new-token" {
		t.Fatalf("credential value = %q, want %q", entry.Credential.Value, "new-token")
	}
}

func TestSize_ReflectsCurrentEntries(t *testing.T) {
	c := NewCredentialCache(60 * time.Second)

	if c.Size() != 0 {
		t.Fatalf("size = %d, want 0", c.Size())
	}

	c.Set("host1.com", 443, &CacheEntry{ExpiresAt: time.Now().Add(time.Minute)})
	if c.Size() != 1 {
		t.Fatalf("size = %d, want 1", c.Size())
	}

	c.Set("host2.com", 443, &CacheEntry{ExpiresAt: time.Now().Add(time.Minute)})
	if c.Size() != 2 {
		t.Fatalf("size = %d, want 2", c.Size())
	}

	c.Set("host1.com", 443, &CacheEntry{ExpiresAt: time.Now().Add(time.Minute)})
	if c.Size() != 2 {
		t.Fatalf("size after overwrite = %d, want 2", c.Size())
	}
}

func TestCleanup_RemovesExpiredEntries(t *testing.T) {
	c := NewCredentialCache(60 * time.Second)

	c.Set("expired.com", 443, &CacheEntry{
		Allowed:   true,
		ExpiresAt: time.Now().Add(-1 * time.Minute),
	})
	c.Set("valid.com", 443, &CacheEntry{
		Allowed:   true,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	})
	c.Set("also-expired.com", 8080, &CacheEntry{
		Allowed:   true,
		ExpiresAt: time.Now().Add(-30 * time.Second),
	})

	if c.Size() != 3 {
		t.Fatalf("size before cleanup = %d, want 3", c.Size())
	}

	c.Cleanup()

	if c.Size() != 1 {
		t.Fatalf("size after cleanup = %d, want 1", c.Size())
	}

	_, ok := c.Get("valid.com", 443)
	if !ok {
		t.Fatal("valid entry should still be accessible after cleanup")
	}

	_, ok = c.Get("expired.com", 443)
	if ok {
		t.Fatal("expired entry should be gone after cleanup")
	}
}

func TestCleanup_EmptyCache(t *testing.T) {
	c := NewCredentialCache(60 * time.Second)
	c.Cleanup()
	if c.Size() != 0 {
		t.Fatalf("size = %d, want 0", c.Size())
	}
}

func TestDefaultTTL(t *testing.T) {
	c := NewCredentialCache(90 * time.Second)
	if c.DefaultTTL() != 90*time.Second {
		t.Fatalf("DefaultTTL = %v, want 90s", c.DefaultTTL())
	}
}

func TestGet_NilCredentialEntry(t *testing.T) {
	c := NewCredentialCache(60 * time.Second)

	c.Set("passthrough.com", 443, &CacheEntry{
		Allowed:    true,
		Credential: nil,
		ExpiresAt:  time.Now().Add(5 * time.Minute),
	})

	entry, ok := c.Get("passthrough.com", 443)
	if !ok {
		t.Fatal("expected hit")
	}
	if entry.Credential != nil {
		t.Fatal("expected nil credential")
	}
	if !entry.Allowed {
		t.Fatal("expected Allowed=true")
	}
}

func TestStartCleanupLoop_StopsOnContextCancel(t *testing.T) {
	c := NewCredentialCache(60 * time.Second)

	c.Set("expired.com", 443, &CacheEntry{
		ExpiresAt: time.Now().Add(-1 * time.Minute),
	})

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		c.StartCleanupLoop(ctx, 50*time.Millisecond)
		close(done)
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("StartCleanupLoop did not stop after context cancel")
	}

	if c.Size() != 0 {
		t.Fatalf("expired entry should have been cleaned, size = %d", c.Size())
	}
}

func TestConcurrency(t *testing.T) {
	c := NewCredentialCache(60 * time.Second)

	var wg sync.WaitGroup
	const goroutines = 100
	const iterations = 100

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				host := "host" + string(rune('A'+id%26))
				port := 443 + (j % 10)

				c.Set(host+".com", port, &CacheEntry{
					Allowed:    true,
					Credential: &injection.Credential{Value: "token"},
					ExpiresAt:  time.Now().Add(time.Minute),
				})

				c.Get(host+".com", port)
				c.Size()

				if j%50 == 0 {
					c.Cleanup()
				}
			}
		}(i)
	}

	wg.Wait()

	if c.Size() < 0 {
		t.Fatal("size should not be negative")
	}
}

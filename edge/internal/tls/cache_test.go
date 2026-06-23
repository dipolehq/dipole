package tls

import (
	"sync"
	"testing"
	"time"
)

func TestCertCache_HitAndMiss(t *testing.T) {
	ca, _ := NewCertificateAuthority()
	cache := NewCertCache(ca)

	cert1, err := cache.GetOrGenerate("a.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if cache.Size() != 1 {
		t.Errorf("size = %d, want 1", cache.Size())
	}

	cert2, _ := cache.GetOrGenerate("a.example.com")
	if cert1 != cert2 {
		t.Error("cache miss on second call (expected hit)")
	}

	cert3, _ := cache.GetOrGenerate("b.example.com")
	if cert3 == cert1 {
		t.Error("different hostname returned same cert")
	}
	if cache.Size() != 2 {
		t.Errorf("size = %d, want 2", cache.Size())
	}
}

func TestCertCache_Expiry(t *testing.T) {
	ca, _ := NewCertificateAuthority()
	cache := &CertCache{
		entries: make(map[string]*cacheEntry),
		ca:      ca,
		ttl:     1 * time.Millisecond,
	}

	cert1, _ := cache.GetOrGenerate("expire.example.com")
	time.Sleep(5 * time.Millisecond)

	cert2, _ := cache.GetOrGenerate("expire.example.com")
	if cert1 == cert2 {
		t.Error("expired entry was served (expected regeneration)")
	}
}

func TestCertCache_Concurrency(t *testing.T) {
	ca, _ := NewCertificateAuthority()
	cache := NewCertCache(ca)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := cache.GetOrGenerate("concurrent.example.com")
			if err != nil {
				t.Errorf("concurrent get: %v", err)
			}
		}()
	}
	wg.Wait()

	if cache.Size() != 1 {
		t.Errorf("size = %d after 100 concurrent calls, want 1", cache.Size())
	}
}

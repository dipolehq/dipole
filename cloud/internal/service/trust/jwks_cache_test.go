package trust

import (
	"context"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
)

func newTestKeySet(kid string) *jose.JSONWebKeySet {
	return &jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{KeyID: kid, Algorithm: "RS256", Use: "sig"}}}
}

// TestJWKSCache_ZeroValueMaxEntries_EvictionDisabled documents and locks
// in the compatibility behavior every existing test in this package
// relies on: JWKSCache{entries: ..., logger: nil} constructed directly
// via struct literal (bypassing NewJWKSCache) has maxEntries == 0, and
// insertLocked must treat that as "no limit", not "limit of zero" — the
// latter would silently discard every entry these tests insert.
func TestJWKSCache_ZeroValueMaxEntries_EvictionDisabled(t *testing.T) {
	c := &JWKSCache{entries: map[string]*jwksCacheEntry{}}

	c.mu.Lock()
	for i := 0; i < 10; i++ {
		c.insertLocked(string(rune('a'+i)), newTestKeySet("kid"))
	}
	c.mu.Unlock()

	if len(c.entries) != 10 {
		t.Errorf("len(entries) = %d, want 10 (eviction must be disabled for maxEntries=0)", len(c.entries))
	}
}

func TestJWKSCache_InsertLocked_EvictsLeastRecentlyAccessed(t *testing.T) {
	c := &JWKSCache{entries: map[string]*jwksCacheEntry{}, maxEntries: 2}

	c.mu.Lock()
	c.insertLocked("first", newTestKeySet("kid-1"))
	c.mu.Unlock()

	// Ensure distinct, monotonically increasing lastAccessedAt values —
	// UnixNano has enough resolution in practice, but a tiny sleep makes
	// the ordering deterministic regardless of clock resolution on any
	// given CI runner.
	time.Sleep(2 * time.Millisecond)

	c.mu.Lock()
	c.insertLocked("second", newTestKeySet("kid-2"))
	c.mu.Unlock()

	time.Sleep(2 * time.Millisecond)

	// Access "first" again so it's now more recently used than "second".
	c.mu.RLock()
	c.entries["first"].touch()
	c.mu.RUnlock()

	time.Sleep(2 * time.Millisecond)

	// Inserting a third entry with maxEntries=2 must evict the least
	// recently accessed one — "second", since "first" was touched more
	// recently than "second" was ever accessed.
	c.mu.Lock()
	c.insertLocked("third", newTestKeySet("kid-3"))
	c.mu.Unlock()

	if len(c.entries) != 2 {
		t.Fatalf("len(entries) = %d, want 2", len(c.entries))
	}
	if _, ok := c.entries["second"]; ok {
		t.Error("expected \"second\" (least recently accessed) to be evicted")
	}
	if _, ok := c.entries["first"]; !ok {
		t.Error("expected \"first\" (recently touched) to survive eviction")
	}
	if _, ok := c.entries["third"]; !ok {
		t.Error("expected \"third\" (just inserted) to be present")
	}
}

func TestJWKSCache_InsertLocked_ReplacingExistingKeyDoesNotEvict(t *testing.T) {
	c := &JWKSCache{entries: map[string]*jwksCacheEntry{}, maxEntries: 2}

	c.mu.Lock()
	c.insertLocked("a", newTestKeySet("kid-a"))
	c.insertLocked("b", newTestKeySet("kid-b"))
	// Re-inserting an existing key (simulating a background refresh)
	// must not trigger eviction of anything, since len(entries) doesn't
	// grow past maxEntries.
	c.insertLocked("a", newTestKeySet("kid-a-refreshed"))
	c.mu.Unlock()

	if len(c.entries) != 2 {
		t.Fatalf("len(entries) = %d, want 2", len(c.entries))
	}
	if c.entries["a"].keys.Keys[0].KeyID != "kid-a-refreshed" {
		t.Error("expected refreshed entry to have the new key material")
	}
	if _, ok := c.entries["b"]; !ok {
		t.Error("expected \"b\" to still be present after refreshing \"a\"")
	}
}

func TestJWKSCache_Get_TouchesEntryOnHit(t *testing.T) {
	c := &JWKSCache{entries: map[string]*jwksCacheEntry{
		"https://example.com": {keys: newTestKeySet("kid-1"), fetchedAt: time.Now()},
	}}

	before := c.entries["https://example.com"].lastAccess()
	time.Sleep(2 * time.Millisecond)

	if _, err := c.Get(context.Background(), "https://example.com"); err != nil {
		t.Fatal(err)
	}

	after := c.entries["https://example.com"].lastAccess()
	if !after.After(before) {
		t.Error("expected Get() on a fresh cache hit to update lastAccessedAt")
	}
}

func TestJWKSCache_MaxEntries_BoundedUnderSustainedInserts(t *testing.T) {
	// Simulates many distinct OIDC endpoints being inserted over time
	// (e.g. many tenants each configuring a different issuer) and
	// asserts the cache never exceeds its configured bound, regardless
	// of how many distinct endpoints have been seen in total.
	c := &JWKSCache{entries: map[string]*jwksCacheEntry{}, maxEntries: 50}

	for i := 0; i < 500; i++ {
		c.mu.Lock()
		c.insertLocked(string(rune(i)), newTestKeySet("kid"))
		c.mu.Unlock()
	}

	if len(c.entries) != 50 {
		t.Errorf("len(entries) = %d, want 50 (bounded)", len(c.entries))
	}
}

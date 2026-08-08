package mcp_gateway

import (
	"sync"
	"testing"

	"github.com/google/uuid"
)

// The buffer is shared by every tenant and its entries name end users, so a read
// must return only the reader's own tenant's entries.
func TestTracingBus_RecentIsTenantScoped(t *testing.T) {
	bus := NewTracingBus()
	tenantA, tenantB := uuid.New(), uuid.New()

	bus.Emit(TracingEntry{TenantID: tenantA, ClientWorkload: TracingWL{Name: "alice@a.com"}})
	bus.Emit(TracingEntry{TenantID: tenantA, ClientWorkload: TracingWL{Name: "bob@a.com"}})
	bus.Emit(TracingEntry{TenantID: tenantB, ClientWorkload: TracingWL{Name: "carol@b.com"}})

	aEntries := bus.Recent(tenantA)
	if len(aEntries) != 2 {
		t.Fatalf("expected 2 entries for tenant A, got %d", len(aEntries))
	}
	for _, e := range aEntries {
		if e.TenantID != tenantA {
			t.Errorf("tenant A read returned an entry for %s", e.TenantID)
		}
		if e.ClientWorkload.Name == "carol@b.com" {
			t.Error("tenant A must not see tenant B's user")
		}
	}

	if got := len(bus.Recent(tenantB)); got != 1 {
		t.Errorf("expected 1 entry for tenant B, got %d", got)
	}
	if got := len(bus.Recent(uuid.New())); got != 0 {
		t.Errorf("expected 0 entries for an unrelated tenant, got %d", got)
	}
	if got := len(bus.Recent(uuid.Nil)); got != 0 {
		t.Errorf("a nil tenant must match nothing, got %d entries", got)
	}
}

// An entry that cannot be attributed to a tenant is dropped: nobody can be shown
// it safely, and keeping it would consume ring capacity and invite a future
// unfiltered read.
func TestTracingBus_DropsUnattributedEntries(t *testing.T) {
	bus := NewTracingBus()
	tenant := uuid.New()

	bus.Emit(TracingEntry{ClientWorkload: TracingWL{Name: "nobody@example.com"}})
	bus.Emit(TracingEntry{TenantID: tenant})

	if got := len(bus.Recent(tenant)); got != 1 {
		t.Fatalf("expected only the attributed entry, got %d", got)
	}
}

// Emit assigns an ID and timestamp when absent, so the frontend always has a
// stable key and ordering.
func TestTracingBus_EmitFillsIDAndTimestamp(t *testing.T) {
	bus := NewTracingBus()
	tenant := uuid.New()

	bus.Emit(TracingEntry{TenantID: tenant})

	entries := bus.Recent(tenant)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].ID == "" {
		t.Error("expected a generated ID")
	}
	if entries[0].Timestamp == "" {
		t.Error("expected a generated timestamp")
	}
}

func TestTracingBus_RingBufferBoundedAndOldestDropped(t *testing.T) {
	bus := NewTracingBus()
	tenant := uuid.New()

	for i := 0; i < maxTracingEntries+10; i++ {
		bus.Emit(TracingEntry{TenantID: tenant, ID: uuid.New().String()})
	}

	if got := len(bus.Recent(tenant)); got != maxTracingEntries {
		t.Errorf("expected the buffer to cap at %d, got %d", maxTracingEntries, got)
	}
}

// A live subscriber receives only its own tenant's entries.
func TestTracingBus_SubscribeIsTenantScoped(t *testing.T) {
	bus := NewTracingBus()
	tenantA, tenantB := uuid.New(), uuid.New()

	id, ch := bus.Subscribe(tenantA)
	defer bus.Unsubscribe(id)

	bus.Emit(TracingEntry{TenantID: tenantB, ID: "b-1"})
	bus.Emit(TracingEntry{TenantID: tenantA, ID: "a-1"})

	select {
	case entry := <-ch:
		if entry.ID != "a-1" {
			t.Fatalf("subscriber for tenant A received %q", entry.ID)
		}
	default:
		t.Fatal("expected tenant A's entry to be delivered")
	}

	select {
	case entry := <-ch:
		t.Fatalf("expected no further entries, got %q for tenant %s", entry.ID, entry.TenantID)
	default:
	}
}

func TestTracingBus_ConcurrentEmitAndReadIsRaceFree(t *testing.T) {
	bus := NewTracingBus()
	tenants := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}

	var wg sync.WaitGroup
	for _, tenant := range tenants {
		wg.Add(2)
		go func(id uuid.UUID) {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				bus.Emit(TracingEntry{TenantID: id})
			}
		}(tenant)
		go func(id uuid.UUID) {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				for _, e := range bus.Recent(id) {
					if e.TenantID != id {
						t.Errorf("read for %s returned an entry for %s", id, e.TenantID)
						return
					}
				}
			}
		}(tenant)
	}
	wg.Wait()
}

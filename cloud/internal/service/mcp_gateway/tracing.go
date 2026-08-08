/*
tracing.go provides a live, in-memory ring buffer for MCP authorization tracing.

TracingBus holds the last N entries and notifies connected subscribers in
real-time. Entries are ephemeral — they are never persisted to the database.
This is a diagnostic tool for debugging MCP Gateway authorization flows.

The buffer is shared by every tenant, so every entry is tagged with the tenant it
belongs to and readers only ever receive their own. This is not optional: an entry
records the end user's identity (an email address, taken from the token's `email`
claim) along with their client IP, so an unfiltered read would disclose one
tenant's users to another's administrators.
*/
package mcp_gateway

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

const maxTracingEntries = 200

type TracingEntry struct {
	// TenantID scopes the entry. Deliberately not serialised: it is used to decide
	// who may read the entry, and the reader already knows their own tenant, so
	// echoing it back would only widen what the response exposes.
	TenantID       uuid.UUID     `json:"-"`
	ID             string        `json:"id"`
	Timestamp      string        `json:"timestamp"`
	ClientIP       string        `json:"clientIP"`
	AccessPolicy   TracingPolicy `json:"accessPolicy"`
	RedirectURI    string        `json:"redirectUri"`
	ClientWorkload TracingWL     `json:"clientWorkload"`
	ResourceServer string        `json:"resourceServer"`
	ServerWorkload TracingWL     `json:"serverWorkload"`
	Severity       string        `json:"severity"`
}

type TracingPolicy struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type TracingWL struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type TracingBus struct {
	mu          sync.RWMutex
	entries     []TracingEntry
	subscribers map[string]tracingSubscriber
}

type tracingSubscriber struct {
	tenantID uuid.UUID
	ch       chan TracingEntry
}

func NewTracingBus() *TracingBus {
	return &TracingBus{
		entries:     make([]TracingEntry, 0, maxTracingEntries),
		subscribers: make(map[string]tracingSubscriber),
	}
}

// Emit records an entry for one tenant and fans it out to that tenant's
// subscribers only.
//
// An entry with no tenant is dropped rather than stored: nobody could be shown it
// safely, so keeping it would consume ring capacity that another tenant's
// diagnostics need, and would leave a row that a future reader might forget to
// filter.
func (b *TracingBus) Emit(entry TracingEntry) {
	if entry.TenantID == uuid.Nil {
		return
	}
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	if entry.Timestamp == "" {
		entry.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)
	}

	b.mu.Lock()
	if len(b.entries) >= maxTracingEntries {
		b.entries = b.entries[1:]
	}
	b.entries = append(b.entries, entry)

	// Notify this tenant's subscribers (non-blocking).
	for _, sub := range b.subscribers {
		if sub.tenantID != entry.TenantID {
			continue
		}
		select {
		case sub.ch <- entry:
		default:
			// Subscriber is slow — drop the entry for them
		}
	}
	b.mu.Unlock()
}

// Recent returns the buffered entries belonging to one tenant, oldest first.
// A nil tenant matches nothing.
func (b *TracingBus) Recent(tenantID uuid.UUID) []TracingEntry {
	b.mu.RLock()
	defer b.mu.RUnlock()

	result := make([]TracingEntry, 0, len(b.entries))
	if tenantID == uuid.Nil {
		return result
	}
	for _, entry := range b.entries {
		if entry.TenantID == tenantID {
			result = append(result, entry)
		}
	}
	return result
}

// Subscribe registers a live listener for one tenant's entries.
func (b *TracingBus) Subscribe(tenantID uuid.UUID) (string, chan TracingEntry) {
	id := uuid.New().String()
	ch := make(chan TracingEntry, 32)

	b.mu.Lock()
	b.subscribers[id] = tracingSubscriber{tenantID: tenantID, ch: ch}
	b.mu.Unlock()

	return id, ch
}

func (b *TracingBus) Unsubscribe(id string) {
	b.mu.Lock()
	if sub, ok := b.subscribers[id]; ok {
		close(sub.ch)
		delete(b.subscribers, id)
	}
	b.mu.Unlock()
}

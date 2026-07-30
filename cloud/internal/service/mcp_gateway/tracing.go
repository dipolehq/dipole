/*
tracing.go provides a live, in-memory ring buffer for MCP authorization tracing.

TracingBus holds the last N entries and notifies connected SSE subscribers in
real-time. Entries are ephemeral — they are never persisted to the database.
This is a diagnostic tool for debugging MCP Gateway authorization flows.
*/
package mcp_gateway

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

const maxTracingEntries = 200

type TracingEntry struct {
	ID              string        `json:"id"`
	Timestamp       string        `json:"timestamp"`
	ClientIP        string        `json:"clientIP"`
	AccessPolicy    TracingPolicy `json:"accessPolicy"`
	RedirectURI     string        `json:"redirectUri"`
	ClientWorkload  TracingWL     `json:"clientWorkload"`
	ResourceServer  string        `json:"resourceServer"`
	ServerWorkload  TracingWL     `json:"serverWorkload"`
	Severity        string        `json:"severity"`
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
	subscribers map[string]chan TracingEntry
}

func NewTracingBus() *TracingBus {
	return &TracingBus{
		entries:     make([]TracingEntry, 0, maxTracingEntries),
		subscribers: make(map[string]chan TracingEntry),
	}
}

func (b *TracingBus) Emit(entry TracingEntry) {
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

	// Notify all subscribers (non-blocking)
	for _, ch := range b.subscribers {
		select {
		case ch <- entry:
		default:
			// Subscriber is slow — drop the entry for them
		}
	}
	b.mu.Unlock()
}

func (b *TracingBus) Recent() []TracingEntry {
	b.mu.RLock()
	defer b.mu.RUnlock()
	result := make([]TracingEntry, len(b.entries))
	copy(result, b.entries)
	return result
}

func (b *TracingBus) Subscribe() (string, chan TracingEntry) {
	id := uuid.New().String()
	ch := make(chan TracingEntry, 32)

	b.mu.Lock()
	b.subscribers[id] = ch
	b.mu.Unlock()

	return id, ch
}

func (b *TracingBus) Unsubscribe(id string) {
	b.mu.Lock()
	if ch, ok := b.subscribers[id]; ok {
		close(ch)
		delete(b.subscribers, id)
	}
	b.mu.Unlock()
}

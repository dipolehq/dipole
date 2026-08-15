/*
tracing_pipeline.go implements an in-memory ring buffer for live MCP
authorization tracing.

Trace entries are ephemeral diagnostic data — never persisted to the database.
The buffer is shared across tenants but reads are always scoped: Recent()
returns only entries belonging to the requested tenant. Subscribers receive
real-time push via channels for live frontend polling.
*/
package events

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

const tracingRingSize = 200

type TracingPipeline struct {
	mu          sync.RWMutex
	entries     []TraceEntry
	subscribers map[string]tracingSubscriber
	metrics     *Metrics
}

type tracingSubscriber struct {
	tenantID uuid.UUID
	ch       chan TraceEntry
}

func newTracingPipeline(metrics *Metrics) *TracingPipeline {
	return &TracingPipeline{
		entries:     make([]TraceEntry, 0, tracingRingSize),
		subscribers: make(map[string]tracingSubscriber),
		metrics:     metrics,
	}
}

func (p *TracingPipeline) Emit(entry TraceEntry) {
	if entry.TenantID == uuid.Nil {
		return
	}
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}

	p.mu.Lock()
	if len(p.entries) >= tracingRingSize {
		p.entries = p.entries[1:]
	}
	p.entries = append(p.entries, entry)

	for _, sub := range p.subscribers {
		if sub.tenantID != entry.TenantID {
			continue
		}
		select {
		case sub.ch <- entry:
		default:
		}
	}
	p.mu.Unlock()

	p.metrics.TracingEmitted.Add(1)
}

func (p *TracingPipeline) Recent(tenantID uuid.UUID) []TraceEntry {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if tenantID == uuid.Nil {
		return []TraceEntry{}
	}

	result := make([]TraceEntry, 0, len(p.entries))
	for _, entry := range p.entries {
		if entry.TenantID == tenantID {
			result = append(result, entry)
		}
	}
	return result
}

func (p *TracingPipeline) Subscribe(tenantID uuid.UUID) (string, chan TraceEntry) {
	id := uuid.New().String()
	ch := make(chan TraceEntry, 32)

	p.mu.Lock()
	p.subscribers[id] = tracingSubscriber{tenantID: tenantID, ch: ch}
	p.mu.Unlock()

	return id, ch
}

func (p *TracingPipeline) Unsubscribe(id string) {
	p.mu.Lock()
	if sub, ok := p.subscribers[id]; ok {
		close(sub.ch)
		delete(p.subscribers, id)
	}
	p.mu.Unlock()
}

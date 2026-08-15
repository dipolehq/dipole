/*
pipeline.go is the single entry point for all event emission in the Dipole
cloud service.

EventBus composes the four category-specific pipelines (Security, Workload,
Tracing, Audit) behind a unified API. Every event producer in the system
receives an *EventBus as its sole dependency for logging — no direct access
to individual pipelines or repositories.
*/
package events

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

type EventBus struct {
	security *SecurityPipeline
	workload *WorkloadPipeline
	tracing  *TracingPipeline
	audit    *AuditPipeline
	metrics  *Metrics
	cancel   context.CancelFunc
}

func NewEventBus(repo Repository, logger *slog.Logger) *EventBus {
	metrics := &Metrics{}

	ctx, cancel := context.WithCancel(context.Background())

	wp := newWorkloadPipeline(repo, logger, metrics)
	go wp.Run(ctx)

	return &EventBus{
		security: newSecurityPipeline(repo, logger, metrics),
		workload: wp,
		tracing:  newTracingPipeline(metrics),
		audit:    newAuditPipeline(repo, logger, metrics),
		metrics:  metrics,
		cancel:   cancel,
	}
}

func (b *EventBus) EmitSecurity(ctx context.Context, event SecurityEvent) error {
	return b.security.Emit(ctx, event)
}

func (b *EventBus) EmitWorkload(event WorkloadEvent) {
	b.workload.Emit(event)
}

func (b *EventBus) EmitTrace(entry TraceEntry) {
	b.tracing.Emit(entry)
}

func (b *EventBus) EmitAudit(ctx context.Context, event AuditEvent) error {
	return b.audit.Emit(ctx, event)
}

func (b *EventBus) IngestFromProxy(ctx context.Context, tenantID uuid.UUID, incoming []WorkloadEvent) error {
	if b.workload.repo == nil {
		return nil
	}
	for i := range incoming {
		incoming[i].TenantID = tenantID
		if incoming[i].Timestamp.IsZero() {
			incoming[i].Timestamp = time.Now().UTC()
		}
		if incoming[i].ConnectionID == uuid.Nil {
			incoming[i].ConnectionID = uuid.New()
		}
		incoming[i].Payload = redactPayload(incoming[i].Payload)
	}
	return b.workload.repo.InsertWorkloadEventBatch(ctx, incoming)
}

func (b *EventBus) RecentTraces(tenantID uuid.UUID) []TraceEntry {
	return b.tracing.Recent(tenantID)
}

func (b *EventBus) SubscribeTraces(tenantID uuid.UUID) (string, chan TraceEntry) {
	return b.tracing.Subscribe(tenantID)
}

func (b *EventBus) UnsubscribeTraces(id string) {
	b.tracing.Unsubscribe(id)
}

func (b *EventBus) Metrics() *Metrics {
	return b.metrics
}

func (b *EventBus) Shutdown(timeout time.Duration) {
	b.cancel()
}

// pipeline.go is the single entry point for all event emission in the Dipole cloud service: EventBus composes the Security, Workload, and Audit pipelines behind a unified API.
package events

import (
	"context"
	"log/slog"
	"time"

	"github.com/dipolehq/dipole/cloud/internal/scope"
	"github.com/google/uuid"
)

type EventBus struct {
	security *SecurityPipeline
	workload *WorkloadPipeline
	audit    *AuditPipeline
	metrics  *Metrics
	logger   *slog.Logger
	cancel   context.CancelFunc
	done     chan struct{}
}

func NewEventBus(repo Repository, logger *slog.Logger) *EventBus {
	metrics := &Metrics{}

	ctx, cancel := context.WithCancel(context.Background())

	wp := newWorkloadPipeline(repo, logger, metrics)
	done := make(chan struct{})
	go func() {
		defer close(done)
		wp.Run(ctx)
	}()

	return &EventBus{
		security: newSecurityPipeline(repo, logger, metrics),
		workload: wp,
		audit:    newAuditPipeline(repo, logger, metrics),
		metrics:  metrics,
		logger:   logger,
		cancel:   cancel,
		done:     done,
	}
}

func (b *EventBus) EmitSecurity(ctx context.Context, event SecurityEvent) error {
	return b.security.Emit(ctx, event)
}

func (b *EventBus) EmitWorkload(event WorkloadEvent) {
	b.workload.Emit(event)
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
		if incoming[i].EventID == uuid.Nil {
			incoming[i].EventID = uuid.New()
		}
		if incoming[i].ConnectionID == uuid.Nil {
			incoming[i].ConnectionID = uuid.New()
		}
		if incoming[i].ResourceSetID == nil {
			defaultRS := scope.DefaultResourceSetID
			incoming[i].ResourceSetID = &defaultRS
		}
		incoming[i].Payload = redactPayload(incoming[i].Payload)
	}
	return b.workload.repo.InsertWorkloadEventBatch(ctx, incoming)
}

func (b *EventBus) Metrics() *Metrics {
	return b.metrics
}

// Shutdown signals the workload pipeline to stop accepting new events and
// flush whatever it's already buffered, then blocks until that drain
// completes or timeout elapses — whichever comes first. A timeout that
// elapses first is logged (rather than silently discarding buffered
// events) so an operator can tell shutdown didn't fully drain.
func (b *EventBus) Shutdown(timeout time.Duration) {
	b.cancel()
	select {
	case <-b.done:
	case <-time.After(timeout):
		if b.logger != nil {
			b.logger.Warn("event bus shutdown timed out before workload pipeline finished draining", "timeout", timeout)
		}
	}
}

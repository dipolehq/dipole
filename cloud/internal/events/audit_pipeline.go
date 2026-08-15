/*
audit_pipeline.go implements synchronous persistence for audit events.

Audit events record administrative mutations (entity CRUD). They are
compliance-critical but low-volume (admin actions, not per-request). Writes
happen inline and errors are returned to the caller with a metric increment.
*/
package events

import (
	"context"
	"log/slog"
	"time"
)

type AuditPipeline struct {
	repo    Repository
	logger  *slog.Logger
	metrics *Metrics
}

func newAuditPipeline(repo Repository, logger *slog.Logger, metrics *Metrics) *AuditPipeline {
	return &AuditPipeline{
		repo:    repo,
		logger:  logger,
		metrics: metrics,
	}
}

func (p *AuditPipeline) Emit(ctx context.Context, event AuditEvent) error {
	if p.repo == nil {
		return nil
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	if err := p.repo.InsertAuditEvent(ctx, &event); err != nil {
		p.logger.Error("audit event persistence failed",
			"category", event.Category,
			"activity", event.Activity,
			"tenant_id", event.TenantID,
			"error", err,
		)
		p.metrics.AuditFailed.Add(1)
		return err
	}

	p.metrics.AuditEmitted.Add(1)
	return nil
}

/*
security_pipeline.go implements synchronous persistence for security events.

Security events represent access decisions with compliance implications — a
lost event means a gap in the customer's audit trail. Writes happen inline in
the request path. If the DB write fails the error is returned to the caller and
a metric is incremented; access is never blocked by a logging failure.
*/
package events

import (
	"context"
	"log/slog"
	"time"

	"github.com/dipolehq/dipole/cloud/internal/scope"
	"github.com/google/uuid"
)

type SecurityPipeline struct {
	repo    Repository
	logger  *slog.Logger
	metrics *Metrics
}

func newSecurityPipeline(repo Repository, logger *slog.Logger, metrics *Metrics) *SecurityPipeline {
	return &SecurityPipeline{
		repo:    repo,
		logger:  logger,
		metrics: metrics,
	}
}

func (p *SecurityPipeline) Emit(ctx context.Context, event SecurityEvent) error {
	if p.repo == nil {
		return nil
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	if event.ContextID == uuid.Nil {
		event.ContextID = uuid.New()
	}
	if event.ResourceSetID == nil {
		// Every producer is expected to set ResourceSetID explicitly (see
		// evaluation.go, method_handler.go, mcp_gateway.go). This default is a
		// last-resort safety net, not the intended path — a producer reaching
		// here means it forgot to attribute the event to a resource set, so
		// it is logged loudly rather than silently defaulted.
		if p.logger != nil {
			p.logger.Warn("security event missing ResourceSetID, defaulting to DefaultResourceSetID",
				"event_type", event.EventType,
				"tenant_id", event.TenantID,
				"source", event.Source,
			)
		}
		defaultRS := scope.DefaultResourceSetID
		event.ResourceSetID = &defaultRS
	}

	if err := p.repo.InsertSecurityEvent(ctx, &event); err != nil {
		p.logger.Error("security event persistence failed",
			"event_type", event.EventType,
			"tenant_id", event.TenantID,
			"error", err,
		)
		p.metrics.SecurityFailed.Add(1)
		return err
	}

	p.metrics.SecurityEmitted.Add(1)
	return nil
}

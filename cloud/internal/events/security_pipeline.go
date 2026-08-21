// Package events implements synchronous persistence for security events.
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
		if p.logger != nil {
			p.logger.Error("security event persistence failed",
				"event_type", event.EventType,
				"tenant_id", event.TenantID,
				"error", err,
			)
		}
		p.metrics.SecurityFailed.Add(1)
		return err
	}

	p.metrics.SecurityEmitted.Add(1)
	return nil
}

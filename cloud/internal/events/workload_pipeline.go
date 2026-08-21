// Package events implements async buffered persistence for workload events.
package events

import (
	"context"
	"log/slog"
	"time"

	"github.com/dipolehq/dipole/cloud/internal/scope"
	"github.com/google/uuid"
)

const (
	workloadChannelSize   = 10000
	workloadBatchSize     = 50
	workloadFlushInterval = 100 * time.Millisecond
	workloadFlushTimeout  = 5 * time.Second
)

type WorkloadPipeline struct {
	ch      chan WorkloadEvent
	repo    Repository
	logger  *slog.Logger
	metrics *Metrics
}

func newWorkloadPipeline(repo Repository, logger *slog.Logger, metrics *Metrics) *WorkloadPipeline {
	return &WorkloadPipeline{
		ch:      make(chan WorkloadEvent, workloadChannelSize),
		repo:    repo,
		logger:  logger,
		metrics: metrics,
	}
}

func (p *WorkloadPipeline) Emit(event WorkloadEvent) {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	if event.EventID == uuid.Nil {
		event.EventID = uuid.New()
	}
	if event.ConnectionID == uuid.Nil {
		event.ConnectionID = uuid.New()
	}
	if event.ResourceSetID == nil {
		if p.logger != nil {
			p.logger.Warn("workload event missing ResourceSetID, defaulting to DefaultResourceSetID",
				"event_type", event.EventType,
				"tenant_id", event.TenantID,
			)
		}
		defaultRS := scope.DefaultResourceSetID
		event.ResourceSetID = &defaultRS
	}

	select {
	case p.ch <- event:
		p.metrics.WorkloadEmitted.Add(1)
	default:
		p.metrics.WorkloadDropped.Add(1)
		p.logger.Warn("workload event channel full, dropping event",
			"event_type", event.EventType,
			"tenant_id", event.TenantID,
			"connection_id", event.ConnectionID,
		)
	}
}

func (p *WorkloadPipeline) Run(ctx context.Context) {
	batch := make([]WorkloadEvent, 0, workloadBatchSize)
	ticker := time.NewTicker(workloadFlushInterval)
	defer ticker.Stop()

	for {
		select {
		case event, ok := <-p.ch:
			if !ok {
				if len(batch) > 0 {
					p.flush(batch)
				}
				return
			}
			batch = append(batch, event)
			if len(batch) >= workloadBatchSize {
				p.flush(batch)
				batch = batch[:0]
			}
		case <-ticker.C:
			if len(batch) > 0 {
				p.flush(batch)
				batch = batch[:0]
			}
			p.metrics.WorkloadChannelSize.Store(int64(len(p.ch)))
		case <-ctx.Done():
			for {
				select {
				case event := <-p.ch:
					batch = append(batch, event)
				default:
					if len(batch) > 0 {
						p.flush(batch)
					}
					return
				}
			}
		}
	}
}

func (p *WorkloadPipeline) flush(batch []WorkloadEvent) {
	if p.repo == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), workloadFlushTimeout)
	defer cancel()

	if err := p.repo.InsertWorkloadEventBatch(ctx, batch); err != nil {
		p.logger.Error("workload event batch persistence failed",
			"batch_size", len(batch),
			"error", err,
		)
		p.metrics.WorkloadBatchFailed.Add(1)
	}
}

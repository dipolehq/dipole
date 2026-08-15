/*
workload_pipeline.go implements async buffered persistence for workload events.

Workload events are high-volume connection-level telemetry (MCP requests,
proxy connections). They are batched in memory and flushed periodically or
when a batch-size threshold is reached. If the channel is full under extreme
load, events are dropped and a counter is incremented — workload events are
best-effort, never blocking the request path.
*/
package events

import (
	"context"
	"log/slog"
	"time"

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
	if event.ConnectionID == uuid.Nil {
		event.ConnectionID = uuid.New()
	}

	select {
	case p.ch <- event:
		p.metrics.WorkloadEmitted.Add(1)
	default:
		p.metrics.WorkloadDropped.Add(1)
		p.logger.Warn("workload event channel full, dropping event",
			"event_type", event.EventType,
			"tenant_id", event.TenantID,
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
			close(p.ch)
			for event := range p.ch {
				batch = append(batch, event)
			}
			if len(batch) > 0 {
				p.flush(batch)
			}
			return
		}
	}
}

func (p *WorkloadPipeline) flush(batch []WorkloadEvent) {
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

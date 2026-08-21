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
		// Safety net, not the intended path — see security_pipeline.go for why
		// this is logged rather than silently defaulted.
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
			// Drain whatever is already buffered via non-blocking
			// receives rather than closing p.ch: Emit's send is a
			// non-blocking select (case p.ch <- event: / default:),
			// and a send on a closed channel panics unconditionally
			// in Go — select does not route it to default. A caller
			// that's still mid-flight past this point (e.g. a request
			// handler that outlived shutdownHTTP's drain window) would
			// otherwise crash the process instead of simply losing an
			// event it's too late to process anyway.
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

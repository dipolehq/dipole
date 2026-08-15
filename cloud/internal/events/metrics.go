/*
metrics.go provides lightweight observability counters for event pipeline
operations.

Counters track emit success, failure, and drop rates across all four pipelines.
The implementation uses atomic int64 values. When a Prometheus /metrics endpoint
is added later, these can be backed by prometheus.Counter with zero API change
to callers — the Metrics struct is the single source of truth.
*/
package events

import "sync/atomic"

type Metrics struct {
	SecurityEmitted     atomic.Int64
	SecurityFailed      atomic.Int64
	WorkloadEmitted     atomic.Int64
	WorkloadDropped     atomic.Int64
	WorkloadBatchFailed atomic.Int64
	WorkloadChannelSize atomic.Int64
	TracingEmitted      atomic.Int64
	AuditEmitted        atomic.Int64
	AuditFailed         atomic.Int64
}

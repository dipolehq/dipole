// metrics.go provides lightweight atomic observability counters for emit success, failure, and drop rates across the Security, Workload, and Audit pipelines.
package events

import "sync/atomic"

type Metrics struct {
	SecurityEmitted     atomic.Int64
	SecurityFailed      atomic.Int64
	WorkloadEmitted     atomic.Int64
	WorkloadDropped     atomic.Int64
	WorkloadBatchFailed atomic.Int64
	WorkloadChannelSize atomic.Int64
	AuditEmitted        atomic.Int64
	AuditFailed         atomic.Int64
}

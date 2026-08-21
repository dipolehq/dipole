// Package events provides lightweight atomic counters for event emission observability.
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

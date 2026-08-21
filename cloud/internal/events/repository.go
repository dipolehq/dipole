// Package events defines the write-only persistence interface for event pipelines.
package events

import "context"

type Repository interface {
	InsertSecurityEvent(ctx context.Context, event *SecurityEvent) error
	InsertWorkloadEventBatch(ctx context.Context, events []WorkloadEvent) error
	InsertAuditEvent(ctx context.Context, event *AuditEvent) error
}

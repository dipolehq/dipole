// repository.go defines the write-only persistence interface consumed by the event pipelines and implemented by the postgres package; all reads go through postgres.EventRepo directly (see cloud/internal/repository/postgres/event.go), not through this interface.
package events

import "context"

type Repository interface {
	InsertSecurityEvent(ctx context.Context, event *SecurityEvent) error
	InsertWorkloadEventBatch(ctx context.Context, events []WorkloadEvent) error
	InsertAuditEvent(ctx context.Context, event *AuditEvent) error
}

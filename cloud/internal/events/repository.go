/*
repository.go defines the persistence interface consumed by the event pipelines.

This interface is implemented by the postgres package and represents the single
contract for all event table operations. Query methods are used by the REST
handlers that serve the frontend reporting pages.
*/
package events

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	InsertSecurityEvent(ctx context.Context, event *SecurityEvent) error
	InsertWorkloadEventBatch(ctx context.Context, events []WorkloadEvent) error
	InsertAuditEvent(ctx context.Context, event *AuditEvent) error
	QuerySecurityEvents(ctx context.Context, tenantID uuid.UUID, opts QueryOptions) ([]SecurityEvent, int, error)
	QueryWorkloadEvents(ctx context.Context, tenantID uuid.UUID, opts QueryOptions) ([]WorkloadEvent, int, error)
	QueryAuditEvents(ctx context.Context, tenantID uuid.UUID, opts QueryOptions) ([]AuditEvent, int, error)
}

type QueryOptions struct {
	Page              int
	PerPage           int
	Search            string
	StartDate         *time.Time
	EndDate           *time.Time
	Severity          string
	Severities        []string
	EventType         string
	Source            string
	ClientWorkloadIDs []string
	ServerWorkloadIDs []string
	UserID            string
	ResourceSetID     *uuid.UUID
}

/*
event_sink.go persists MCP workload events to the database.

DBEventSink implements the EventSink interface by converting MCPWorkloadEvent
to domain.WorkloadEvent and persisting via the EventRepository. Emission is
fire-and-forget to avoid blocking the request path.
*/
package mcp_gateway

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
)

// DBEventSink holds no tenant. It is shared by every tenant's Gateway traffic,
// and each event is attributed to the tenant supplied by the caller — previously
// one tenant was captured at construction, so every tenant's MCP events were
// written under it.
type DBEventSink struct {
	events repository.EventRepository
	logger *slog.Logger
}

func NewDBEventSink(events repository.EventRepository, logger *slog.Logger) *DBEventSink {
	return &DBEventSink{
		events: events,
		logger: logger,
	}
}

func (s *DBEventSink) Emit(tenantID uuid.UUID, event *MCPWorkloadEvent) {
	// Dropped rather than written to an arbitrary tenant: an unattributed event
	// is worse than a missing one, because it corrupts another tenant's audit
	// trail.
	if tenantID == uuid.Nil {
		s.logger.Warn("dropping MCP workload event with no tenant", "event_type", event.Meta.EventType)
		return
	}
	go s.persist(tenantID, event)
}

func (s *DBEventSink) persist(tenantID uuid.UUID, event *MCPWorkloadEvent) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	connID, _ := uuid.Parse(event.Meta.ConnectionID)
	if connID == uuid.Nil {
		connID = uuid.New()
	}

	timestamp, err := time.Parse(time.RFC3339Nano, event.Meta.Timestamp)
	if err != nil {
		timestamp = time.Now().UTC()
	}

	payload, _ := json.Marshal(event.Application)

	serverID, _ := uuid.Parse(event.Workload.Server.ID)

	we := &domain.WorkloadEvent{
		ID:               uuid.New(),
		TenantID:         tenantID,
		ConnectionID:     connID,
		EventType:        event.Meta.EventType,
		ServerWorkloadID: uuidPtr(serverID),
		Outcome:          event.Outcome.Result,
		Severity:         event.Meta.Severity,
		Timestamp:        timestamp,
		Payload:          payload,
	}

	if err := s.events.InsertWorkloadEvent(ctx, tenantID, we); err != nil {
		s.logger.Debug("failed to persist MCP workload event", "error", err)
	}
}

func uuidPtr(id uuid.UUID) *uuid.UUID {
	if id == uuid.Nil {
		return nil
	}
	return &id
}

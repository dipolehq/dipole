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

type DBEventSink struct {
	events   repository.EventRepository
	tenantID uuid.UUID
	logger   *slog.Logger
}

func NewDBEventSink(events repository.EventRepository, tenantID uuid.UUID, logger *slog.Logger) *DBEventSink {
	return &DBEventSink{
		events:   events,
		tenantID: tenantID,
		logger:   logger,
	}
}

func (s *DBEventSink) Emit(event *MCPWorkloadEvent) {
	go s.persist(event)
}

func (s *DBEventSink) persist(event *MCPWorkloadEvent) {
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
		TenantID:         s.tenantID,
		ConnectionID:     connID,
		EventType:        event.Meta.EventType,
		ServerWorkloadID: uuidPtr(serverID),
		Outcome:          event.Outcome.Result,
		Severity:         event.Meta.Severity,
		Timestamp:        timestamp,
		Payload:          payload,
	}

	if err := s.events.InsertWorkloadEvent(ctx, s.tenantID, we); err != nil {
		s.logger.Debug("failed to persist MCP workload event", "error", err)
	}
}

func uuidPtr(id uuid.UUID) *uuid.UUID {
	if id == uuid.Nil {
		return nil
	}
	return &id
}

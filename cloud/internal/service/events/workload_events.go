/*
workload_events.go implements Workload Event ingestion from the Agent Proxy.

WorkloadEventIngester receives events streamed via gRPC, redacts sensitive
headers (Authorization, Set-Cookie, Cookie, X-API-Key) from HTTP/MCP payloads
before storage, and writes to the workload_events table. Events include protocol-specific
application data (HTTP methods, DB queries, MCP tool calls, etc.).
*/
package events

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
)

type WorkloadEventIngester struct {
	repo repository.EventRepository
}

func NewWorkloadEventIngester(repo repository.EventRepository) *WorkloadEventIngester {
	return &WorkloadEventIngester{repo: repo}
}

type IncomingWorkloadEvent struct {
	TenantID         uuid.UUID
	EventType        string
	ConnectionID     uuid.UUID
	PolicyID         *uuid.UUID
	ClientWorkloadID *uuid.UUID
	ServerWorkloadID *uuid.UUID
	Severity         string
	Outcome          string
	Timestamp        time.Time
	Payload          json.RawMessage
	ResourceSetID    *uuid.UUID
}

func (i *WorkloadEventIngester) Ingest(ctx context.Context, event *IncomingWorkloadEvent) error {
	redactedPayload := redactSensitiveHeaders(event.Payload)

	return i.repo.InsertWorkloadEvent(ctx, event.TenantID, &domain.WorkloadEvent{
		ID:               uuid.New(),
		TenantID:         event.TenantID,
		ConnectionID:     event.ConnectionID,
		EventType:        event.EventType,
		PolicyID:         event.PolicyID,
		ClientWorkloadID: event.ClientWorkloadID,
		ServerWorkloadID: event.ServerWorkloadID,
		Outcome:          event.Outcome,
		Severity:         event.Severity,
		Timestamp:        event.Timestamp,
		Payload:          redactedPayload,
		ResourceSetID:    event.ResourceSetID,
	})
}

func (i *WorkloadEventIngester) IngestBatch(ctx context.Context, events []*IncomingWorkloadEvent) error {
	for _, event := range events {
		if err := i.Ingest(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

func redactSensitiveHeaders(payload json.RawMessage) json.RawMessage {
	if len(payload) == 0 {
		return payload
	}

	var data map[string]any
	if err := json.Unmarshal(payload, &data); err != nil {
		return payload
	}

	modified := false

	if app, ok := data["application"].(map[string]any); ok {
		if http, ok := app["http"].(map[string]any); ok {
			if headers, ok := http["headers"].([]any); ok {
				http["headers"] = redactHeaders(headers)
				modified = true
			}
		}
		if mcp, ok := app["mcp"].(map[string]any); ok {
			if headers, ok := mcp["headers"].([]any); ok {
				mcp["headers"] = redactHeaders(headers)
				modified = true
			}
		}
	}

	if !modified {
		return payload
	}

	result, err := json.Marshal(data)
	if err != nil {
		return payload
	}
	return result
}

func redactHeaders(headers []any) []any {
	redacted := make([]any, 0, len(headers))
	for _, h := range headers {
		header, ok := h.(map[string]any)
		if !ok {
			redacted = append(redacted, h)
			continue
		}

		shouldRedact := false
		for key := range header {
			lower := strings.ToLower(key)
			if lower == "authorization" || lower == "set-cookie" || lower == "cookie" || lower == "x-api-key" {
				shouldRedact = true
				break
			}
		}

		if shouldRedact {
			redacted = append(redacted, map[string]any{"[REDACTED]": "[REDACTED]"})
		} else {
			redacted = append(redacted, header)
		}
	}
	return redacted
}

/*
event_sink_test.go tests the DBEventSink that converts MCPWorkloadEvent to
domain.WorkloadEvent and persists them via EventRepository.
*/
package mcp_gateway

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type mockEventRepo struct {
	mu           sync.Mutex
	lastEvent    *domain.WorkloadEvent
	lastTenantID uuid.UUID
	callCount    int
	events       []*domain.WorkloadEvent
}

func (m *mockEventRepo) InsertWorkloadEvent(_ context.Context, tenantID uuid.UUID, event *domain.WorkloadEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastTenantID = tenantID
	m.lastEvent = event
	m.callCount++
	m.events = append(m.events, event)
	return nil
}

// count and all give tests the full history, which asserting per-tenant
// attribution needs — the "last" fields alone can't show two tenants.
func (m *mockEventRepo) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.events)
}

func (m *mockEventRepo) all() []*domain.WorkloadEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*domain.WorkloadEvent, len(m.events))
	copy(out, m.events)
	return out
}

func (m *mockEventRepo) InsertAuthEvent(_ context.Context, _ uuid.UUID, _ *domain.AuthEvent) error {
	panic("not used")
}

func (m *mockEventRepo) InsertAuditLog(_ context.Context, _ uuid.UUID, _ *domain.AuditLog) error {
	panic("not used")
}

func (m *mockEventRepo) QueryAuthEvents(_ context.Context, _ uuid.UUID, _ domain.EventQueryOptions) ([]domain.AuthEvent, int, error) {
	panic("not used")
}

func (m *mockEventRepo) QueryWorkloadEvents(_ context.Context, _ uuid.UUID, _ domain.EventQueryOptions) ([]domain.WorkloadEvent, int, error) {
	panic("not used")
}

func (m *mockEventRepo) QueryAuditLogs(_ context.Context, _ uuid.UUID, _ domain.EventQueryOptions) ([]domain.AuditLog, int, error) {
	panic("not used")
}

func (m *mockEventRepo) DashboardEventStats(_ context.Context, _ uuid.UUID, _ string) (*domain.DashboardEventData, error) {
	panic("not used")
}

func TestDBEventSink_Emit_PersistsEvent(t *testing.T) {
	tenantID := uuid.New()
	serverID := uuid.New()
	connID := uuid.New()

	repo := &mockEventRepo{}
	logger := slog.Default()
	sink := NewDBEventSink(repo, logger)

	event := &MCPWorkloadEvent{
		Meta: EventMeta{
			EventType:    "mcp.request",
			ConnectionID: connID.String(),
			Severity:     "Info",
			Timestamp:    time.Now().UTC().Format(time.RFC3339Nano),
		},
		Workload: EventWorkload{
			Client: EventActor{Name: "MCP Client"},
			Server: EventActor{ID: serverID.String(), Name: "jira"},
		},
		Outcome: EventOutcome{Result: "Modified"},
		Application: EventApplication{
			MCP: EventMCP{
				HTTPMethod: "POST",
				URI:        "/mcp",
				UserID:     "alice@corp.com",
			},
		},
	}

	sink.Emit(tenantID, event)

	// Wait for fire-and-forget goroutine to complete
	time.Sleep(100 * time.Millisecond)

	repo.mu.Lock()
	defer repo.mu.Unlock()

	if repo.callCount != 1 {
		t.Fatalf("expected InsertWorkloadEvent to be called once, got %d", repo.callCount)
	}
	if repo.lastTenantID != tenantID {
		t.Errorf("expected tenantID %s, got %s", tenantID, repo.lastTenantID)
	}
	if repo.lastEvent == nil {
		t.Fatal("expected non-nil event")
	}
	if repo.lastEvent.EventType != "mcp.request" {
		t.Errorf("expected event type 'mcp.request', got '%s'", repo.lastEvent.EventType)
	}
	if repo.lastEvent.Outcome != "Modified" {
		t.Errorf("expected outcome 'Modified', got '%s'", repo.lastEvent.Outcome)
	}
	if repo.lastEvent.Severity != "Info" {
		t.Errorf("expected severity 'Info', got '%s'", repo.lastEvent.Severity)
	}
	if repo.lastEvent.ConnectionID != connID {
		t.Errorf("expected connection ID %s, got %s", connID, repo.lastEvent.ConnectionID)
	}
	if repo.lastEvent.ServerWorkloadID == nil || *repo.lastEvent.ServerWorkloadID != serverID {
		t.Errorf("expected server workload ID %s", serverID)
	}
}

func TestDBEventSink_Emit_NilEvent_NoPanic(t *testing.T) {
	tenantID := uuid.New()
	repo := &mockEventRepo{}
	logger := slog.Default()
	sink := NewDBEventSink(repo, logger)

	// Emit with minimal data (edge case: empty connection ID, no server ID)
	event := &MCPWorkloadEvent{
		Meta: EventMeta{
			EventType:    "mcp.response",
			ConnectionID: "", // empty — should be handled gracefully
			Severity:     "Info",
			Timestamp:    "invalid-timestamp",
		},
		Workload: EventWorkload{
			Client: EventActor{Name: "test"},
			Server: EventActor{ID: "", Name: "test-server"},
		},
		Outcome: EventOutcome{Result: "Error"},
		Application: EventApplication{
			MCP: EventMCP{UserID: "user@test.com"},
		},
	}

	// Should not panic
	sink.Emit(tenantID, event)

	time.Sleep(100 * time.Millisecond)

	repo.mu.Lock()
	defer repo.mu.Unlock()

	if repo.callCount != 1 {
		t.Fatalf("expected InsertWorkloadEvent to be called once, got %d", repo.callCount)
	}
	// The empty connection ID should be replaced with a new UUID
	if repo.lastEvent.ConnectionID == uuid.Nil {
		t.Error("expected non-nil connection ID for empty input")
	}
	// Invalid timestamp should fall back to current time
	if repo.lastEvent.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
}

// Events are attributed to the tenant the caller supplies. The sink is shared by
// every tenant's Gateway traffic, so a captured tenant would file all of them
// under one account's audit trail.
func TestDBEventSink_AttributesEventToSuppliedTenant(t *testing.T) {
	repo := &mockEventRepo{}
	sink := NewDBEventSink(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	tenantA, tenantB := uuid.New(), uuid.New()
	sink.Emit(tenantA, &MCPWorkloadEvent{Meta: EventMeta{EventType: "mcp.request", Severity: "Info"}})
	sink.Emit(tenantB, &MCPWorkloadEvent{Meta: EventMeta{EventType: "mcp.response", Severity: "Info"}})

	waitFor(t, func() bool { return repo.count() == 2 })

	seen := map[uuid.UUID]string{}
	for _, e := range repo.all() {
		seen[e.TenantID] = e.EventType
	}
	if seen[tenantA] != "mcp.request" {
		t.Errorf("expected tenant A's request event, got %q", seen[tenantA])
	}
	if seen[tenantB] != "mcp.response" {
		t.Errorf("expected tenant B's response event, got %q", seen[tenantB])
	}
}

// An unattributed event is dropped rather than filed under an arbitrary tenant:
// a wrong-tenant audit record is worse than a missing one.
func TestDBEventSink_DropsEventWithNoTenant(t *testing.T) {
	repo := &mockEventRepo{}
	sink := NewDBEventSink(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	sink.Emit(uuid.Nil, &MCPWorkloadEvent{Meta: EventMeta{EventType: "mcp.request", Severity: "Info"}})

	time.Sleep(100 * time.Millisecond)
	if got := repo.count(); got != 0 {
		t.Errorf("expected the event to be dropped, got %d persisted", got)
	}
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition not met within timeout")
}

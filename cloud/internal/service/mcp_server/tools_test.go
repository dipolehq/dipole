package mcp_server

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

func TestListTools_Returns3Tools(t *testing.T) {
	svc := NewToolService(nil, nil)
	result := svc.ListTools()

	resp, ok := result.(map[string]any)
	if !ok {
		t.Fatal("expected map result")
	}
	tools, ok := resp["tools"].([]map[string]any)
	if !ok {
		t.Fatal("expected tools array")
	}
	if len(tools) != 3 {
		t.Fatalf("expected 3 tools, got %d", len(tools))
	}

	names := map[string]bool{}
	for _, tool := range tools {
		names[tool["name"].(string)] = true
	}
	if !names["get_audit_logs"] {
		t.Error("missing get_audit_logs")
	}
	if !names["get_auth_events"] {
		t.Error("missing get_auth_events")
	}
	if !names["get_workload_events"] {
		t.Error("missing get_workload_events")
	}
}

func TestPerPageCap(t *testing.T) {
	if capPerPage(500) != 100 {
		t.Error("expected 500 to be capped to 100")
	}
	if capPerPage(50) != 50 {
		t.Error("expected 50 to stay 50")
	}
	if capPerPage(0) != 25 {
		t.Error("expected 0 to default to 25")
	}
	if capPerPage(100) != 100 {
		t.Error("expected 100 to stay 100")
	}
}

func TestDateRangePriority_StartEndOverridesSpan(t *testing.T) {
	args := map[string]any{
		"startDate":       "2025-01-01T00:00:00Z",
		"endDate":         "2025-01-02T00:00:00Z",
		"spanLastMinutes": float64(60),
		"spanLastDays":    float64(7),
	}

	start, end := resolveAuditDateRange(args)
	if start == nil || end == nil {
		t.Fatal("expected non-nil dates")
	}
	if start.Year() != 2025 || start.Month() != 1 || start.Day() != 1 {
		t.Errorf("expected startDate 2025-01-01, got %v", start)
	}
}

func TestDateRangePriority_SpanMinutesOverridesDays(t *testing.T) {
	args := map[string]any{
		"spanLastMinutes": float64(30),
		"spanLastDays":    float64(7),
	}

	start, end := resolveAuditDateRange(args)
	if start == nil || end == nil {
		t.Fatal("expected non-nil dates")
	}

	duration := end.Sub(*start)
	if duration < 29*time.Minute || duration > 31*time.Minute {
		t.Errorf("expected ~30 minutes range, got %v", duration)
	}
}

func TestDateRangePriority_DefaultSpanDays(t *testing.T) {
	args := map[string]any{}

	start, end := resolveAuditDateRange(args)
	if start == nil || end == nil {
		t.Fatal("expected non-nil dates")
	}

	duration := end.Sub(*start)
	if duration < 29*24*time.Hour || duration > 31*24*time.Hour {
		t.Errorf("expected ~30 days range, got %v", duration)
	}
}

func TestEndDateClampedToNow(t *testing.T) {
	future := time.Now().UTC().Add(48 * time.Hour).Format(time.RFC3339)
	args := map[string]any{
		"startDate": "2025-01-01T00:00:00Z",
		"endDate":   future,
	}

	_, end := resolveAuditDateRange(args)
	if end == nil {
		t.Fatal("expected non-nil end date")
	}
	if end.After(time.Now().UTC().Add(1 * time.Second)) {
		t.Error("endDate should be clamped to now")
	}
}

func TestMapEventType(t *testing.T) {
	cases := []struct {
		input, expected string
	}{
		{"Request", "access.request"},
		{"Authorization", "access.authorization"},
		{"Credential", "access.credential"},
		{"Unknown", "Unknown"},
	}
	for _, tc := range cases {
		if got := mapEventType(tc.input); got != tc.expected {
			t.Errorf("mapEventType(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestAuditLogger_LogsToolCall(t *testing.T) {
	repo := &mockEventRepo{}
	logger := NewAuditLogger(repo)
	tenantID := uuid.New()

	logger.LogToolCall(t.Context(), tenantID, "get_auth_events", "user@corp.com")

	if len(repo.auditLogs) != 1 {
		t.Fatalf("expected 1 audit log, got %d", len(repo.auditLogs))
	}

	log := repo.auditLogs[0]
	if log.Category != "Analysis" {
		t.Errorf("expected category=Analysis, got %s", log.Category)
	}
	if log.Activity != "QueryAuthEvents" {
		t.Errorf("expected activity=QueryAuthEvents, got %s", log.Activity)
	}
	if log.TargetType != "mcp_server_query" {
		t.Errorf("expected target_type=mcp_server_query, got %s", log.TargetType)
	}
	if log.Severity != "Info" {
		t.Errorf("expected severity=Info, got %s", log.Severity)
	}
	if log.ActorEmail != "user@corp.com" {
		t.Errorf("expected actorEmail=user@corp.com, got %s", log.ActorEmail)
	}
}

type mockEventRepo struct {
	auditLogs      []*domain.AuditLog
	authEvents     []domain.AuthEvent
	workloadEvents []domain.WorkloadEvent
}

func (r *mockEventRepo) InsertAuthEvent(_ context.Context, _ uuid.UUID, _ *domain.AuthEvent) error {
	return nil
}
func (r *mockEventRepo) InsertWorkloadEvent(_ context.Context, _ uuid.UUID, _ *domain.WorkloadEvent) error {
	return nil
}
func (r *mockEventRepo) InsertAuditLog(_ context.Context, _ uuid.UUID, log *domain.AuditLog) error {
	r.auditLogs = append(r.auditLogs, log)
	return nil
}
func (r *mockEventRepo) QueryAuthEvents(_ context.Context, _ uuid.UUID, _ domain.EventQueryOptions) ([]domain.AuthEvent, int, error) {
	return r.authEvents, len(r.authEvents), nil
}
func (r *mockEventRepo) QueryWorkloadEvents(_ context.Context, _ uuid.UUID, _ domain.EventQueryOptions) ([]domain.WorkloadEvent, int, error) {
	return r.workloadEvents, len(r.workloadEvents), nil
}
func (r *mockEventRepo) QueryAuditLogs(_ context.Context, _ uuid.UUID, _ domain.EventQueryOptions) ([]domain.AuditLog, int, error) {
	var results []domain.AuditLog
	for _, l := range r.auditLogs {
		results = append(results, *l)
	}
	return results, len(results), nil
}
func (r *mockEventRepo) DashboardEventStats(_ context.Context, _ uuid.UUID, _ string) (*domain.DashboardEventData, error) {
	return &domain.DashboardEventData{}, nil
}

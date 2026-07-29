package events

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type mockEventRepo struct {
	authEvents     []*domain.AuthEvent
	workloadEvents []*domain.WorkloadEvent
	auditLogs      []*domain.AuditLog
}

func (r *mockEventRepo) InsertAuthEvent(_ context.Context, _ uuid.UUID, e *domain.AuthEvent) error {
	r.authEvents = append(r.authEvents, e)
	return nil
}
func (r *mockEventRepo) InsertWorkloadEvent(_ context.Context, _ uuid.UUID, e *domain.WorkloadEvent) error {
	r.workloadEvents = append(r.workloadEvents, e)
	return nil
}
func (r *mockEventRepo) InsertAuditLog(_ context.Context, _ uuid.UUID, l *domain.AuditLog) error {
	r.auditLogs = append(r.auditLogs, l)
	return nil
}
func (r *mockEventRepo) QueryAuthEvents(_ context.Context, _ uuid.UUID, _ domain.EventQueryOptions) ([]domain.AuthEvent, int, error) {
	return nil, 0, nil
}
func (r *mockEventRepo) QueryWorkloadEvents(_ context.Context, _ uuid.UUID, _ domain.EventQueryOptions) ([]domain.WorkloadEvent, int, error) {
	return nil, 0, nil
}
func (r *mockEventRepo) QueryAuditLogs(_ context.Context, _ uuid.UUID, _ domain.EventQueryOptions) ([]domain.AuditLog, int, error) {
	return nil, 0, nil
}
func (r *mockEventRepo) DashboardEventStats(_ context.Context, _ uuid.UUID) (*domain.DashboardEventData, error) {
	return &domain.DashboardEventData{}, nil
}

func TestAuthEventEmitter_EmitsAllFourTypes(t *testing.T) {
	repo := &mockEventRepo{}
	emitter := NewAuthEventEmitter(repo)

	tenantID := uuid.New()
	cwID := uuid.New()
	swID := uuid.New()
	policyID := uuid.New()
	cpID := uuid.New()

	eval := &EvaluationContext{
		TenantID:   tenantID,
		SourceIP:   "10.0.0.1",
		TargetHost: "api.stripe.com",
		TargetPort: 443,
		Result: &domain.EvaluationResult{
			Allowed:    true,
			Credential: &domain.Credential{Value: "token"},
		},
		ClientWorkload:           &domain.ClientWorkload{ID: cwID, Name: "MyApp"},
		ServerWorkload:           &domain.ServerWorkload{ID: swID, Name: "Stripe"},
		Policy:                   &domain.AccessPolicy{ID: policyID, Name: "Allow"},
		TrustProviders:           []domain.TrustProvider{},
		CredentialProvider:       &domain.CredentialProvider{ID: cpID, Name: "Stripe Key", Type: "api_key"},
		CandidateClientWorkloads: []domain.ClientWorkload{{ID: cwID, Name: "MyApp"}},
		CandidateServerWorkloads: []domain.ServerWorkload{{ID: swID, Name: "Stripe"}},
	}

	err := emitter.Emit(context.Background(), eval)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repo.authEvents) != 4 {
		t.Fatalf("expected 4 events, got %d", len(repo.authEvents))
	}

	// Verify event types
	types := map[string]bool{}
	for _, e := range repo.authEvents {
		types[e.EventType] = true
	}
	if !types["access.request"] {
		t.Error("missing access.request")
	}
	if !types["access.authorization"] {
		t.Error("missing access.authorization")
	}
	if !types["access.credential"] {
		t.Error("missing access.credential")
	}
	if !types["access.discovery"] {
		t.Error("missing access.discovery")
	}

	// Verify shared contextId
	contextID := repo.authEvents[0].ContextID
	for _, e := range repo.authEvents[1:] {
		if e.ContextID != contextID {
			t.Error("events do not share contextId")
		}
	}
}

func TestAuthEventEmitter_DeniedHasSeverityWarn(t *testing.T) {
	repo := &mockEventRepo{}
	emitter := NewAuthEventEmitter(repo)

	eval := &EvaluationContext{
		TenantID:   uuid.New(),
		SourceIP:   "10.0.0.1",
		TargetHost: "api.example.com",
		TargetPort: 443,
		Result:     &domain.EvaluationResult{Allowed: false, DenyReason: "AttestationFailed"},
	}

	emitter.Emit(context.Background(), eval)

	// access.authorization event should have severity=Warn
	for _, e := range repo.authEvents {
		if e.EventType == "access.authorization" {
			if e.Severity != "Warn" {
				t.Errorf("expected severity=Warn for denied, got %s", e.Severity)
			}
		}
	}
}

func TestWorkloadEventIngester_RedactsSensitiveHeaders(t *testing.T) {
	repo := &mockEventRepo{}
	ingester := NewWorkloadEventIngester(repo)

	payload := json.RawMessage(`{"application":{"http":{"headers":[{"content-type":"application/json"},{"authorization":"Bearer secret"},{"set-cookie":"session=abc"},{"cookie":"session_id=s3cr3t"},{"x-api-key":"sk_live_key123"}]}}}`)

	err := ingester.Ingest(context.Background(), &IncomingWorkloadEvent{
		TenantID:     uuid.New(),
		EventType:    "http.request",
		ConnectionID: uuid.New(),
		Severity:     "Info",
		Outcome:      "Modified",
		Payload:      payload,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repo.workloadEvents) != 1 {
		t.Fatal("expected 1 event")
	}

	stored := string(repo.workloadEvents[0].Payload)
	if !contains(stored, "content-type") {
		t.Error("content-type should be preserved")
	}
	if contains(stored, "Bearer secret") {
		t.Error("authorization header should be redacted")
	}
	if contains(stored, "session=abc") {
		t.Error("set-cookie header should be redacted")
	}
	if contains(stored, "s3cr3t") {
		t.Error("cookie header should be redacted")
	}
	if contains(stored, "sk_live_key123") {
		t.Error("x-api-key header should be redacted")
	}
	if !contains(stored, "[REDACTED]") {
		t.Error("expected [REDACTED] placeholder")
	}
}

func TestAuditLogEmitter_LogsCreated(t *testing.T) {
	repo := &mockEventRepo{}
	emitter := NewAuditLogEmitter(repo)

	tenantID := uuid.New()
	targetID := uuid.New()

	emitter.Created(context.Background(), tenantID, "admin@corp.com", "Workloads", "client_workload", targetID, "MyApp", "1.2.3.4", "Mozilla/5.0 Chrome/120")

	if len(repo.auditLogs) != 1 {
		t.Fatal("expected 1 audit log")
	}

	log := repo.auditLogs[0]
	if log.Activity != "Created" {
		t.Errorf("expected activity=Created, got %s", log.Activity)
	}
	if log.Category != "Workloads" {
		t.Errorf("expected category=Workloads, got %s", log.Category)
	}
	if log.ActorEmail != "admin@corp.com" {
		t.Errorf("expected actor=admin@corp.com, got %s", log.ActorEmail)
	}
	if log.Result != "Success" {
		t.Errorf("expected result=Success, got %s", log.Result)
	}
	if log.Browser != "Chrome" {
		t.Errorf("expected browser=Chrome, got %s", log.Browser)
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

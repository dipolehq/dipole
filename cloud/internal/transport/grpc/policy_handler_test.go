// policy_handler_test.go tests the gRPC PolicyEvaluationServer transport handlers, specifically the ReportHealth RPC and the mapWorkloadEvent mapping function.
package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/google/uuid"

	pb "github.com/dipolehq/dipole/gen/dipole/v1"
)

type mockHealthUpdater struct {
	calledTenantID uuid.UUID
	calledProxyID  uuid.UUID
	calledStatus   string
	err            error
}

func (m *mockHealthUpdater) UpdateHealth(_ context.Context, tenantID uuid.UUID, proxyID uuid.UUID, status string) error {
	m.calledTenantID = tenantID
	m.calledProxyID = proxyID
	m.calledStatus = status
	return m.err
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestReportHealth_UpdatesProxyStatus(t *testing.T) {
	healthUpdater := &mockHealthUpdater{}
	server := &PolicyEvaluationServer{
		healthUpdater: healthUpdater,
		logger:        newTestLogger(),
	}

	tenantID := uuid.New()
	proxyID := uuid.New()

	ctx := withVerifiedProxy(context.Background(), tenantID, proxyID)
	resp, err := server.ReportHealth(ctx, &pb.HealthReport{
		ComponentId: proxyID.String(),
		TenantId:    tenantID.String(),
		Status:      "Healthy",
		Timestamp:   1700000000,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Acknowledged {
		t.Error("expected Acknowledged=true")
	}
	if healthUpdater.calledTenantID != tenantID {
		t.Errorf("expected tenantID %s, got %s", tenantID, healthUpdater.calledTenantID)
	}
	if healthUpdater.calledProxyID != proxyID {
		t.Errorf("expected proxyID %s, got %s", proxyID, healthUpdater.calledProxyID)
	}
	if healthUpdater.calledStatus != "Healthy" {
		t.Errorf("expected status 'Healthy', got '%s'", healthUpdater.calledStatus)
	}
}

func TestReportHealth_NoVerifiedIdentity_NotAcknowledged(t *testing.T) {
	server := &PolicyEvaluationServer{
		healthUpdater: &mockHealthUpdater{},
		logger:        newTestLogger(),
	}

	resp, err := server.ReportHealth(context.Background(), &pb.HealthReport{
		ComponentId: uuid.New().String(),
		TenantId:    uuid.New().String(),
		Status:      "Healthy",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Acknowledged {
		t.Error("expected Acknowledged=false with no verified identity in context")
	}
}

func TestReportHealth_ComponentIDMismatch_NotAcknowledged(t *testing.T) {
	healthUpdater := &mockHealthUpdater{}
	server := &PolicyEvaluationServer{
		healthUpdater: healthUpdater,
		logger:        newTestLogger(),
	}

	tenantID := uuid.New()
	proxyID := uuid.New()
	ctx := withVerifiedProxy(context.Background(), tenantID, proxyID)

	resp, err := server.ReportHealth(ctx, &pb.HealthReport{
		ComponentId: uuid.New().String(),
		TenantId:    tenantID.String(),
		Status:      "Healthy",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Acknowledged {
		t.Error("expected Acknowledged=false when component_id does not match verified session identity")
	}
	if healthUpdater.calledProxyID != uuid.Nil {
		t.Error("UpdateHealth must not be called when identity does not match")
	}
}

func TestReportHealth_UpdateFails_NotAcknowledged(t *testing.T) {
	healthUpdater := &mockHealthUpdater{err: fmt.Errorf("db connection lost")}
	server := &PolicyEvaluationServer{
		healthUpdater: healthUpdater,
		logger:        newTestLogger(),
	}

	tenantID := uuid.New()
	proxyID := uuid.New()
	ctx := withVerifiedProxy(context.Background(), tenantID, proxyID)

	resp, err := server.ReportHealth(ctx, &pb.HealthReport{
		ComponentId: proxyID.String(),
		TenantId:    tenantID.String(),
		Status:      "Healthy",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Acknowledged {
		t.Error("expected Acknowledged=false when updater fails")
	}
}

func TestMapWorkloadEvent_MapsAllFields(t *testing.T) {

	eventID := uuid.New()
	connID := uuid.New()
	policyID := uuid.New()
	cwID := uuid.New()
	swID := uuid.New()
	rsID := uuid.New()

	ev := &pb.WorkloadEvent{
		EventId:          eventID.String(),
		ConnectionId:     connID.String(),
		EventType:        "http.request",
		PolicyId:         policyID.String(),
		ClientWorkloadId: cwID.String(),
		ServerWorkloadId: swID.String(),
		ResourceSetId:    rsID.String(),
		Outcome:          "Modified",
		Severity:         "Info",
		Timestamp:        1700000000,
		Payload:          []byte(`{"network":{"serverWorkloadHost":"api.stripe.com"}}`),
	}

	incoming := mapWorkloadEvent(ev)

	if incoming.EventID != eventID {
		t.Errorf("eventID = %s, want %s", incoming.EventID, eventID)
	}
	if incoming.ConnectionID != connID {
		t.Errorf("connectionID = %s, want %s", incoming.ConnectionID, connID)
	}
	if incoming.EventType != "http.request" {
		t.Errorf("eventType = %q, want %q", incoming.EventType, "http.request")
	}
	if incoming.PolicyID == nil || *incoming.PolicyID != policyID {
		t.Errorf("policyID = %v, want %s", incoming.PolicyID, policyID)
	}
	if incoming.ClientWorkloadID == nil || *incoming.ClientWorkloadID != cwID {
		t.Errorf("clientWorkloadID = %v, want %s", incoming.ClientWorkloadID, cwID)
	}
	if incoming.ServerWorkloadID == nil || *incoming.ServerWorkloadID != swID {
		t.Errorf("serverWorkloadID = %v, want %s", incoming.ServerWorkloadID, swID)
	}
	if incoming.ResourceSetID == nil || *incoming.ResourceSetID != rsID {
		t.Errorf("resourceSetID = %v, want %s", incoming.ResourceSetID, rsID)
	}
	if incoming.Outcome != "Modified" {
		t.Errorf("outcome = %q, want %q", incoming.Outcome, "Modified")
	}
	if incoming.Severity != "Info" {
		t.Errorf("severity = %q, want %q", incoming.Severity, "Info")
	}
	if incoming.Timestamp.Unix() != 1700000000 {
		t.Errorf("timestamp = %d, want 1700000000", incoming.Timestamp.Unix())
	}
	if string(incoming.Payload) != `{"network":{"serverWorkloadHost":"api.stripe.com"}}` {
		t.Errorf("payload mismatch: %s", string(incoming.Payload))
	}
}

func TestMapWorkloadEvent_EmptyOptionalFields(t *testing.T) {

	ev := &pb.WorkloadEvent{
		EventType: "tcp.open",
		Outcome:   "Passthrough",
		Severity:  "Info",
		Payload:   []byte(`{}`),
	}

	incoming := mapWorkloadEvent(ev)

	if incoming.EventID != uuid.Nil {
		t.Errorf("eventID should be zero-value UUID for empty string, got %s", incoming.EventID)
	}
	if incoming.PolicyID != nil {
		t.Error("policyID should be nil for empty string")
	}
	if incoming.ClientWorkloadID != nil {
		t.Error("clientWorkloadID should be nil for empty string")
	}
	if incoming.ServerWorkloadID != nil {
		t.Error("serverWorkloadID should be nil for empty string")
	}
	if incoming.ResourceSetID != nil {
		t.Error("resourceSetID should be nil for empty string")
	}
	if incoming.ConnectionID != uuid.Nil {
		t.Errorf("connectionID should be zero-value UUID for empty string, got %s", incoming.ConnectionID)
	}
	if incoming.EventType != "tcp.open" {
		t.Errorf("eventType = %q, want %q", incoming.EventType, "tcp.open")
	}
	if !incoming.Timestamp.IsZero() {
		t.Errorf("timestamp should be zero when not set in proto, got %v", incoming.Timestamp)
	}
}

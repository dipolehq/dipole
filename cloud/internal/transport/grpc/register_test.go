/*
register_test.go tests the Register RPC handler, specifically that internal
error details are never returned to the caller. Register is reachable with
no session-token authentication at all (it is the RPC that establishes it),
so this is the RPC where an information-disclosure leak has the widest
possible audience.
*/
package grpc

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	pb "github.com/dipolehq/dipole/gen/dipole/v1"
)

type mockRegistrar struct {
	proxy        *domain.AgentProxy
	sessionToken string
	err          error
}

func (m *mockRegistrar) RegisterProxy(_ context.Context, _ uuid.UUID, _ string) (*domain.AgentProxy, string, error) {
	return m.proxy, m.sessionToken, m.err
}

type mockCAProvider struct {
	certPEM string
	err     error
}

func (m *mockCAProvider) GetOrCreateTenantCA(_ context.Context, _ uuid.UUID) (string, string, error) {
	return m.certPEM, "", m.err
}

func TestRegister_InternalErrorNotLeaked(t *testing.T) {
	sensitiveErr := fmt.Errorf("persist proxy: pq: connection to server at \"10.0.4.12\" failed")
	server := &PolicyEvaluationServer{
		registrar: &mockRegistrar{err: sensitiveErr},
		logger:    newTestLogger(),
	}

	resp, err := server.Register(context.Background(), &pb.RegisterRequest{
		TenantId:          uuid.New().String(),
		RegistrationToken: "some-token",
	})
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if resp.Success {
		t.Fatal("expected registration to fail")
	}
	if resp.ErrorMessage != "registration failed" {
		t.Errorf("ErrorMessage = %q, want a fixed generic message", resp.ErrorMessage)
	}
}

func TestRegister_RegistrationErrorMessageIsReturnedVerbatim(t *testing.T) {
	server := &PolicyEvaluationServer{
		registrar: &mockRegistrar{err: domain.NewRegistrationError("invalid or already consumed registration token")},
		logger:    newTestLogger(),
	}

	resp, err := server.Register(context.Background(), &pb.RegisterRequest{
		TenantId:          uuid.New().String(),
		RegistrationToken: "some-token",
	})
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if resp.Success {
		t.Fatal("expected registration to fail")
	}
	if resp.ErrorMessage != "invalid or already consumed registration token" {
		t.Errorf("ErrorMessage = %q, want the RegistrationError message returned as-is", resp.ErrorMessage)
	}
}

func TestRegister_HappyPath_ReturnsSessionTokenAndCACert(t *testing.T) {
	proxyID := uuid.New()
	tenantID := uuid.New()
	server := &PolicyEvaluationServer{
		registrar: &mockRegistrar{
			proxy:        &domain.AgentProxy{ID: proxyID, TenantID: tenantID},
			sessionToken: "raw-session-token",
		},
		caProvider: &mockCAProvider{certPEM: "-----BEGIN CERTIFICATE-----\nfake\n-----END CERTIFICATE-----"},
		logger:     newTestLogger(),
	}

	resp, err := server.Register(context.Background(), &pb.RegisterRequest{
		TenantId:          tenantID.String(),
		RegistrationToken: "some-token",
	})
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got error: %s", resp.ErrorMessage)
	}
	if resp.SessionToken != "raw-session-token" {
		t.Errorf("SessionToken = %q, want %q", resp.SessionToken, "raw-session-token")
	}
	if resp.ProxyId != proxyID.String() {
		t.Errorf("ProxyId = %q, want %q", resp.ProxyId, proxyID.String())
	}
	if resp.CaCertPem == "" {
		t.Error("expected CaCertPem to be set")
	}
}

func TestRegister_InvalidTenantID_Denied(t *testing.T) {
	server := &PolicyEvaluationServer{logger: newTestLogger()}

	resp, err := server.Register(context.Background(), &pb.RegisterRequest{
		TenantId:          "not-a-uuid",
		RegistrationToken: "some-token",
	})
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if resp.Success {
		t.Fatal("expected denial for invalid tenant_id")
	}
}

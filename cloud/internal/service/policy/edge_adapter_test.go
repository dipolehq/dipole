/*
edge_adapter_test.go verifies EdgePolicyAdapter correctly delegates to
PolicyEvaluationService and passes results through unchanged.
*/
package policy

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

// mockPolicyRepo implements repository.AccessPolicyRepository for adapter tests.
type mockPolicyRepo struct {
	policies []domain.AccessPolicyWithRelations
	err      error
}

func (m *mockPolicyRepo) Create(_ context.Context, _ uuid.UUID, _ *domain.AccessPolicy) error {
	return nil
}
func (m *mockPolicyRepo) GetByID(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*domain.AccessPolicy, error) {
	return nil, nil
}
func (m *mockPolicyRepo) List(_ context.Context, _ uuid.UUID, _ domain.ListOptions) ([]domain.AccessPolicy, int, error) {
	return nil, 0, nil
}
func (m *mockPolicyRepo) Update(_ context.Context, _ uuid.UUID, _ *domain.AccessPolicy) error {
	return nil
}
func (m *mockPolicyRepo) Delete(_ context.Context, _ uuid.UUID, _ uuid.UUID) error { return nil }
func (m *mockPolicyRepo) FindByServerTarget(_ context.Context, _ uuid.UUID, _ string, _ int, _ *uuid.UUID) ([]domain.AccessPolicyWithRelations, error) {
	return m.policies, m.err
}
func (m *mockPolicyRepo) FindByAppProtocol(_ context.Context, _ uuid.UUID, _ string) ([]domain.AccessPolicyWithRelations, error) {
	return m.policies, m.err
}
func (m *mockPolicyRepo) ListActiveWithRelations(_ context.Context, _ uuid.UUID) ([]domain.AccessPolicyWithRelations, error) {
	return m.policies, m.err
}

// mockTrustForAdapter satisfies the TrustValidator interface.
type mockTrustForAdapter struct {
	result *domain.TrustProviderResult
	err    error
}

func (m *mockTrustForAdapter) Validate(_ context.Context, _ *domain.TrustProvider, _ *domain.ClientEvidence) (*domain.TrustProviderResult, error) {
	return m.result, m.err
}

// mockBrokerForAdapter satisfies the CredentialBroker interface.
type mockBrokerForAdapter struct {
	cred *domain.Credential
	err  error
}

func (m *mockBrokerForAdapter) Retrieve(_ context.Context, _ uuid.UUID, _ *domain.CredentialProvider, _ *domain.ClientEvidence, _ map[string]interface{}) (*domain.Credential, error) {
	return m.cred, m.err
}

func TestEdgePolicyAdapter_NoMatchingPolicy_ReturnsDeny(t *testing.T) {
	repo := &mockPolicyRepo{policies: nil, err: nil}
	svc := NewPolicyEvaluationService(repo, nil, nil, nil, nil, nil)
	adapter := NewEdgePolicyAdapter(svc)

	tenantID := uuid.New()
	evidence := &domain.ClientEvidence{}
	target := &domain.ServerTarget{Host: "api.stripe.com", Port: 443}

	result, err := adapter.Evaluate(tenantID, evidence, target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Allowed {
		t.Error("expected Allowed=false when no policies match")
	}
	if result.DenyReason != "NoMatchingPolicy" {
		t.Errorf("expected DenyReason 'NoMatchingPolicy', got '%s'", result.DenyReason)
	}
}

func TestEdgePolicyAdapter_RepoError_ReturnsError(t *testing.T) {
	repo := &mockPolicyRepo{policies: nil, err: context.DeadlineExceeded}
	svc := NewPolicyEvaluationService(repo, nil, nil, nil, nil, nil)
	adapter := NewEdgePolicyAdapter(svc)

	_, err := adapter.Evaluate(uuid.New(), &domain.ClientEvidence{}, &domain.ServerTarget{Host: "db.local", Port: 5432})
	if err == nil {
		t.Fatal("expected error from repo failure")
	}
	if err != context.DeadlineExceeded {
		t.Errorf("expected context.DeadlineExceeded, got %v", err)
	}
}

func TestEdgePolicyAdapter_MatchingPolicy_ReturnsCredential(t *testing.T) {
	tenantID := uuid.New()
	policyID := uuid.New()
	cwID := uuid.New()
	swID := uuid.New()
	tpID := uuid.New()
	cpID := uuid.New()

	repo := &mockPolicyRepo{
		policies: []domain.AccessPolicyWithRelations{
			{
				Policy: domain.AccessPolicy{
					ID:       policyID,
					TenantID: tenantID,
					IsActive: true,
				},
				ClientWorkload: domain.ClientWorkload{
					ID:          cwID,
					Identifiers: []domain.Identifier{{Type: "hostname", Value: "ci-runner"}},
				},
				ServerWorkload: domain.ServerWorkload{
					ID:         swID,
					Host:       "api.stripe.com",
					Port:       443,
					AuthMethod: "http_bearer",
				},
				TrustProviders: []domain.TrustProvider{
					{ID: tpID, IsActive: true, Type: "github_action"},
				},
				CredentialProviders: []domain.CredentialProvider{
					{ID: cpID, Type: "api_key"},
				},
			},
		},
	}

	mockTrust := &mockTrustForAdapter{result: &domain.TrustProviderResult{Attested: true, ProviderID: tpID}}
	mockBroker := &mockBrokerForAdapter{cred: &domain.Credential{Type: "api_key", Value: "sk_live_test"}}
	svc := NewPolicyEvaluationService(repo, mockTrust, nil, mockBroker, nil, nil)
	adapter := NewEdgePolicyAdapter(svc)

	evidence := &domain.ClientEvidence{Hostname: "ci-runner"}
	target := &domain.ServerTarget{Host: "api.stripe.com", Port: 443}

	result, err := adapter.Evaluate(tenantID, evidence, target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Fatalf("expected Allowed=true, got deny reason: %s", result.DenyReason)
	}
	if result.Credential == nil || result.Credential.Value != "sk_live_test" {
		t.Error("expected credential with value 'sk_live_test'")
	}
	if result.PolicyID != policyID {
		t.Errorf("expected policyID %s, got %s", policyID, result.PolicyID)
	}
}

func (m *mockPolicyRepo) CountInactive(_ context.Context, _ uuid.UUID) int { return 0 }

func (m *mockPolicyRepo) IsHostnameAuthorizedForTLS(_ context.Context, _ uuid.UUID, _ string) (bool, error) {
	return false, nil
}

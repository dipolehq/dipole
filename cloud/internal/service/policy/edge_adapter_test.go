// edge_adapter_test.go verifies EdgePolicyAdapter correctly delegates to PolicyEvaluationService and passes results through unchanged.
package policy

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

func TestEdgePolicyAdapter_NoMatchingPolicy_ReturnsDeny(t *testing.T) {
	repo := &mockRepo{policies: nil, err: nil}
	svc := NewPolicyEvaluationService(repo, nil, nil, nil, nil, nil)
	adapter := NewEdgePolicyAdapter(svc)

	tenantID := uuid.New()
	evidence := &domain.ClientEvidence{}
	target := &domain.ServerTarget{Host: "api.stripe.com", Port: 443}

	result, err := adapter.Evaluate(context.Background(), tenantID, evidence, target)
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
	repo := &mockRepo{policies: nil, err: context.DeadlineExceeded}
	svc := NewPolicyEvaluationService(repo, nil, nil, nil, nil, nil)
	adapter := NewEdgePolicyAdapter(svc)

	_, err := adapter.Evaluate(context.Background(), uuid.New(), &domain.ClientEvidence{}, &domain.ServerTarget{Host: "db.local", Port: 5432})
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

	repo := &mockRepo{
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

	trust := &mockTrust{attested: true}
	broker := &mockBroker{cred: &domain.Credential{Type: "api_key", Value: "sk_live_test"}}
	svc := NewPolicyEvaluationService(repo, trust, nil, broker, nil, nil)
	adapter := NewEdgePolicyAdapter(svc)

	evidence := &domain.ClientEvidence{Hostname: "ci-runner"}
	target := &domain.ServerTarget{Host: "api.stripe.com", Port: 443}

	result, err := adapter.Evaluate(context.Background(), tenantID, evidence, target)
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

package policy

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type mockRepo struct {
	policies []domain.AccessPolicyWithRelations
	err      error
}

func (m *mockRepo) Create(_ context.Context, _ uuid.UUID, _ *domain.AccessPolicy) error { return nil }
func (m *mockRepo) GetByID(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*domain.AccessPolicy, error) {
	return nil, nil
}
func (m *mockRepo) List(_ context.Context, _ uuid.UUID, _ domain.ListOptions) ([]domain.AccessPolicy, int, error) {
	return nil, 0, nil
}
func (m *mockRepo) Update(_ context.Context, _ uuid.UUID, _ *domain.AccessPolicy) error { return nil }
func (m *mockRepo) Delete(_ context.Context, _ uuid.UUID, _ uuid.UUID) error            { return nil }
func (m *mockRepo) FindByServerTarget(_ context.Context, _ uuid.UUID, _ string, _ int) ([]domain.AccessPolicyWithRelations, error) {
	return m.policies, m.err
}

type mockTrust struct{ attested bool }

func (m *mockTrust) Validate(_ context.Context, p *domain.TrustProvider, _ *domain.ClientEvidence) (*domain.TrustProviderResult, error) {
	return &domain.TrustProviderResult{ProviderID: p.ID, Attested: m.attested}, nil
}

type mockConditions struct{ passed bool }

func (m *mockConditions) EvaluateAll(_ context.Context, _ []domain.AccessCondition, _ *domain.ClientEvidence) (bool, string, error) {
	return m.passed, "", nil
}

type mockBroker struct {
	cred *domain.Credential
	err  error
}

func (m *mockBroker) Retrieve(_ context.Context, _ *domain.CredentialProvider, _ *domain.ClientEvidence, _ map[string]interface{}) (*domain.Credential, error) {
	return m.cred, m.err
}

func newTestService(policies []domain.AccessPolicyWithRelations, repoErr error, cred *domain.Credential, brokerErr error) *PolicyEvaluationService {
	return NewPolicyEvaluationService(
		&mockRepo{policies: policies, err: repoErr},
		&mockTrust{attested: true},
		&mockConditions{passed: true},
		&mockBroker{cred: cred, err: brokerErr},
	)
}

func TestEvaluate_MatchingPolicy_ReturnsAllow(t *testing.T) {
	policies := []domain.AccessPolicyWithRelations{{
		Policy:              domain.AccessPolicy{ID: uuid.New(), IsActive: true},
		ClientWorkload:      domain.ClientWorkload{Identifiers: []domain.Identifier{{Type: "k8s_namespace", Value: "production"}}},
		ServerWorkload:      domain.ServerWorkload{AuthMethod: "http_bearer"},
		CredentialProviders: []domain.CredentialProvider{{Type: "api_key"}},
	}}
	cred := &domain.Credential{Type: "bearer", Value: "sk_live_123"}
	svc := newTestService(policies, nil, cred, nil)

	ev := &domain.ClientEvidence{K8s: &domain.K8sEvidence{Namespace: "production"}}
	result, err := svc.Evaluate(context.Background(), uuid.New(), ev, &domain.ServerTarget{Host: "api.stripe.com", Port: 443})

	if err != nil {
		t.Fatal(err)
	}
	if !result.Allowed {
		t.Errorf("allowed = false, want true")
	}
	if result.Credential.Value != "sk_live_123" {
		t.Errorf("credential = %q, want %q", result.Credential.Value, "sk_live_123")
	}
	if result.InjectionConfig.Method != "http_bearer" {
		t.Errorf("injection method = %q, want %q", result.InjectionConfig.Method, "http_bearer")
	}
}

func TestEvaluate_NoServerMatch_ReturnsDeny(t *testing.T) {
	svc := newTestService(nil, nil, nil, nil)

	result, err := svc.Evaluate(context.Background(), uuid.New(), &domain.ClientEvidence{}, &domain.ServerTarget{Host: "unknown.com", Port: 443})

	if err != nil {
		t.Fatal(err)
	}
	if result.Allowed {
		t.Error("allowed = true, want false")
	}
	if result.DenyReason != "NoMatchingPolicy" {
		t.Errorf("reason = %q, want %q", result.DenyReason, "NoMatchingPolicy")
	}
}

func TestEvaluate_ClientIdentifierMismatch_ReturnsDeny(t *testing.T) {
	policies := []domain.AccessPolicyWithRelations{{
		Policy:         domain.AccessPolicy{ID: uuid.New(), IsActive: true},
		ClientWorkload: domain.ClientWorkload{Identifiers: []domain.Identifier{{Type: "k8s_namespace", Value: "production"}}},
		ServerWorkload: domain.ServerWorkload{AuthMethod: "http_bearer"},
	}}
	svc := newTestService(policies, nil, nil, nil)

	ev := &domain.ClientEvidence{K8s: &domain.K8sEvidence{Namespace: "staging"}}
	result, _ := svc.Evaluate(context.Background(), uuid.New(), ev, &domain.ServerTarget{Host: "x", Port: 443})

	if result.Allowed {
		t.Error("allowed = true, want false (namespace mismatch)")
	}
}

func TestEvaluate_ANDLogic_BothMustMatch(t *testing.T) {
	policies := []domain.AccessPolicyWithRelations{{
		Policy: domain.AccessPolicy{ID: uuid.New(), IsActive: true},
		ClientWorkload: domain.ClientWorkload{Identifiers: []domain.Identifier{
			{Type: "k8s_namespace", Value: "production"},
			{Type: "k8s_pod_name_prefix", Value: "myapp-"},
		}},
		ServerWorkload: domain.ServerWorkload{AuthMethod: "http_bearer"},
	}}
	svc := newTestService(policies, nil, nil, nil)

	// Namespace matches but pod doesn't
	ev := &domain.ClientEvidence{K8s: &domain.K8sEvidence{Namespace: "production", PodName: "other-xyz"}}
	result, _ := svc.Evaluate(context.Background(), uuid.New(), ev, &domain.ServerTarget{Host: "x", Port: 443})

	if result.Allowed {
		t.Error("allowed = true, want false (pod prefix mismatch)")
	}
}

func TestEvaluate_PodNamePrefix_Matches(t *testing.T) {
	policies := []domain.AccessPolicyWithRelations{{
		Policy: domain.AccessPolicy{ID: uuid.New(), IsActive: true},
		ClientWorkload: domain.ClientWorkload{Identifiers: []domain.Identifier{
			{Type: "k8s_pod_name_prefix", Value: "myapp-"},
		}},
		ServerWorkload:      domain.ServerWorkload{AuthMethod: "http_bearer"},
		CredentialProviders: []domain.CredentialProvider{{Type: "api_key"}},
	}}
	cred := &domain.Credential{Type: "bearer", Value: "key"}
	svc := newTestService(policies, nil, cred, nil)

	ev := &domain.ClientEvidence{K8s: &domain.K8sEvidence{PodName: "myapp-abc123"}}
	result, _ := svc.Evaluate(context.Background(), uuid.New(), ev, &domain.ServerTarget{Host: "x", Port: 443})

	if !result.Allowed {
		t.Error("allowed = false, want true (prefix matches)")
	}
}

func TestEvaluate_FirstMatchWins(t *testing.T) {
	policies := []domain.AccessPolicyWithRelations{
		{
			Policy:              domain.AccessPolicy{ID: uuid.MustParse("00000000-0000-0000-0000-000000000001"), IsActive: true},
			ClientWorkload:      domain.ClientWorkload{Identifiers: []domain.Identifier{{Type: "k8s_namespace", Value: "production"}}},
			ServerWorkload:      domain.ServerWorkload{AuthMethod: "http_bearer"},
			CredentialProviders: []domain.CredentialProvider{{Type: "api_key"}},
		},
		{
			Policy:              domain.AccessPolicy{ID: uuid.MustParse("00000000-0000-0000-0000-000000000002"), IsActive: true},
			ClientWorkload:      domain.ClientWorkload{Identifiers: []domain.Identifier{{Type: "k8s_namespace", Value: "production"}}},
			ServerWorkload:      domain.ServerWorkload{AuthMethod: "api_key_header", AuthConfig: map[string]string{"header_name": "X-Key"}},
			CredentialProviders: []domain.CredentialProvider{{Type: "api_key"}},
		},
	}
	cred := &domain.Credential{Type: "bearer", Value: "first-wins"}
	svc := newTestService(policies, nil, cred, nil)

	ev := &domain.ClientEvidence{K8s: &domain.K8sEvidence{Namespace: "production"}}
	result, _ := svc.Evaluate(context.Background(), uuid.New(), ev, &domain.ServerTarget{Host: "x", Port: 443})

	if result.PolicyID != policies[0].Policy.ID {
		t.Errorf("matched policy = %s, want first policy %s", result.PolicyID, policies[0].Policy.ID)
	}
}

func TestEvaluate_NilCredentialProvider_AllowsWithNilCred(t *testing.T) {
	policies := []domain.AccessPolicyWithRelations{{
		Policy:         domain.AccessPolicy{ID: uuid.New(), IsActive: true},
		ClientWorkload: domain.ClientWorkload{Identifiers: []domain.Identifier{}},
		ServerWorkload: domain.ServerWorkload{AuthMethod: "http_bearer"},
	}}
	svc := newTestService(policies, nil, nil, nil)

	result, _ := svc.Evaluate(context.Background(), uuid.New(), &domain.ClientEvidence{}, &domain.ServerTarget{Host: "x", Port: 443})

	if !result.Allowed {
		t.Error("allowed = false, want true (no CP = passthrough match)")
	}
	if result.Credential != nil {
		t.Error("credential should be nil when no CP configured")
	}
}

func TestEvaluate_CredentialRetrievalFails_ReturnsDeny(t *testing.T) {
	policies := []domain.AccessPolicyWithRelations{{
		Policy:              domain.AccessPolicy{ID: uuid.New(), IsActive: true},
		ClientWorkload:      domain.ClientWorkload{Identifiers: []domain.Identifier{}},
		ServerWorkload:      domain.ServerWorkload{AuthMethod: "http_bearer"},
		CredentialProviders: []domain.CredentialProvider{{Type: "api_key"}},
	}}
	svc := NewPolicyEvaluationService(
		&mockRepo{policies: policies},
		&mockTrust{attested: true},
		&mockConditions{passed: true},
		&mockBroker{err: fmt.Errorf("decryption failed")},
	)

	result, _ := svc.Evaluate(context.Background(), uuid.New(), &domain.ClientEvidence{}, &domain.ServerTarget{Host: "x", Port: 443})

	if result.Allowed {
		t.Error("allowed = true, want false (credential retrieval failed)")
	}
	if result.DenyReason != "CredentialRetrievalFailed" {
		t.Errorf("reason = %q, want %q", result.DenyReason, "CredentialRetrievalFailed")
	}
}

func TestEvaluate_DBError_ReturnsError(t *testing.T) {
	svc := NewPolicyEvaluationService(
		&mockRepo{err: fmt.Errorf("db connection lost")},
		&mockTrust{attested: true},
		&mockConditions{passed: true},
		&mockBroker{},
	)

	_, err := svc.Evaluate(context.Background(), uuid.New(), &domain.ClientEvidence{}, &domain.ServerTarget{Host: "x", Port: 443})

	if err == nil {
		t.Error("expected error from DB failure")
	}
}

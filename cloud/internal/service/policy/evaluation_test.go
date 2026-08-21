// evaluation_test.go tests PolicyEvaluationService.Evaluate: identifier matching, trust/condition/credential flow, deny reasons, and security event emission.
package policy

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/events"
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
func (m *mockRepo) FindByServerTarget(_ context.Context, _ uuid.UUID, _ string, _ int, _ *uuid.UUID) ([]domain.AccessPolicyWithRelations, error) {
	return m.policies, m.err
}
func (m *mockRepo) FindByAppProtocol(_ context.Context, _ uuid.UUID, _ string) ([]domain.AccessPolicyWithRelations, error) {
	return m.policies, m.err
}
func (m *mockRepo) ListActiveWithRelations(_ context.Context, _ uuid.UUID) ([]domain.AccessPolicyWithRelations, error) {
	return m.policies, m.err
}

type mockTrust struct {
	attested bool
	claims   map[string]interface{}
}

func (m *mockTrust) Validate(_ context.Context, p *domain.TrustProvider, _ *domain.ClientEvidence) (*domain.TrustProviderResult, error) {
	return &domain.TrustProviderResult{ProviderID: p.ID, Attested: m.attested, Claims: m.claims}, nil
}

type mockConditions struct{ passed bool }

func (m *mockConditions) EvaluateAll(_ context.Context, _ []domain.AccessCondition, _ *domain.ClientEvidence) (bool, string, error) {
	return m.passed, "", nil
}

type mockBroker struct {
	cred           *domain.Credential
	err            error
	receivedClaims map[string]interface{}
}

func (m *mockBroker) Retrieve(_ context.Context, _ uuid.UUID, _ *domain.CredentialProvider, _ *domain.ClientEvidence, trustClaims map[string]interface{}) (*domain.Credential, error) {
	m.receivedClaims = trustClaims
	return m.cred, m.err
}

type recordingEventRepo struct {
	securityEvents []*events.SecurityEvent
}

var errResourceSetNotNull = fmt.Errorf(`insert security event: ERROR: null value in column "resource_set_id" violates not-null constraint (SQLSTATE 23502)`)

func (r *recordingEventRepo) InsertSecurityEvent(_ context.Context, event *events.SecurityEvent) error {
	if event.ResourceSetID == nil {
		return errResourceSetNotNull
	}
	r.securityEvents = append(r.securityEvents, event)
	return nil
}

func (r *recordingEventRepo) InsertWorkloadEventBatch(_ context.Context, batch []events.WorkloadEvent) error {
	for i := range batch {
		if batch[i].ResourceSetID == nil {
			return errResourceSetNotNull
		}
	}
	return nil
}

func (r *recordingEventRepo) InsertAuditEvent(_ context.Context, _ *events.AuditEvent) error {
	return nil
}

func newTestService(policies []domain.AccessPolicyWithRelations, repoErr error, cred *domain.Credential, brokerErr error) *PolicyEvaluationService {
	return NewPolicyEvaluationService(
		&mockRepo{policies: policies, err: repoErr},
		&mockTrust{attested: true},
		&mockConditions{passed: true},
		&mockBroker{cred: cred, err: brokerErr},
		nil,
		slog.Default(),
	)
}

func TestEvaluate_MatchingPolicy_ReturnsAllow(t *testing.T) {
	policies := []domain.AccessPolicyWithRelations{{
		Policy:              domain.AccessPolicy{ID: uuid.New(), IsActive: true},
		ClientWorkload:      domain.ClientWorkload{Identifiers: []domain.Identifier{{Type: "k8s_namespace", Value: "production"}}},
		ServerWorkload:      domain.ServerWorkload{AuthMethod: "http_bearer"},
		TrustProviders:      []domain.TrustProvider{{ID: uuid.New(), Type: "k8s_service_account"}},
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

	ev := &domain.ClientEvidence{K8s: &domain.K8sEvidence{Namespace: "production", PodName: "other-xyz"}}
	result, _ := svc.Evaluate(context.Background(), uuid.New(), ev, &domain.ServerTarget{Host: "x", Port: 443})

	if result.Allowed {
		t.Error("allowed = true, want false (pod prefix mismatch)")
	}
}

func TestEvaluate_SameType_ORLogic_OneValueMatches(t *testing.T) {
	policies := []domain.AccessPolicyWithRelations{{
		Policy: domain.AccessPolicy{ID: uuid.New(), IsActive: true},
		ClientWorkload: domain.ClientWorkload{Identifiers: []domain.Identifier{
			{Type: "k8s_namespace", Value: "production"},
			{Type: "k8s_namespace", Value: "staging"},
		}},
		ServerWorkload:      domain.ServerWorkload{AuthMethod: "http_bearer"},
		TrustProviders:      []domain.TrustProvider{{ID: uuid.New(), Type: "k8s_service_account"}},
		CredentialProviders: []domain.CredentialProvider{{Type: "api_key"}},
	}}
	cred := &domain.Credential{Type: "bearer", Value: "key"}
	svc := newTestService(policies, nil, cred, nil)

	ev := &domain.ClientEvidence{K8s: &domain.K8sEvidence{Namespace: "staging"}}
	result, _ := svc.Evaluate(context.Background(), uuid.New(), ev, &domain.ServerTarget{Host: "x", Port: 443})

	if !result.Allowed {
		t.Error("allowed = false, want true (staging is one of the OR'd namespace values)")
	}
}

func TestEvaluate_MixedTypes_ORWithinType_ANDAcrossTypes(t *testing.T) {
	policies := []domain.AccessPolicyWithRelations{{
		Policy: domain.AccessPolicy{ID: uuid.New(), IsActive: true},
		ClientWorkload: domain.ClientWorkload{Identifiers: []domain.Identifier{
			{Type: "k8s_namespace", Value: "production"},
			{Type: "k8s_namespace", Value: "staging"},
			{Type: "k8s_service_account_name", Value: "payments-sa"},
		}},
		ServerWorkload: domain.ServerWorkload{AuthMethod: "http_bearer"},
	}}
	svc := newTestService(policies, nil, nil, nil)

	ev := &domain.ClientEvidence{K8s: &domain.K8sEvidence{Namespace: "staging", ServiceAccountName: "other-sa"}}
	result, _ := svc.Evaluate(context.Background(), uuid.New(), ev, &domain.ServerTarget{Host: "x", Port: 443})

	if result.Allowed {
		t.Error("allowed = true, want false (service account type group has no matching value)")
	}
}

func TestEvaluate_PodNamePrefix_Matches(t *testing.T) {
	policies := []domain.AccessPolicyWithRelations{{
		Policy: domain.AccessPolicy{ID: uuid.New(), IsActive: true},
		ClientWorkload: domain.ClientWorkload{Identifiers: []domain.Identifier{
			{Type: "k8s_pod_name_prefix", Value: "myapp-"},
		}},
		ServerWorkload:      domain.ServerWorkload{AuthMethod: "http_bearer"},
		TrustProviders:      []domain.TrustProvider{{ID: uuid.New(), Type: "k8s_service_account"}},
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
			TrustProviders:      []domain.TrustProvider{{ID: uuid.New(), Type: "k8s_service_account"}},
			CredentialProviders: []domain.CredentialProvider{{Type: "api_key"}},
		},
		{
			Policy:              domain.AccessPolicy{ID: uuid.MustParse("00000000-0000-0000-0000-000000000002"), IsActive: true},
			ClientWorkload:      domain.ClientWorkload{Identifiers: []domain.Identifier{{Type: "k8s_namespace", Value: "production"}}},
			ServerWorkload:      domain.ServerWorkload{AuthMethod: "api_key_header", AuthConfig: map[string]string{"header_name": "X-Key"}},
			TrustProviders:      []domain.TrustProvider{{ID: uuid.New(), Type: "k8s_service_account"}},
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
		TrustProviders: []domain.TrustProvider{{ID: uuid.New(), Type: "k8s_service_account"}},
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
		TrustProviders:      []domain.TrustProvider{{ID: uuid.New(), Type: "k8s_service_account"}},
		CredentialProviders: []domain.CredentialProvider{{Type: "api_key"}},
	}}
	svc := NewPolicyEvaluationService(
		&mockRepo{policies: policies},
		&mockTrust{attested: true},
		&mockConditions{passed: true},
		&mockBroker{err: fmt.Errorf("decryption failed")},
		nil,
		slog.Default(),
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
		nil,
		slog.Default(),
	)

	_, err := svc.Evaluate(context.Background(), uuid.New(), &domain.ClientEvidence{}, &domain.ServerTarget{Host: "x", Port: 443})

	if err == nil {
		t.Error("expected error from DB failure")
	}
}

func TestEvaluate_ZeroTrustProviders_ReturnsDeny(t *testing.T) {
	policies := []domain.AccessPolicyWithRelations{{
		Policy:              domain.AccessPolicy{ID: uuid.New(), IsActive: true},
		ClientWorkload:      domain.ClientWorkload{Identifiers: []domain.Identifier{{Type: "k8s_namespace", Value: "production"}}},
		ServerWorkload:      domain.ServerWorkload{AuthMethod: "http_bearer"},
		TrustProviders:      []domain.TrustProvider{},
		CredentialProviders: []domain.CredentialProvider{{Type: "api_key"}},
	}}
	svc := newTestService(policies, nil, &domain.Credential{Value: "token"}, nil)

	ev := &domain.ClientEvidence{K8s: &domain.K8sEvidence{Namespace: "production"}}
	result, err := svc.Evaluate(context.Background(), uuid.New(), ev, &domain.ServerTarget{Host: "api.stripe.com", Port: 443})

	if err != nil {
		t.Fatal(err)
	}
	if result.Allowed {
		t.Error("allowed = true, want false (zero trust providers)")
	}
	if result.DenyReason != "NoTrustProvider" {
		t.Errorf("reason = %q, want %q", result.DenyReason, "NoTrustProvider")
	}
}

func TestEvaluate_EmitterCalledOnAllow(t *testing.T) {
	repo := &recordingEventRepo{}
	bus := events.NewEventBus(repo, nil)
	policies := []domain.AccessPolicyWithRelations{{
		Policy:              domain.AccessPolicy{ID: uuid.New(), IsActive: true},
		ClientWorkload:      domain.ClientWorkload{Identifiers: []domain.Identifier{{Type: "k8s_namespace", Value: "prod"}}},
		ServerWorkload:      domain.ServerWorkload{AuthMethod: "http_bearer"},
		TrustProviders:      []domain.TrustProvider{{ID: uuid.New(), Type: "k8s_service_account"}},
		CredentialProviders: []domain.CredentialProvider{{Type: "api_key"}},
	}}

	svc := NewPolicyEvaluationService(
		&mockRepo{policies: policies},
		&mockTrust{attested: true},
		&mockConditions{passed: true},
		&mockBroker{cred: &domain.Credential{Value: "token"}},
		bus,
		slog.Default(),
	)

	ev := &domain.ClientEvidence{K8s: &domain.K8sEvidence{Namespace: "prod"}}
	result, err := svc.Evaluate(context.Background(), uuid.New(), ev, &domain.ServerTarget{Host: "x", Port: 443})

	if err != nil {
		t.Fatal(err)
	}
	if !result.Allowed {
		t.Fatal("expected allow")
	}
	if len(repo.securityEvents) < 3 {
		t.Fatalf("expected at least 3 security events (request+authorization+discovery), got %d", len(repo.securityEvents))
	}

	contextID := repo.securityEvents[0].ContextID
	for _, ev := range repo.securityEvents {
		if ev.ContextID != contextID {
			t.Error("all events should share the same contextID")
		}
		if ev.Source != "edge_proxy" {
			t.Errorf("expected source 'edge_proxy', got %q", ev.Source)
		}
	}
}

func TestEvaluate_EmitterCalledOnDeny(t *testing.T) {
	repo := &recordingEventRepo{}
	bus := events.NewEventBus(repo, nil)
	policies := []domain.AccessPolicyWithRelations{{
		Policy:              domain.AccessPolicy{ID: uuid.New(), IsActive: true},
		ClientWorkload:      domain.ClientWorkload{Identifiers: []domain.Identifier{{Type: "k8s_namespace", Value: "prod"}}},
		ServerWorkload:      domain.ServerWorkload{AuthMethod: "http_bearer"},
		TrustProviders:      []domain.TrustProvider{{ID: uuid.New(), Type: "k8s_service_account"}},
		CredentialProviders: []domain.CredentialProvider{{Type: "api_key"}},
	}}

	svc := NewPolicyEvaluationService(
		&mockRepo{policies: policies},
		&mockTrust{attested: false},
		&mockConditions{passed: true},
		&mockBroker{},
		bus,
		slog.Default(),
	)

	ev := &domain.ClientEvidence{K8s: &domain.K8sEvidence{Namespace: "prod"}}
	result, err := svc.Evaluate(context.Background(), uuid.New(), ev, &domain.ServerTarget{Host: "x", Port: 443})

	if err != nil {
		t.Fatal(err)
	}
	if result.Allowed {
		t.Fatal("expected deny")
	}
	if len(repo.securityEvents) < 3 {
		t.Fatalf("expected at least 3 security events emitted on deny, got %d", len(repo.securityEvents))
	}
	hasAuth := false
	for _, ev := range repo.securityEvents {
		if ev.EventType != "access.authorization" {
			continue
		}
		auth, ok := ev.Detail.(events.AuthorizationDetail)
		if ok && !auth.Allowed {
			hasAuth = true
		}
	}
	if !hasAuth {
		t.Error("expected an access.authorization event with Allowed=false")
	}
}

func TestEvaluate_ResourceSetID_ThreadedFromTarget(t *testing.T) {
	repo := &recordingEventRepo{}
	bus := events.NewEventBus(repo, nil)
	resourceSetID := uuid.New()
	policies := []domain.AccessPolicyWithRelations{{
		Policy:              domain.AccessPolicy{ID: uuid.New(), IsActive: true},
		ClientWorkload:      domain.ClientWorkload{Identifiers: []domain.Identifier{{Type: "k8s_namespace", Value: "prod"}}},
		ServerWorkload:      domain.ServerWorkload{AuthMethod: "http_bearer"},
		TrustProviders:      []domain.TrustProvider{{ID: uuid.New(), Type: "k8s_service_account"}},
		CredentialProviders: []domain.CredentialProvider{{Type: "api_key"}},
	}}

	svc := NewPolicyEvaluationService(
		&mockRepo{policies: policies},
		&mockTrust{attested: true},
		&mockConditions{passed: true},
		&mockBroker{cred: &domain.Credential{Value: "token"}},
		bus,
		slog.Default(),
	)

	ev := &domain.ClientEvidence{K8s: &domain.K8sEvidence{Namespace: "prod"}}
	_, err := svc.Evaluate(context.Background(), uuid.New(), ev, &domain.ServerTarget{Host: "x", Port: 443, ResourceSetID: &resourceSetID})
	if err != nil {
		t.Fatal(err)
	}

	if len(repo.securityEvents) == 0 {
		t.Fatal("expected security events to be recorded; a NOT NULL constraint failure would silently drop all of them")
	}
	for _, sev := range repo.securityEvents {
		if sev.ResourceSetID == nil {
			t.Errorf("event %q has nil ResourceSetID; would violate NOT NULL constraint in production", sev.EventType)
			continue
		}
		if *sev.ResourceSetID != resourceSetID {
			t.Errorf("event %q ResourceSetID = %s, want %s (from target.ResourceSetID)", sev.EventType, sev.ResourceSetID, resourceSetID)
		}
	}
}

func TestEvaluate_TrustClaims_PassedToCredentialBroker(t *testing.T) {
	policies := []domain.AccessPolicyWithRelations{{
		Policy:              domain.AccessPolicy{ID: uuid.New(), IsActive: true},
		ClientWorkload:      domain.ClientWorkload{Identifiers: []domain.Identifier{{Type: "k8s_namespace", Value: "prod"}}},
		ServerWorkload:      domain.ServerWorkload{AuthMethod: "http_bearer"},
		TrustProviders:      []domain.TrustProvider{{ID: uuid.New(), Type: "k8s_service_account"}},
		CredentialProviders: []domain.CredentialProvider{{Type: "jwt"}},
	}}
	claims := map[string]interface{}{"sub": "system:serviceaccount:prod:my-sa", "aud": "https://acme-id.dipolehq.com"}
	broker := &mockBroker{cred: &domain.Credential{Value: "minted-token"}}

	svc := NewPolicyEvaluationService(
		&mockRepo{policies: policies},
		&mockTrust{attested: true, claims: claims},
		&mockConditions{passed: true},
		broker,
		nil,
		slog.Default(),
	)

	ev := &domain.ClientEvidence{K8s: &domain.K8sEvidence{Namespace: "prod"}}
	result, err := svc.Evaluate(context.Background(), uuid.New(), ev, &domain.ServerTarget{Host: "x", Port: 443})

	if err != nil {
		t.Fatal(err)
	}
	if !result.Allowed {
		t.Fatalf("expected allow, got deny reason: %s", result.DenyReason)
	}
	if broker.receivedClaims == nil {
		t.Fatal("expected trust claims to be passed to the credential broker")
	}
	if broker.receivedClaims["sub"] != "system:serviceaccount:prod:my-sa" {
		t.Errorf("receivedClaims[sub] = %v, want system:serviceaccount:prod:my-sa", broker.receivedClaims["sub"])
	}
}

func TestEvaluate_NoTrustClaims_PassesNilToBroker(t *testing.T) {
	policies := []domain.AccessPolicyWithRelations{{
		Policy:              domain.AccessPolicy{ID: uuid.New(), IsActive: true},
		ClientWorkload:      domain.ClientWorkload{Identifiers: []domain.Identifier{{Type: "k8s_namespace", Value: "prod"}}},
		ServerWorkload:      domain.ServerWorkload{AuthMethod: "http_bearer"},
		TrustProviders:      []domain.TrustProvider{{ID: uuid.New(), Type: "k8s_service_account"}},
		CredentialProviders: []domain.CredentialProvider{{Type: "api_key"}},
	}}
	broker := &mockBroker{cred: &domain.Credential{Value: "token"}}

	svc := NewPolicyEvaluationService(
		&mockRepo{policies: policies},
		&mockTrust{attested: true},
		&mockConditions{passed: true},
		broker,
		nil,
		slog.Default(),
	)

	ev := &domain.ClientEvidence{K8s: &domain.K8sEvidence{Namespace: "prod"}}
	result, err := svc.Evaluate(context.Background(), uuid.New(), ev, &domain.ServerTarget{Host: "x", Port: 443})

	if err != nil {
		t.Fatal(err)
	}
	if !result.Allowed {
		t.Fatalf("expected allow, got deny reason: %s", result.DenyReason)
	}
	if broker.receivedClaims != nil {
		t.Errorf("expected nil trust claims when the trust provider returned none, got %v", broker.receivedClaims)
	}
}

func TestMergeTrustClaims_MultipleProviders_LastWriteWinsOnCollision(t *testing.T) {
	results := []domain.TrustProviderResult{
		{Attested: true, Claims: map[string]interface{}{"sub": "first", "shared": "from-first"}},
		{Attested: true, Claims: map[string]interface{}{"aud": "second", "shared": "from-second"}},
	}

	merged := mergeTrustClaims(results)

	if merged["sub"] != "first" {
		t.Errorf("merged[sub] = %v, want first", merged["sub"])
	}
	if merged["aud"] != "second" {
		t.Errorf("merged[aud] = %v, want second", merged["aud"])
	}
	if merged["shared"] != "from-second" {
		t.Errorf("merged[shared] = %v, want from-second (last provider in order wins)", merged["shared"])
	}
}

func TestMergeTrustClaims_NoAttestedClaims_ReturnsNil(t *testing.T) {
	results := []domain.TrustProviderResult{
		{Attested: true, Claims: nil},
	}

	if merged := mergeTrustClaims(results); merged != nil {
		t.Errorf("expected nil, got %v", merged)
	}

	if merged := mergeTrustClaims(nil); merged != nil {
		t.Errorf("expected nil for empty results, got %v", merged)
	}
}

func (m *mockRepo) CountInactive(_ context.Context, _ uuid.UUID) int { return 0 }

func (m *mockRepo) IsHostnameAuthorizedForTLS(_ context.Context, _ uuid.UUID, _ string) (bool, error) {
	return false, nil
}

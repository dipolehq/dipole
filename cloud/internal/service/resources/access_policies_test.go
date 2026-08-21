package resources

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/scope"
)

type fakeAccessPolicyRepo struct {
	stored map[uuid.UUID]*domain.AccessPolicy
}

func newFakeAccessPolicyRepo() *fakeAccessPolicyRepo {
	return &fakeAccessPolicyRepo{stored: make(map[uuid.UUID]*domain.AccessPolicy)}
}

func (f *fakeAccessPolicyRepo) Create(_ context.Context, tenantID uuid.UUID, p *domain.AccessPolicy) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	p.TenantID = tenantID
	copy := *p
	f.stored[p.ID] = &copy
	return nil
}
func (f *fakeAccessPolicyRepo) GetByID(_ context.Context, _ uuid.UUID, id uuid.UUID) (*domain.AccessPolicy, error) {
	return f.stored[id], nil
}
func (f *fakeAccessPolicyRepo) List(_ context.Context, _ uuid.UUID, _ domain.ListOptions) ([]domain.AccessPolicy, int, error) {
	return nil, 0, nil
}
func (f *fakeAccessPolicyRepo) Update(_ context.Context, tenantID uuid.UUID, p *domain.AccessPolicy) error {
	copy := *p
	copy.TenantID = tenantID
	f.stored[p.ID] = &copy
	return nil
}
func (f *fakeAccessPolicyRepo) Delete(_ context.Context, _ uuid.UUID, id uuid.UUID) error {
	delete(f.stored, id)
	return nil
}
func (f *fakeAccessPolicyRepo) FindByServerTarget(_ context.Context, _ uuid.UUID, _ string, _ int, _ *uuid.UUID) ([]domain.AccessPolicyWithRelations, error) {
	return nil, nil
}
func (f *fakeAccessPolicyRepo) FindByAppProtocol(_ context.Context, _ uuid.UUID, _ string) ([]domain.AccessPolicyWithRelations, error) {
	return nil, nil
}
func (f *fakeAccessPolicyRepo) ListActiveWithRelations(_ context.Context, _ uuid.UUID) ([]domain.AccessPolicyWithRelations, error) {
	return nil, nil
}
func (f *fakeAccessPolicyRepo) CountInactive(_ context.Context, _ uuid.UUID) int { return 0 }
func (f *fakeAccessPolicyRepo) IsHostnameAuthorizedForTLS(_ context.Context, _ uuid.UUID, _ string) (bool, error) {
	return false, nil
}

type tenantScopedEntityRepo[T any] struct {
	byTenant map[uuid.UUID]map[uuid.UUID]*T
}

func newTenantScopedEntityRepo[T any]() *tenantScopedEntityRepo[T] {
	return &tenantScopedEntityRepo[T]{byTenant: make(map[uuid.UUID]map[uuid.UUID]*T)}
}

func (f *tenantScopedEntityRepo[T]) add(tenantID, id uuid.UUID, entity *T) {
	if f.byTenant[tenantID] == nil {
		f.byTenant[tenantID] = make(map[uuid.UUID]*T)
	}
	f.byTenant[tenantID][id] = entity
}

func (f *tenantScopedEntityRepo[T]) getByID(tenantID, id uuid.UUID) *T {
	tenantEntities, ok := f.byTenant[tenantID]
	if !ok {
		return nil
	}
	return tenantEntities[id]
}

type fakeClientWorkloadRepo struct {
	*tenantScopedEntityRepo[domain.ClientWorkload]
}
type fakeServerWorkloadRepo struct {
	*tenantScopedEntityRepo[domain.ServerWorkload]
}
type fakeTrustProviderRepoForPolicy struct {
	*tenantScopedEntityRepo[domain.TrustProvider]
}
type fakeCredentialProviderRepoForPolicy struct {
	*tenantScopedEntityRepo[domain.CredentialProvider]
}
type fakeAccessConditionRepoForPolicy struct {
	*tenantScopedEntityRepo[domain.AccessCondition]
}

func (f *fakeClientWorkloadRepo) Create(_ context.Context, _ uuid.UUID, _ *domain.ClientWorkload) error {
	return nil
}
func (f *fakeClientWorkloadRepo) GetByID(_ context.Context, tenantID, id uuid.UUID) (*domain.ClientWorkload, error) {
	return f.getByID(tenantID, id), nil
}
func (f *fakeClientWorkloadRepo) List(_ context.Context, _ uuid.UUID, _ domain.ListOptions) ([]domain.ClientWorkload, int, error) {
	return nil, 0, nil
}
func (f *fakeClientWorkloadRepo) Update(_ context.Context, _ uuid.UUID, _ *domain.ClientWorkload) error {
	return nil
}
func (f *fakeClientWorkloadRepo) Delete(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return nil
}
func (f *fakeClientWorkloadRepo) CountInactive(_ context.Context, _ uuid.UUID) int { return 0 }

func (f *fakeServerWorkloadRepo) Create(_ context.Context, _ uuid.UUID, _ *domain.ServerWorkload) error {
	return nil
}
func (f *fakeServerWorkloadRepo) GetByID(_ context.Context, tenantID, id uuid.UUID) (*domain.ServerWorkload, error) {
	return f.getByID(tenantID, id), nil
}
func (f *fakeServerWorkloadRepo) List(_ context.Context, _ uuid.UUID, _ domain.ListOptions) ([]domain.ServerWorkload, int, error) {
	return nil, 0, nil
}
func (f *fakeServerWorkloadRepo) Update(_ context.Context, _ uuid.UUID, _ *domain.ServerWorkload) error {
	return nil
}
func (f *fakeServerWorkloadRepo) Delete(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return nil
}
func (f *fakeServerWorkloadRepo) CountInactive(_ context.Context, _ uuid.UUID) int { return 0 }

func (f *fakeTrustProviderRepoForPolicy) Create(_ context.Context, _ uuid.UUID, _ *domain.TrustProvider) error {
	return nil
}
func (f *fakeTrustProviderRepoForPolicy) GetByID(_ context.Context, tenantID, id uuid.UUID) (*domain.TrustProvider, error) {
	return f.getByID(tenantID, id), nil
}
func (f *fakeTrustProviderRepoForPolicy) List(_ context.Context, _ uuid.UUID, _ domain.ListOptions) ([]domain.TrustProvider, int, error) {
	return nil, 0, nil
}
func (f *fakeTrustProviderRepoForPolicy) Update(_ context.Context, _ uuid.UUID, _ *domain.TrustProvider) error {
	return nil
}
func (f *fakeTrustProviderRepoForPolicy) Delete(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return nil
}
func (f *fakeTrustProviderRepoForPolicy) CountInactive(_ context.Context, _ uuid.UUID) int { return 0 }

func (f *fakeCredentialProviderRepoForPolicy) Create(_ context.Context, _ uuid.UUID, _ *domain.CredentialProvider) error {
	return nil
}
func (f *fakeCredentialProviderRepoForPolicy) GetByID(_ context.Context, tenantID, id uuid.UUID) (*domain.CredentialProvider, error) {
	return f.getByID(tenantID, id), nil
}
func (f *fakeCredentialProviderRepoForPolicy) List(_ context.Context, _ uuid.UUID, _ domain.ListOptions) ([]domain.CredentialProvider, int, error) {
	return nil, 0, nil
}
func (f *fakeCredentialProviderRepoForPolicy) Update(_ context.Context, _ uuid.UUID, _ *domain.CredentialProvider) error {
	return nil
}
func (f *fakeCredentialProviderRepoForPolicy) Delete(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return nil
}
func (f *fakeCredentialProviderRepoForPolicy) CountInactive(_ context.Context, _ uuid.UUID) int {
	return 0
}

func (f *fakeAccessConditionRepoForPolicy) Create(_ context.Context, _ uuid.UUID, _ *domain.AccessCondition) error {
	return nil
}
func (f *fakeAccessConditionRepoForPolicy) GetByID(_ context.Context, tenantID, id uuid.UUID) (*domain.AccessCondition, error) {
	return f.getByID(tenantID, id), nil
}
func (f *fakeAccessConditionRepoForPolicy) List(_ context.Context, _ uuid.UUID, _ domain.ListOptions) ([]domain.AccessCondition, int, error) {
	return nil, 0, nil
}
func (f *fakeAccessConditionRepoForPolicy) Update(_ context.Context, _ uuid.UUID, _ *domain.AccessCondition) error {
	return nil
}
func (f *fakeAccessConditionRepoForPolicy) Delete(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return nil
}
func (f *fakeAccessConditionRepoForPolicy) CountInactive(_ context.Context, _ uuid.UUID) int {
	return 0
}

type accessPolicyTestFixture struct {
	svc              *AccessPolicyService
	repo             *fakeAccessPolicyRepo
	clientWorkloads  *fakeClientWorkloadRepo
	serverWorkloads  *fakeServerWorkloadRepo
	trustProviders   *fakeTrustProviderRepoForPolicy
	credentialProvs  *fakeCredentialProviderRepoForPolicy
	accessConditions *fakeAccessConditionRepoForPolicy
}

func newAccessPolicyTestFixture() *accessPolicyTestFixture {
	f := &accessPolicyTestFixture{
		repo:             newFakeAccessPolicyRepo(),
		clientWorkloads:  &fakeClientWorkloadRepo{newTenantScopedEntityRepo[domain.ClientWorkload]()},
		serverWorkloads:  &fakeServerWorkloadRepo{newTenantScopedEntityRepo[domain.ServerWorkload]()},
		trustProviders:   &fakeTrustProviderRepoForPolicy{newTenantScopedEntityRepo[domain.TrustProvider]()},
		credentialProvs:  &fakeCredentialProviderRepoForPolicy{newTenantScopedEntityRepo[domain.CredentialProvider]()},
		accessConditions: &fakeAccessConditionRepoForPolicy{newTenantScopedEntityRepo[domain.AccessCondition]()},
	}
	f.svc = NewAccessPolicyService(f.repo, f.clientWorkloads, f.serverWorkloads, f.trustProviders, f.credentialProvs, f.accessConditions)
	return f
}

func TestAccessPolicyService_Create_RejectsCrossTenantTrustProvider(t *testing.T) {
	f := newAccessPolicyTestFixture()
	tenantA := uuid.New()
	tenantB := uuid.New()

	cwID, swID := uuid.New(), uuid.New()
	f.clientWorkloads.add(tenantA, cwID, &domain.ClientWorkload{ID: cwID, TenantID: tenantA})
	f.serverWorkloads.add(tenantA, swID, &domain.ServerWorkload{ID: swID, TenantID: tenantA})

	otherTenantsTPID := uuid.New()
	f.trustProviders.add(tenantB, otherTenantsTPID, &domain.TrustProvider{ID: otherTenantsTPID, TenantID: tenantB})

	policy := &domain.AccessPolicy{
		Name:             "cross-tenant-attempt",
		ClientWorkloadID: cwID,
		ServerWorkloadID: swID,
		TrustProviderIDs: []uuid.UUID{otherTenantsTPID},
	}

	err := f.svc.Create(context.Background(), tenantA, policy)
	if err == nil {
		t.Fatal("expected an error when referencing another tenant's Trust Provider, got nil")
	}
	if _, ok := err.(*domain.ValidationError); !ok {
		t.Errorf("expected a domain.ValidationError, got %T: %v", err, err)
	}
	if len(f.repo.stored) != 0 {
		t.Error("expected the policy to NOT be persisted when a cross-tenant reference is rejected")
	}
}

func TestAccessPolicyService_Create_RejectsCrossTenantCredentialProvider(t *testing.T) {
	f := newAccessPolicyTestFixture()
	tenantA := uuid.New()
	tenantB := uuid.New()

	cwID, swID := uuid.New(), uuid.New()
	f.clientWorkloads.add(tenantA, cwID, &domain.ClientWorkload{ID: cwID, TenantID: tenantA})
	f.serverWorkloads.add(tenantA, swID, &domain.ServerWorkload{ID: swID, TenantID: tenantA})

	otherTenantsCPID := uuid.New()
	f.credentialProvs.add(tenantB, otherTenantsCPID, &domain.CredentialProvider{ID: otherTenantsCPID, TenantID: tenantB})

	policy := &domain.AccessPolicy{
		Name:                  "cross-tenant-attempt",
		ClientWorkloadID:      cwID,
		ServerWorkloadID:      swID,
		CredentialProviderIDs: []domain.PolicyCredentialProvider{{CredentialProviderID: otherTenantsCPID}},
	}

	err := f.svc.Create(context.Background(), tenantA, policy)
	if err == nil {
		t.Fatal("expected an error when referencing another tenant's Credential Provider, got nil")
	}
}

func TestAccessPolicyService_Create_RejectsCrossTenantAccessCondition(t *testing.T) {
	f := newAccessPolicyTestFixture()
	tenantA := uuid.New()
	tenantB := uuid.New()

	cwID, swID := uuid.New(), uuid.New()
	f.clientWorkloads.add(tenantA, cwID, &domain.ClientWorkload{ID: cwID, TenantID: tenantA})
	f.serverWorkloads.add(tenantA, swID, &domain.ServerWorkload{ID: swID, TenantID: tenantA})

	otherTenantsACID := uuid.New()
	f.accessConditions.add(tenantB, otherTenantsACID, &domain.AccessCondition{ID: otherTenantsACID, TenantID: tenantB})

	policy := &domain.AccessPolicy{
		Name:               "cross-tenant-attempt",
		ClientWorkloadID:   cwID,
		ServerWorkloadID:   swID,
		AccessConditionIDs: []uuid.UUID{otherTenantsACID},
	}

	err := f.svc.Create(context.Background(), tenantA, policy)
	if err == nil {
		t.Fatal("expected an error when referencing another tenant's Access Condition, got nil")
	}
}

func TestAccessPolicyService_Create_RejectsCrossTenantClientWorkload(t *testing.T) {
	f := newAccessPolicyTestFixture()
	tenantA := uuid.New()
	tenantB := uuid.New()

	otherTenantsCWID := uuid.New()
	f.clientWorkloads.add(tenantB, otherTenantsCWID, &domain.ClientWorkload{ID: otherTenantsCWID, TenantID: tenantB})
	swID := uuid.New()
	f.serverWorkloads.add(tenantA, swID, &domain.ServerWorkload{ID: swID, TenantID: tenantA})

	policy := &domain.AccessPolicy{
		Name:             "cross-tenant-attempt",
		ClientWorkloadID: otherTenantsCWID,
		ServerWorkloadID: swID,
	}

	err := f.svc.Create(context.Background(), tenantA, policy)
	if err == nil {
		t.Fatal("expected an error when referencing another tenant's Client Workload, got nil")
	}
}

func TestAccessPolicyService_Create_RejectsCrossTenantServerWorkload(t *testing.T) {
	f := newAccessPolicyTestFixture()
	tenantA := uuid.New()
	tenantB := uuid.New()

	cwID := uuid.New()
	f.clientWorkloads.add(tenantA, cwID, &domain.ClientWorkload{ID: cwID, TenantID: tenantA})
	otherTenantsSWID := uuid.New()
	f.serverWorkloads.add(tenantB, otherTenantsSWID, &domain.ServerWorkload{ID: otherTenantsSWID, TenantID: tenantB})

	policy := &domain.AccessPolicy{
		Name:             "cross-tenant-attempt",
		ClientWorkloadID: cwID,
		ServerWorkloadID: otherTenantsSWID,
	}

	err := f.svc.Create(context.Background(), tenantA, policy)
	if err == nil {
		t.Fatal("expected an error when referencing another tenant's Server Workload, got nil")
	}
}

func TestAccessPolicyService_Create_AllSameTenant_Succeeds(t *testing.T) {
	f := newAccessPolicyTestFixture()
	tenantA := uuid.New()

	cwID, swID, tpID, cpID, acID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	f.clientWorkloads.add(tenantA, cwID, &domain.ClientWorkload{ID: cwID, TenantID: tenantA})
	f.serverWorkloads.add(tenantA, swID, &domain.ServerWorkload{ID: swID, TenantID: tenantA})
	f.trustProviders.add(tenantA, tpID, &domain.TrustProvider{ID: tpID, TenantID: tenantA})
	f.credentialProvs.add(tenantA, cpID, &domain.CredentialProvider{ID: cpID, TenantID: tenantA})
	f.accessConditions.add(tenantA, acID, &domain.AccessCondition{ID: acID, TenantID: tenantA})

	policy := &domain.AccessPolicy{
		Name:                  "same-tenant-policy",
		ClientWorkloadID:      cwID,
		ServerWorkloadID:      swID,
		TrustProviderIDs:      []uuid.UUID{tpID},
		CredentialProviderIDs: []domain.PolicyCredentialProvider{{CredentialProviderID: cpID}},
		AccessConditionIDs:    []uuid.UUID{acID},
	}

	err := f.svc.Create(context.Background(), tenantA, policy)
	if err != nil {
		t.Fatalf("expected a same-tenant policy to be created successfully, got error: %v", err)
	}
	if len(f.repo.stored) != 1 {
		t.Errorf("expected exactly 1 policy to be persisted, got %d", len(f.repo.stored))
	}
}

func TestAccessPolicyService_Update_RejectsCrossTenantReference(t *testing.T) {
	f := newAccessPolicyTestFixture()
	tenantA := uuid.New()
	tenantB := uuid.New()

	cwID, swID := uuid.New(), uuid.New()
	f.clientWorkloads.add(tenantA, cwID, &domain.ClientWorkload{ID: cwID, TenantID: tenantA})
	f.serverWorkloads.add(tenantA, swID, &domain.ServerWorkload{ID: swID, TenantID: tenantA})

	existing := &domain.AccessPolicy{Name: "existing", ClientWorkloadID: cwID, ServerWorkloadID: swID}
	if err := f.svc.Create(context.Background(), tenantA, existing); err != nil {
		t.Fatalf("setup create failed: %v", err)
	}

	otherTenantsTPID := uuid.New()
	f.trustProviders.add(tenantB, otherTenantsTPID, &domain.TrustProvider{ID: otherTenantsTPID, TenantID: tenantB})

	update := &domain.AccessPolicy{
		ID:               existing.ID,
		Name:             "existing",
		ClientWorkloadID: cwID,
		ServerWorkloadID: swID,
		TrustProviderIDs: []uuid.UUID{otherTenantsTPID},
	}

	err := f.svc.Update(context.Background(), tenantA, update)
	if err == nil {
		t.Fatal("expected Update to reject a cross-tenant Trust Provider reference")
	}
}

func TestAccessPolicyService_Create_NilResourceSetID_DefaultsToWellKnownID(t *testing.T) {
	f := newAccessPolicyTestFixture()
	tenantA := uuid.New()
	cwID, swID := uuid.New(), uuid.New()
	f.clientWorkloads.add(tenantA, cwID, &domain.ClientWorkload{ID: cwID, TenantID: tenantA})
	f.serverWorkloads.add(tenantA, swID, &domain.ServerWorkload{ID: swID, TenantID: tenantA})

	policy := &domain.AccessPolicy{Name: "test", ClientWorkloadID: cwID, ServerWorkloadID: swID}
	if err := f.svc.Create(context.Background(), tenantA, policy); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	stored := f.repo.stored[policy.ID]
	if stored.ResourceSetID == nil || *stored.ResourceSetID != scope.DefaultResourceSetID {
		t.Errorf("expected ResourceSetID to default to %s, got %v", scope.DefaultResourceSetID, stored.ResourceSetID)
	}
}

func TestAccessPolicyService_Update_OmittingResourceSetID_PreservesExisting(t *testing.T) {
	f := newAccessPolicyTestFixture()
	tenantA := uuid.New()
	cwID, swID := uuid.New(), uuid.New()
	f.clientWorkloads.add(tenantA, cwID, &domain.ClientWorkload{ID: cwID, TenantID: tenantA})
	f.serverWorkloads.add(tenantA, swID, &domain.ServerWorkload{ID: swID, TenantID: tenantA})

	customRS := uuid.New()
	policy := &domain.AccessPolicy{Name: "test", ClientWorkloadID: cwID, ServerWorkloadID: swID, ResourceSetID: &customRS}
	if err := f.svc.Create(context.Background(), tenantA, policy); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	update := &domain.AccessPolicy{ID: policy.ID, Name: "renamed", ClientWorkloadID: cwID, ServerWorkloadID: swID}
	if err := f.svc.Update(context.Background(), tenantA, update); err != nil {
		t.Fatalf("update failed: %v", err)
	}
	stored := f.repo.stored[policy.ID]
	if stored.ResourceSetID == nil || *stored.ResourceSetID != customRS {
		t.Errorf("expected the existing custom ResourceSetID %s to be preserved, got %v", customRS, stored.ResourceSetID)
	}
}

package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/service/identity"
)

// wellKnownDefaultResourceSetID mirrors scope.DefaultResourceSetID. Not
// imported directly to avoid a cross-package dependency in this test; the
// value itself (the well-known "all zeros/ffff" UUID every resource-scoped
// table's column default points at) must match exactly.
const wellKnownDefaultResourceSetID = "ffffffff-ffff-ffff-ffff-ffffffffffff"

type fakeTenantRepoForSignup struct {
	created []domain.Tenant
}

func (f *fakeTenantRepoForSignup) Create(_ context.Context, t *domain.Tenant) error {
	t.ID = uuid.New()
	f.created = append(f.created, *t)
	return nil
}
func (f *fakeTenantRepoForSignup) GetByID(_ context.Context, _ uuid.UUID) (*domain.Tenant, error) {
	return nil, nil
}
func (f *fakeTenantRepoForSignup) GetBySubdomain(_ context.Context, _ string) (*domain.Tenant, error) {
	return nil, nil
}
func (f *fakeTenantRepoForSignup) GetByAPITokenHash(_ context.Context, _ string) (*domain.Tenant, error) {
	return nil, nil
}
func (f *fakeTenantRepoForSignup) List(_ context.Context, _ domain.ListOptions) ([]domain.Tenant, int, error) {
	return nil, 0, nil
}
func (f *fakeTenantRepoForSignup) Update(_ context.Context, _ *domain.Tenant) error { return nil }
func (f *fakeTenantRepoForSignup) Delete(_ context.Context, _ uuid.UUID) error      { return nil }

type fakeUserRepoForSignup struct{}

func (f *fakeUserRepoForSignup) Create(_ context.Context, _ uuid.UUID, u *domain.User) error {
	u.ID = uuid.New()
	return nil
}
func (f *fakeUserRepoForSignup) GetByID(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*domain.User, error) {
	return nil, nil
}
func (f *fakeUserRepoForSignup) GetByEmail(_ context.Context, _ uuid.UUID, _ string) (*domain.User, error) {
	return nil, nil
}
func (f *fakeUserRepoForSignup) GetTenantsByEmail(_ context.Context, _ string) ([]domain.TenantSummary, error) {
	return nil, nil
}
func (f *fakeUserRepoForSignup) List(_ context.Context, _ uuid.UUID, _ domain.ListOptions) ([]domain.User, int, error) {
	return nil, 0, nil
}
func (f *fakeUserRepoForSignup) Update(_ context.Context, _ uuid.UUID, _ *domain.User) error {
	return nil
}
func (f *fakeUserRepoForSignup) Delete(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return nil
}

type fakeRoleRepoForSignup struct{}

func (f *fakeRoleRepoForSignup) Create(_ context.Context, _ uuid.UUID, r *domain.Role) error {
	r.ID = uuid.New()
	return nil
}
func (f *fakeRoleRepoForSignup) GetByID(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*domain.Role, error) {
	return nil, nil
}
func (f *fakeRoleRepoForSignup) List(_ context.Context, _ uuid.UUID, _ domain.ListOptions) ([]domain.Role, int, error) {
	return nil, 0, nil
}
func (f *fakeRoleRepoForSignup) Update(_ context.Context, _ uuid.UUID, _ *domain.Role) error {
	return nil
}
func (f *fakeRoleRepoForSignup) Delete(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return nil
}
func (f *fakeRoleRepoForSignup) AssignResourceSets(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ []uuid.UUID) error {
	return nil
}
func (f *fakeRoleRepoForSignup) GetResourceSetIDs(_ context.Context, _ uuid.UUID, _ uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}

// fakeResourceSetRepoForSignup mimics the FIXED ResourceSetRepo.Create
// behavior (respects a caller-supplied ID, only generates one if absent)
// so this test actually exercises the contract SignupService depends on,
// not a repo that happens to always generate its own ID regardless of
// input.
type fakeResourceSetRepoForSignup struct {
	created []domain.ResourceSet
}

func (f *fakeResourceSetRepoForSignup) Create(_ context.Context, tenantID uuid.UUID, rs *domain.ResourceSet) error {
	if rs.ID == uuid.Nil {
		rs.ID = uuid.New()
	}
	rs.TenantID = tenantID
	f.created = append(f.created, *rs)
	return nil
}
func (f *fakeResourceSetRepoForSignup) GetByID(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*domain.ResourceSet, error) {
	return nil, nil
}
func (f *fakeResourceSetRepoForSignup) List(_ context.Context, _ uuid.UUID, _ domain.ListOptions) ([]domain.ResourceSet, int, error) {
	return nil, 0, nil
}
func (f *fakeResourceSetRepoForSignup) Update(_ context.Context, _ uuid.UUID, _ *domain.ResourceSet) error {
	return nil
}
func (f *fakeResourceSetRepoForSignup) Delete(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return nil
}
func (f *fakeResourceSetRepoForSignup) GetRoleIDs(_ context.Context, _ uuid.UUID, _ uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}
func (f *fakeResourceSetRepoForSignup) GetResourceCounts(_ context.Context, _ uuid.UUID, _ []uuid.UUID) (map[uuid.UUID]domain.ResourceSetCounts, error) {
	return nil, nil
}

type fakeTenantProfileWriterForSignup struct{}

func (f *fakeTenantProfileWriterForSignup) Upsert(_ context.Context, _ uuid.UUID, _ *domain.TenantProfile) error {
	return nil
}

func TestSignup_CreatesDefaultResourceSetWithWellKnownID(t *testing.T) {
	tenants := &fakeTenantRepoForSignup{}
	users := &fakeUserRepoForSignup{}
	roles := &fakeRoleRepoForSignup{}
	resourceSets := &fakeResourceSetRepoForSignup{}
	profiles := &fakeTenantProfileWriterForSignup{}

	issuer, err := identity.NewIdentityIssuer("https://id.dipolehq.dev")
	if err != nil {
		t.Fatalf("failed to construct issuer: %v", err)
	}

	svc := NewSignupService(tenants, users, roles, resourceSets, profiles, issuer)

	_, err = svc.Signup(context.Background(), &SignupRequest{
		FirstName:   "Ada",
		LastName:    "Lovelace",
		CompanyName: "Acme Inc",
		Email:       "ada@acme.example",
		Password:    "correct-horse-battery",
	})
	if err != nil {
		t.Fatalf("signup failed: %v", err)
	}

	if len(resourceSets.created) != 1 {
		t.Fatalf("expected exactly 1 resource set created, got %d", len(resourceSets.created))
	}

	got := resourceSets.created[0]
	if got.ID.String() != wellKnownDefaultResourceSetID {
		t.Errorf("expected Default resource set to have the well-known ID %s, got %s — this is the exact bug where ResourceSetRepo.Create used to discard the caller-supplied ID", wellKnownDefaultResourceSetID, got.ID)
	}
	if got.Name != "Default" {
		t.Errorf("expected resource set name %q, got %q", "Default", got.Name)
	}
	if len(tenants.created) != 1 {
		t.Fatalf("expected exactly 1 tenant created, got %d", len(tenants.created))
	}
	if got.TenantID != resourceSets.created[0].TenantID {
		t.Errorf("resource set TenantID mismatch")
	}
}

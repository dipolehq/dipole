package resources

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type fakeRoleRepoForTest struct {
	stored map[uuid.UUID]*domain.Role
}

func newFakeRoleRepoForTest() *fakeRoleRepoForTest {
	return &fakeRoleRepoForTest{stored: make(map[uuid.UUID]*domain.Role)}
}

func (f *fakeRoleRepoForTest) Create(_ context.Context, _ uuid.UUID, role *domain.Role) error {
	if role.ID == uuid.Nil {
		role.ID = uuid.New()
	}
	copy := *role
	f.stored[role.ID] = &copy
	return nil
}
func (f *fakeRoleRepoForTest) GetByID(_ context.Context, _ uuid.UUID, id uuid.UUID) (*domain.Role, error) {
	return f.stored[id], nil
}
func (f *fakeRoleRepoForTest) List(_ context.Context, _ uuid.UUID, _ domain.ListOptions) ([]domain.Role, int, error) {
	return nil, 0, nil
}
func (f *fakeRoleRepoForTest) Update(_ context.Context, _ uuid.UUID, role *domain.Role) error {
	if _, ok := f.stored[role.ID]; !ok {
		return nil
	}
	copy := *role
	f.stored[role.ID] = &copy
	return nil
}
func (f *fakeRoleRepoForTest) Delete(_ context.Context, _ uuid.UUID, id uuid.UUID) error {
	delete(f.stored, id)
	return nil
}
func (f *fakeRoleRepoForTest) AssignResourceSets(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ []uuid.UUID) error {
	return nil
}
func (f *fakeRoleRepoForTest) GetResourceSetIDs(_ context.Context, _ uuid.UUID, _ uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}

func TestRoleService_Create_MissingName_ReturnsValidationError(t *testing.T) {
	svc := NewRoleService(newFakeRoleRepoForTest())

	err := svc.Create(context.Background(), uuid.New(), &domain.Role{})

	var ve *domain.ValidationError
	if !errors.As(err, &ve) || ve.Field != "name" {
		t.Fatalf("expected validation error on name, got %v", err)
	}
}

func TestRoleService_Create_TrimsFields(t *testing.T) {
	repo := newFakeRoleRepoForTest()
	svc := NewRoleService(repo)

	role := &domain.Role{Name: "  Admin  ", Description: " full access "}
	if err := svc.Create(context.Background(), uuid.New(), role); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := repo.stored[role.ID]
	if got.Name != "Admin" || got.Description != "full access" {
		t.Errorf("expected trimmed fields, got %+v", got)
	}
}

func TestRoleService_Update_MissingRecord_ReturnsNotFound(t *testing.T) {
	svc := NewRoleService(newFakeRoleRepoForTest())

	err := svc.Update(context.Background(), uuid.New(), &domain.Role{ID: uuid.New(), Name: "Admin"})

	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestRoleService_Delete_MissingRecord_ReturnsNotFound(t *testing.T) {
	svc := NewRoleService(newFakeRoleRepoForTest())

	err := svc.Delete(context.Background(), uuid.New(), uuid.New())

	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestRoleService_Get_NotFound_ReturnsErrNotFound(t *testing.T) {
	svc := NewRoleService(newFakeRoleRepoForTest())

	_, err := svc.Get(context.Background(), uuid.New(), uuid.New())

	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

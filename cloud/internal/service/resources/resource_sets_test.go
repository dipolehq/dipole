package resources

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/scope"
)

type fakeResourceSetRepoForServiceTest struct {
	stored map[uuid.UUID]*domain.ResourceSet
}

func newFakeResourceSetRepoForServiceTest() *fakeResourceSetRepoForServiceTest {
	return &fakeResourceSetRepoForServiceTest{stored: make(map[uuid.UUID]*domain.ResourceSet)}
}

func (f *fakeResourceSetRepoForServiceTest) Create(_ context.Context, _ uuid.UUID, rs *domain.ResourceSet) error {
	if rs.ID == uuid.Nil {
		rs.ID = uuid.New()
	}
	copy := *rs
	f.stored[rs.ID] = &copy
	return nil
}
func (f *fakeResourceSetRepoForServiceTest) GetByID(_ context.Context, _ uuid.UUID, id uuid.UUID) (*domain.ResourceSet, error) {
	return f.stored[id], nil
}
func (f *fakeResourceSetRepoForServiceTest) List(_ context.Context, _ uuid.UUID, _ domain.ListOptions) ([]domain.ResourceSet, int, error) {
	return nil, 0, nil
}
func (f *fakeResourceSetRepoForServiceTest) Update(_ context.Context, _ uuid.UUID, rs *domain.ResourceSet) error {
	if _, ok := f.stored[rs.ID]; !ok {
		return nil
	}
	copy := *rs
	f.stored[rs.ID] = &copy
	return nil
}
func (f *fakeResourceSetRepoForServiceTest) Delete(_ context.Context, _ uuid.UUID, id uuid.UUID) error {
	delete(f.stored, id)
	return nil
}
func (f *fakeResourceSetRepoForServiceTest) GetRoleIDs(_ context.Context, _ uuid.UUID, _ uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}
func (f *fakeResourceSetRepoForServiceTest) GetResourceCounts(_ context.Context, _ uuid.UUID, _ []uuid.UUID) (map[uuid.UUID]domain.ResourceSetCounts, error) {
	return nil, nil
}

func TestResourceSetService_Create_MissingName_ReturnsValidationError(t *testing.T) {
	svc := NewResourceSetService(newFakeResourceSetRepoForServiceTest())

	err := svc.Create(context.Background(), uuid.New(), &domain.ResourceSet{})

	var ve *domain.ValidationError
	if !errors.As(err, &ve) || ve.Field != "name" {
		t.Fatalf("expected validation error on name, got %v", err)
	}
}

func TestResourceSetService_Create_ChecksLimiterBeforeCreating(t *testing.T) {
	repo := newFakeResourceSetRepoForServiceTest()
	svc := NewResourceSetService(repo)
	svc.SetLimiter(denyingPlanLimiterForTest())

	err := svc.Create(context.Background(), uuid.New(), &domain.ResourceSet{Name: "Staging"})

	if err == nil {
		t.Fatal("expected plan limit error")
	}
	if len(repo.stored) != 0 {
		t.Error("expected no resource set to be created when the plan limit denies")
	}
}

func TestResourceSetService_Delete_DefaultResourceSet_ReturnsValidationError(t *testing.T) {
	repo := newFakeResourceSetRepoForServiceTest()
	repo.stored[scope.DefaultResourceSetID] = &domain.ResourceSet{ID: scope.DefaultResourceSetID, Name: "Default"}
	svc := NewResourceSetService(repo)

	err := svc.Delete(context.Background(), uuid.New(), scope.DefaultResourceSetID)

	var ve *domain.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected validation error, got %v", err)
	}
	if _, ok := repo.stored[scope.DefaultResourceSetID]; !ok {
		t.Error("expected the Default Resource Set to NOT be deleted")
	}
}

func TestResourceSetService_Delete_NonDefaultResourceSet_Allowed(t *testing.T) {
	repo := newFakeResourceSetRepoForServiceTest()
	customID := uuid.New()
	repo.stored[customID] = &domain.ResourceSet{ID: customID, Name: "Staging"}
	svc := NewResourceSetService(repo)

	if err := svc.Delete(context.Background(), uuid.New(), customID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := repo.stored[customID]; ok {
		t.Error("expected the resource set to be deleted")
	}
}

func TestResourceSetService_Delete_MissingRecord_ReturnsNotFound(t *testing.T) {
	svc := NewResourceSetService(newFakeResourceSetRepoForServiceTest())

	err := svc.Delete(context.Background(), uuid.New(), uuid.New())

	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestResourceSetService_Get_NotFound_ReturnsErrNotFound(t *testing.T) {
	svc := NewResourceSetService(newFakeResourceSetRepoForServiceTest())

	_, err := svc.Get(context.Background(), uuid.New(), uuid.New())

	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

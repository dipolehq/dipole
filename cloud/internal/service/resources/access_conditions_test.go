package resources

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/scope"
)

type fakeAccessConditionRepo struct {
	stored map[uuid.UUID]*domain.AccessCondition
}

func newFakeAccessConditionRepo() *fakeAccessConditionRepo {
	return &fakeAccessConditionRepo{stored: make(map[uuid.UUID]*domain.AccessCondition)}
}

func (f *fakeAccessConditionRepo) Create(_ context.Context, tenantID uuid.UUID, ac *domain.AccessCondition) error {
	if ac.ID == uuid.Nil {
		ac.ID = uuid.New()
	}
	ac.TenantID = tenantID
	copy := *ac
	f.stored[ac.ID] = &copy
	return nil
}
func (f *fakeAccessConditionRepo) GetByID(_ context.Context, _ uuid.UUID, id uuid.UUID) (*domain.AccessCondition, error) {
	return f.stored[id], nil
}
func (f *fakeAccessConditionRepo) List(_ context.Context, _ uuid.UUID, _ domain.ListOptions) ([]domain.AccessCondition, int, error) {
	return nil, 0, nil
}
func (f *fakeAccessConditionRepo) Update(_ context.Context, tenantID uuid.UUID, ac *domain.AccessCondition) error {
	if _, ok := f.stored[ac.ID]; !ok {
		return nil
	}
	copy := *ac
	copy.TenantID = tenantID
	f.stored[ac.ID] = &copy
	return nil
}
func (f *fakeAccessConditionRepo) Delete(_ context.Context, _ uuid.UUID, id uuid.UUID) error {
	delete(f.stored, id)
	return nil
}
func (f *fakeAccessConditionRepo) CountInactive(_ context.Context, _ uuid.UUID) int { return 0 }

func TestAccessConditionService_Create_TrimsNameAndDescription(t *testing.T) {
	repo := newFakeAccessConditionRepo()
	svc := NewAccessConditionService(repo)

	ac := &domain.AccessCondition{Name: "  Business Hours  ", Description: "  9-5 only  ", Type: "time_window"}
	if err := svc.Create(context.Background(), uuid.New(), ac); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stored := repo.stored[ac.ID]
	if stored.Name != "Business Hours" {
		t.Errorf("expected trimmed name, got %q", stored.Name)
	}
	if stored.Description != "9-5 only" {
		t.Errorf("expected trimmed description, got %q", stored.Description)
	}
}

func TestAccessConditionService_Create_MissingName_ReturnsValidationError(t *testing.T) {
	svc := NewAccessConditionService(newFakeAccessConditionRepo())
	err := svc.Create(context.Background(), uuid.New(), &domain.AccessCondition{Type: "time_window"})

	ve, ok := err.(*domain.ValidationError)
	if !ok {
		t.Fatalf("expected *domain.ValidationError, got %T: %v", err, err)
	}
	if ve.Field != "name" {
		t.Errorf("expected validation error on field %q, got %q", "name", ve.Field)
	}
}

func TestAccessConditionService_Create_MissingType_ReturnsValidationError(t *testing.T) {
	svc := NewAccessConditionService(newFakeAccessConditionRepo())
	err := svc.Create(context.Background(), uuid.New(), &domain.AccessCondition{Name: "test"})

	ve, ok := err.(*domain.ValidationError)
	if !ok {
		t.Fatalf("expected *domain.ValidationError, got %T: %v", err, err)
	}
	if ve.Field != "type" {
		t.Errorf("expected validation error on field %q, got %q", "type", ve.Field)
	}
}

func TestAccessConditionService_Create_ChecksLimiterBeforeCreating(t *testing.T) {
	repo := newFakeAccessConditionRepo()
	svc := NewAccessConditionService(repo)
	svc.SetLimiter(denyingPlanLimiterForTest())

	err := svc.Create(context.Background(), uuid.New(), &domain.AccessCondition{Name: "test", Type: "time_window"})
	if err == nil {
		t.Fatal("expected the limiter's error to be returned")
	}
	if len(repo.stored) != 0 {
		t.Error("expected nothing to be persisted when the limiter denies")
	}
}

func TestAccessConditionService_Update_MissingRecord_ReturnsNotFound(t *testing.T) {
	svc := NewAccessConditionService(newFakeAccessConditionRepo())
	err := svc.Update(context.Background(), uuid.New(), &domain.AccessCondition{ID: uuid.New(), Name: "test", Type: "time_window"})

	if err != domain.ErrNotFound {
		t.Errorf("expected domain.ErrNotFound, got %v", err)
	}
}

func TestAccessConditionService_Update_ValidatesBeforeCheckingExistence(t *testing.T) {
	svc := NewAccessConditionService(newFakeAccessConditionRepo())
	err := svc.Update(context.Background(), uuid.New(), &domain.AccessCondition{ID: uuid.New(), Type: "time_window"})

	if _, ok := err.(*domain.ValidationError); !ok {
		t.Errorf("expected a validation error (missing name) rather than not-found, got %T: %v", err, err)
	}
}

func TestAccessConditionService_Update_ExistingRecord_Succeeds(t *testing.T) {
	repo := newFakeAccessConditionRepo()
	svc := NewAccessConditionService(repo)
	tenantID := uuid.New()

	ac := &domain.AccessCondition{Name: "original", Type: "time_window"}
	if err := svc.Create(context.Background(), tenantID, ac); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	update := &domain.AccessCondition{ID: ac.ID, Name: "updated", Type: "time_window"}
	if err := svc.Update(context.Background(), tenantID, update); err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if repo.stored[ac.ID].Name != "updated" {
		t.Errorf("expected name to be updated, got %q", repo.stored[ac.ID].Name)
	}
}

func TestAccessConditionService_Delete_MissingRecord_ReturnsNotFound(t *testing.T) {
	svc := NewAccessConditionService(newFakeAccessConditionRepo())
	err := svc.Delete(context.Background(), uuid.New(), uuid.New())

	if err != domain.ErrNotFound {
		t.Errorf("expected domain.ErrNotFound, got %v", err)
	}
}

func TestAccessConditionService_Delete_ExistingRecord_Succeeds(t *testing.T) {
	repo := newFakeAccessConditionRepo()
	svc := NewAccessConditionService(repo)
	tenantID := uuid.New()

	ac := &domain.AccessCondition{Name: "to-delete", Type: "time_window"}
	if err := svc.Create(context.Background(), tenantID, ac); err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if err := svc.Delete(context.Background(), tenantID, ac.ID); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if _, ok := repo.stored[ac.ID]; ok {
		t.Error("expected record to be removed after delete")
	}
}

func TestAccessConditionService_Get_NotFound_ReturnsErrNotFound(t *testing.T) {
	svc := NewAccessConditionService(newFakeAccessConditionRepo())
	_, err := svc.Get(context.Background(), uuid.New(), uuid.New())

	if err != domain.ErrNotFound {
		t.Errorf("expected domain.ErrNotFound, got %v", err)
	}
}

func TestAccessConditionService_Create_NilResourceSetID_DefaultsToWellKnownID(t *testing.T) {
	repo := newFakeAccessConditionRepo()
	svc := NewAccessConditionService(repo)
	ac := &domain.AccessCondition{Name: "test", Type: "time_window"}

	if err := svc.Create(context.Background(), uuid.New(), ac); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	stored := repo.stored[ac.ID]
	if stored.ResourceSetID == nil || *stored.ResourceSetID != scope.DefaultResourceSetID {
		t.Errorf("expected ResourceSetID to default to %s, got %v", scope.DefaultResourceSetID, stored.ResourceSetID)
	}
}

func TestAccessConditionService_Update_OmittingResourceSetID_PreservesExisting(t *testing.T) {
	repo := newFakeAccessConditionRepo()
	svc := NewAccessConditionService(repo)
	tenantID := uuid.New()
	customRS := uuid.New()

	ac := &domain.AccessCondition{Name: "original", Type: "time_window", ResourceSetID: &customRS}
	if err := svc.Create(context.Background(), tenantID, ac); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	update := &domain.AccessCondition{ID: ac.ID, Name: "updated", Type: "time_window"}
	if err := svc.Update(context.Background(), tenantID, update); err != nil {
		t.Fatalf("update failed: %v", err)
	}
	stored := repo.stored[ac.ID]
	if stored.ResourceSetID == nil || *stored.ResourceSetID != customRS {
		t.Errorf("expected the existing custom ResourceSetID %s to be preserved, got %v", customRS, stored.ResourceSetID)
	}
}

package resources

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type fakeClientWorkloadRepoForTest struct {
	stored map[uuid.UUID]*domain.ClientWorkload
}

func newFakeClientWorkloadRepoForTest() *fakeClientWorkloadRepoForTest {
	return &fakeClientWorkloadRepoForTest{stored: make(map[uuid.UUID]*domain.ClientWorkload)}
}

func (f *fakeClientWorkloadRepoForTest) Create(_ context.Context, tenantID uuid.UUID, cw *domain.ClientWorkload) error {
	if cw.ID == uuid.Nil {
		cw.ID = uuid.New()
	}
	cw.TenantID = tenantID
	copy := *cw
	f.stored[cw.ID] = &copy
	return nil
}
func (f *fakeClientWorkloadRepoForTest) GetByID(_ context.Context, _ uuid.UUID, id uuid.UUID) (*domain.ClientWorkload, error) {
	return f.stored[id], nil
}
func (f *fakeClientWorkloadRepoForTest) List(_ context.Context, _ uuid.UUID, _ domain.ListOptions) ([]domain.ClientWorkload, int, error) {
	return nil, 0, nil
}
func (f *fakeClientWorkloadRepoForTest) Update(_ context.Context, tenantID uuid.UUID, cw *domain.ClientWorkload) error {
	if _, ok := f.stored[cw.ID]; !ok {
		return nil
	}
	copy := *cw
	copy.TenantID = tenantID
	f.stored[cw.ID] = &copy
	return nil
}
func (f *fakeClientWorkloadRepoForTest) Delete(_ context.Context, _ uuid.UUID, id uuid.UUID) error {
	delete(f.stored, id)
	return nil
}
func (f *fakeClientWorkloadRepoForTest) CountInactive(_ context.Context, _ uuid.UUID) int { return 0 }

func validClientWorkload(name string) *domain.ClientWorkload {
	return &domain.ClientWorkload{
		Name:        name,
		Identifiers: []domain.Identifier{{Type: "hostname", Value: "worker-1.internal"}},
	}
}

func TestClientWorkloadService_Create_TrimsFields(t *testing.T) {
	repo := newFakeClientWorkloadRepoForTest()
	svc := NewClientWorkloadService(repo)

	cw := &domain.ClientWorkload{
		Name:        "  worker  ",
		Description: "  does things  ",
		Identifiers: []domain.Identifier{{Type: "  hostname  ", Value: "  worker-1.internal  "}},
	}
	if err := svc.Create(context.Background(), uuid.New(), cw); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stored := repo.stored[cw.ID]
	if stored.Name != "worker" || stored.Description != "does things" {
		t.Errorf("expected trimmed name/description, got %q / %q", stored.Name, stored.Description)
	}
	if stored.Identifiers[0].Type != "hostname" || stored.Identifiers[0].Value != "worker-1.internal" {
		t.Errorf("expected trimmed identifier fields, got %+v", stored.Identifiers[0])
	}
}

func TestClientWorkloadService_Create_MissingName_ReturnsValidationError(t *testing.T) {
	svc := NewClientWorkloadService(newFakeClientWorkloadRepoForTest())
	cw := validClientWorkload("")

	err := svc.Create(context.Background(), uuid.New(), cw)
	ve, ok := err.(*domain.ValidationError)
	if !ok {
		t.Fatalf("expected *domain.ValidationError, got %T: %v", err, err)
	}
	if ve.Field != "name" {
		t.Errorf("expected error on field %q, got %q", "name", ve.Field)
	}
}

func TestClientWorkloadService_Create_NoIdentifiers_ReturnsValidationError(t *testing.T) {
	svc := NewClientWorkloadService(newFakeClientWorkloadRepoForTest())
	err := svc.Create(context.Background(), uuid.New(), &domain.ClientWorkload{Name: "worker"})

	ve, ok := err.(*domain.ValidationError)
	if !ok {
		t.Fatalf("expected *domain.ValidationError, got %T: %v", err, err)
	}
	if ve.Field != "identifiers" {
		t.Errorf("expected error on field %q, got %q", "identifiers", ve.Field)
	}
}

func TestClientWorkloadService_Create_UnknownIdentifierType_ReturnsValidationError(t *testing.T) {
	svc := NewClientWorkloadService(newFakeClientWorkloadRepoForTest())
	cw := &domain.ClientWorkload{
		Name:        "worker",
		Identifiers: []domain.Identifier{{Type: "not_a_real_type", Value: "x"}},
	}

	err := svc.Create(context.Background(), uuid.New(), cw)
	if _, ok := err.(*domain.ValidationError); !ok {
		t.Fatalf("expected *domain.ValidationError for an unrecognized identifier type, got %T: %v", err, err)
	}
}

func TestClientWorkloadService_Create_IdentifierMissingValue_ReturnsValidationError(t *testing.T) {
	svc := NewClientWorkloadService(newFakeClientWorkloadRepoForTest())
	cw := &domain.ClientWorkload{
		Name:        "worker",
		Identifiers: []domain.Identifier{{Type: "hostname", Value: ""}},
	}

	err := svc.Create(context.Background(), uuid.New(), cw)
	if _, ok := err.(*domain.ValidationError); !ok {
		t.Fatalf("expected *domain.ValidationError for a missing identifier value, got %T: %v", err, err)
	}
}

func TestClientWorkloadService_Create_ValidWorkload_Succeeds(t *testing.T) {
	repo := newFakeClientWorkloadRepoForTest()
	svc := NewClientWorkloadService(repo)

	cw := validClientWorkload("worker")
	if err := svc.Create(context.Background(), uuid.New(), cw); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.stored) != 1 {
		t.Errorf("expected exactly 1 workload to be persisted, got %d", len(repo.stored))
	}
}

func TestClientWorkloadService_Create_ChecksLimiterBeforeCreating(t *testing.T) {
	repo := newFakeClientWorkloadRepoForTest()
	svc := NewClientWorkloadService(repo)
	svc.SetLimiter(denyingPlanLimiterForTest())

	err := svc.Create(context.Background(), uuid.New(), validClientWorkload("worker"))
	if err == nil {
		t.Fatal("expected the limiter's error to be returned")
	}
	if len(repo.stored) != 0 {
		t.Error("expected nothing to be persisted when the limiter denies")
	}
}

func TestClientWorkloadService_Update_MissingRecord_ReturnsNotFound(t *testing.T) {
	svc := NewClientWorkloadService(newFakeClientWorkloadRepoForTest())
	cw := validClientWorkload("worker")
	cw.ID = uuid.New()

	err := svc.Update(context.Background(), uuid.New(), cw)
	if err != domain.ErrNotFound {
		t.Errorf("expected domain.ErrNotFound, got %v", err)
	}
}

func TestClientWorkloadService_Update_InvalidWorkload_ReturnsValidationErrorBeforeLookup(t *testing.T) {
	svc := NewClientWorkloadService(newFakeClientWorkloadRepoForTest())
	cw := &domain.ClientWorkload{ID: uuid.New()} // no name, no identifiers

	err := svc.Update(context.Background(), uuid.New(), cw)
	if _, ok := err.(*domain.ValidationError); !ok {
		t.Errorf("expected a validation error rather than not-found, got %T: %v", err, err)
	}
}

func TestClientWorkloadService_Delete_MissingRecord_ReturnsNotFound(t *testing.T) {
	svc := NewClientWorkloadService(newFakeClientWorkloadRepoForTest())
	err := svc.Delete(context.Background(), uuid.New(), uuid.New())

	if err != domain.ErrNotFound {
		t.Errorf("expected domain.ErrNotFound, got %v", err)
	}
}

func TestClientWorkloadService_Get_NotFound_ReturnsErrNotFound(t *testing.T) {
	svc := NewClientWorkloadService(newFakeClientWorkloadRepoForTest())
	_, err := svc.Get(context.Background(), uuid.New(), uuid.New())

	if err != domain.ErrNotFound {
		t.Errorf("expected domain.ErrNotFound, got %v", err)
	}
}

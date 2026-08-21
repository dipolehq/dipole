package resources

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/scope"
)

type fakeServerWorkloadRepoForTest struct {
	stored map[uuid.UUID]*domain.ServerWorkload
}

func newFakeServerWorkloadRepoForTest() *fakeServerWorkloadRepoForTest {
	return &fakeServerWorkloadRepoForTest{stored: make(map[uuid.UUID]*domain.ServerWorkload)}
}

func (f *fakeServerWorkloadRepoForTest) Create(_ context.Context, tenantID uuid.UUID, sw *domain.ServerWorkload) error {
	if sw.ID == uuid.Nil {
		sw.ID = uuid.New()
	}
	sw.TenantID = tenantID
	copy := *sw
	f.stored[sw.ID] = &copy
	return nil
}
func (f *fakeServerWorkloadRepoForTest) GetByID(_ context.Context, _ uuid.UUID, id uuid.UUID) (*domain.ServerWorkload, error) {
	return f.stored[id], nil
}
func (f *fakeServerWorkloadRepoForTest) List(_ context.Context, _ uuid.UUID, _ domain.ListOptions) ([]domain.ServerWorkload, int, error) {
	return nil, 0, nil
}
func (f *fakeServerWorkloadRepoForTest) Update(_ context.Context, tenantID uuid.UUID, sw *domain.ServerWorkload) error {
	if _, ok := f.stored[sw.ID]; !ok {
		return nil
	}
	copy := *sw
	copy.TenantID = tenantID
	f.stored[sw.ID] = &copy
	return nil
}
func (f *fakeServerWorkloadRepoForTest) Delete(_ context.Context, _ uuid.UUID, id uuid.UUID) error {
	delete(f.stored, id)
	return nil
}
func (f *fakeServerWorkloadRepoForTest) CountInactive(_ context.Context, _ uuid.UUID) int { return 0 }

func validServerWorkload(name string) *domain.ServerWorkload {
	return &domain.ServerWorkload{Name: name, Host: "api.internal", Port: 443}
}

func TestServerWorkloadService_Create_TrimsFieldsAndDefaultsTransportProtocol(t *testing.T) {
	repo := newFakeServerWorkloadRepoForTest()
	svc := NewServerWorkloadService(repo)

	sw := &domain.ServerWorkload{Name: "  api  ", Description: "  backend  ", Host: "  api.internal  ", Port: 443}
	if err := svc.Create(context.Background(), uuid.New(), sw); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stored := repo.stored[sw.ID]
	if stored.Name != "api" || stored.Description != "backend" || stored.Host != "api.internal" {
		t.Errorf("expected trimmed fields, got name=%q description=%q host=%q", stored.Name, stored.Description, stored.Host)
	}
	if stored.TransportProtocol != "TCP" {
		t.Errorf("expected TransportProtocol to default to TCP, got %q", stored.TransportProtocol)
	}
}

func TestServerWorkloadService_Create_PreservesExplicitTransportProtocol(t *testing.T) {
	repo := newFakeServerWorkloadRepoForTest()
	svc := NewServerWorkloadService(repo)

	sw := validServerWorkload("api")
	sw.TransportProtocol = "UDP"
	if err := svc.Create(context.Background(), uuid.New(), sw); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.stored[sw.ID].TransportProtocol != "UDP" {
		t.Errorf("expected explicit TransportProtocol to be preserved, got %q", repo.stored[sw.ID].TransportProtocol)
	}
}

func TestServerWorkloadService_Create_MissingName_ReturnsValidationError(t *testing.T) {
	svc := NewServerWorkloadService(newFakeServerWorkloadRepoForTest())
	sw := validServerWorkload("")

	err := svc.Create(context.Background(), uuid.New(), sw)
	ve, ok := err.(*domain.ValidationError)
	if !ok {
		t.Fatalf("expected *domain.ValidationError, got %T: %v", err, err)
	}
	if ve.Field != "name" {
		t.Errorf("expected error on field %q, got %q", "name", ve.Field)
	}
}

func TestServerWorkloadService_Create_MissingHost_ReturnsValidationError(t *testing.T) {
	svc := NewServerWorkloadService(newFakeServerWorkloadRepoForTest())
	sw := &domain.ServerWorkload{Name: "api", Port: 443}

	err := svc.Create(context.Background(), uuid.New(), sw)
	ve, ok := err.(*domain.ValidationError)
	if !ok {
		t.Fatalf("expected *domain.ValidationError, got %T: %v", err, err)
	}
	if ve.Field != "host" {
		t.Errorf("expected error on field %q, got %q", "host", ve.Field)
	}
}

func TestServerWorkloadService_Create_InvalidPort_ReturnsValidationError(t *testing.T) {
	cases := []struct {
		name string
		port int
	}{
		{"zero", 0},
		{"negative", -1},
		{"too large", 65536},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewServerWorkloadService(newFakeServerWorkloadRepoForTest())
			sw := &domain.ServerWorkload{Name: "api", Host: "api.internal", Port: tc.port}

			err := svc.Create(context.Background(), uuid.New(), sw)
			ve, ok := err.(*domain.ValidationError)
			if !ok {
				t.Fatalf("expected *domain.ValidationError for port=%d, got %T: %v", tc.port, err, err)
			}
			if ve.Field != "port" {
				t.Errorf("expected error on field %q, got %q", "port", ve.Field)
			}
		})
	}
}

func TestServerWorkloadService_Create_InvalidRequestedPort_ReturnsValidationError(t *testing.T) {
	svc := NewServerWorkloadService(newFakeServerWorkloadRepoForTest())
	sw := validServerWorkload("api")
	sw.RequestedPort = 70000

	err := svc.Create(context.Background(), uuid.New(), sw)
	ve, ok := err.(*domain.ValidationError)
	if !ok {
		t.Fatalf("expected *domain.ValidationError, got %T: %v", err, err)
	}
	if ve.Field != "forwardToPort" {
		t.Errorf("expected error on field %q, got %q", "forwardToPort", ve.Field)
	}
}

func TestServerWorkloadService_Create_ZeroRequestedPortIsValid(t *testing.T) {
	repo := newFakeServerWorkloadRepoForTest()
	svc := NewServerWorkloadService(repo)
	sw := validServerWorkload("api") // RequestedPort defaults to 0 (unset)

	if err := svc.Create(context.Background(), uuid.New(), sw); err != nil {
		t.Fatalf("expected zero RequestedPort (meaning 'not forwarding') to be valid, got: %v", err)
	}
}

func TestServerWorkloadService_Create_ChecksLimiterBeforeCreating(t *testing.T) {
	repo := newFakeServerWorkloadRepoForTest()
	svc := NewServerWorkloadService(repo)
	svc.SetLimiter(denyingPlanLimiterForTest())

	err := svc.Create(context.Background(), uuid.New(), validServerWorkload("api"))
	if err == nil {
		t.Fatal("expected the limiter's error to be returned")
	}
	if len(repo.stored) != 0 {
		t.Error("expected nothing to be persisted when the limiter denies")
	}
}

func TestServerWorkloadService_Update_MissingRecord_ReturnsNotFound(t *testing.T) {
	svc := NewServerWorkloadService(newFakeServerWorkloadRepoForTest())
	sw := validServerWorkload("api")
	sw.ID = uuid.New()

	err := svc.Update(context.Background(), uuid.New(), sw)
	if err != domain.ErrNotFound {
		t.Errorf("expected domain.ErrNotFound, got %v", err)
	}
}

func TestServerWorkloadService_Update_ExistingRecord_Succeeds(t *testing.T) {
	repo := newFakeServerWorkloadRepoForTest()
	svc := NewServerWorkloadService(repo)
	tenantID := uuid.New()

	sw := validServerWorkload("api")
	if err := svc.Create(context.Background(), tenantID, sw); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	update := validServerWorkload("api-renamed")
	update.ID = sw.ID
	if err := svc.Update(context.Background(), tenantID, update); err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if repo.stored[sw.ID].Name != "api-renamed" {
		t.Errorf("expected name to be updated, got %q", repo.stored[sw.ID].Name)
	}
}

func TestServerWorkloadService_Delete_MissingRecord_ReturnsNotFound(t *testing.T) {
	svc := NewServerWorkloadService(newFakeServerWorkloadRepoForTest())
	err := svc.Delete(context.Background(), uuid.New(), uuid.New())

	if err != domain.ErrNotFound {
		t.Errorf("expected domain.ErrNotFound, got %v", err)
	}
}

func TestServerWorkloadService_Get_NotFound_ReturnsErrNotFound(t *testing.T) {
	svc := NewServerWorkloadService(newFakeServerWorkloadRepoForTest())
	_, err := svc.Get(context.Background(), uuid.New(), uuid.New())

	if err != domain.ErrNotFound {
		t.Errorf("expected domain.ErrNotFound, got %v", err)
	}
}

func TestServerWorkloadService_Create_NilResourceSetID_DefaultsToWellKnownID(t *testing.T) {
	repo := newFakeServerWorkloadRepoForTest()
	svc := NewServerWorkloadService(repo)

	sw := validServerWorkload("api")
	if err := svc.Create(context.Background(), uuid.New(), sw); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	stored := repo.stored[sw.ID]
	if stored.ResourceSetID == nil || *stored.ResourceSetID != scope.DefaultResourceSetID {
		t.Errorf("expected ResourceSetID to default to %s, got %v", scope.DefaultResourceSetID, stored.ResourceSetID)
	}
}

func TestServerWorkloadService_Update_OmittingResourceSetID_PreservesExisting(t *testing.T) {
	repo := newFakeServerWorkloadRepoForTest()
	svc := NewServerWorkloadService(repo)

	customRS := uuid.New()
	sw := validServerWorkload("api")
	sw.ResourceSetID = &customRS
	if err := svc.Create(context.Background(), uuid.New(), sw); err != nil {
		t.Fatalf("unexpected error on create: %v", err)
	}

	update := validServerWorkload("api-renamed")
	update.ID = sw.ID
	if err := svc.Update(context.Background(), sw.TenantID, update); err != nil {
		t.Fatalf("unexpected error on update: %v", err)
	}
	stored := repo.stored[sw.ID]
	if stored.ResourceSetID == nil || *stored.ResourceSetID != customRS {
		t.Errorf("expected the existing custom ResourceSetID %s to be preserved, got %v", customRS, stored.ResourceSetID)
	}
}

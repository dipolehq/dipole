package resources

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type fakeLogStreamRepoForTest struct {
	stored map[uuid.UUID]*domain.LogStream
}

func newFakeLogStreamRepoForTest() *fakeLogStreamRepoForTest {
	return &fakeLogStreamRepoForTest{stored: make(map[uuid.UUID]*domain.LogStream)}
}

func (f *fakeLogStreamRepoForTest) Create(_ context.Context, _ uuid.UUID, ls *domain.LogStream) error {
	if ls.ID == uuid.Nil {
		ls.ID = uuid.New()
	}
	copy := *ls
	f.stored[ls.ID] = &copy
	return nil
}
func (f *fakeLogStreamRepoForTest) GetByID(_ context.Context, _ uuid.UUID, id uuid.UUID) (*domain.LogStream, error) {
	return f.stored[id], nil
}
func (f *fakeLogStreamRepoForTest) List(_ context.Context, _ uuid.UUID, _ domain.ListOptions) ([]domain.LogStream, int, error) {
	return nil, 0, nil
}
func (f *fakeLogStreamRepoForTest) ListAllActive(_ context.Context) ([]domain.LogStream, error) {
	return nil, nil
}
func (f *fakeLogStreamRepoForTest) Update(_ context.Context, _ uuid.UUID, ls *domain.LogStream) error {
	if _, ok := f.stored[ls.ID]; !ok {
		return nil
	}
	copy := *ls
	f.stored[ls.ID] = &copy
	return nil
}
func (f *fakeLogStreamRepoForTest) Delete(_ context.Context, _ uuid.UUID, id uuid.UUID) error {
	delete(f.stored, id)
	return nil
}
func (f *fakeLogStreamRepoForTest) UpdateLastExportedAt(_ context.Context, _ uuid.UUID, _ time.Time) error {
	return nil
}
func (f *fakeLogStreamRepoForTest) SetLastExportError(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}

func validLogStream(name string) *domain.LogStream {
	return &domain.LogStream{Name: name, EventType: "audit_logs", DestinationType: "aws_s3"}
}

func TestLogStreamService_Create_MissingName_ReturnsValidationError(t *testing.T) {
	svc := NewLogStreamService(newFakeLogStreamRepoForTest())

	ls := validLogStream("")
	err := svc.Create(context.Background(), uuid.New(), ls)

	var ve *domain.ValidationError
	if !errors.As(err, &ve) || ve.Field != "name" {
		t.Fatalf("expected validation error on name, got %v", err)
	}
}

func TestLogStreamService_Create_InvalidEventType_ReturnsValidationError(t *testing.T) {
	svc := NewLogStreamService(newFakeLogStreamRepoForTest())

	ls := validLogStream("siem-export")
	ls.EventType = "bogus"
	err := svc.Create(context.Background(), uuid.New(), ls)

	var ve *domain.ValidationError
	if !errors.As(err, &ve) || ve.Field != "eventType" {
		t.Fatalf("expected validation error on eventType, got %v", err)
	}
}

func TestLogStreamService_Create_InvalidDestinationType_ReturnsValidationError(t *testing.T) {
	svc := NewLogStreamService(newFakeLogStreamRepoForTest())

	ls := validLogStream("siem-export")
	ls.DestinationType = "bogus"
	err := svc.Create(context.Background(), uuid.New(), ls)

	var ve *domain.ValidationError
	if !errors.As(err, &ve) || ve.Field != "destinationType" {
		t.Fatalf("expected validation error on destinationType, got %v", err)
	}
}

func TestLogStreamService_Create_ValidStream_Succeeds(t *testing.T) {
	repo := newFakeLogStreamRepoForTest()
	svc := NewLogStreamService(repo)

	ls := validLogStream("siem-export")
	if err := svc.Create(context.Background(), uuid.New(), ls); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := repo.stored[ls.ID]; !ok {
		t.Error("expected log stream to be created")
	}
}

func TestLogStreamService_Create_ChecksLimiterBeforeCreating(t *testing.T) {
	repo := newFakeLogStreamRepoForTest()
	svc := NewLogStreamService(repo)
	svc.SetLimiter(denyingPlanLimiterForTest())

	err := svc.Create(context.Background(), uuid.New(), validLogStream("siem-export"))

	if err == nil {
		t.Fatal("expected plan limit error")
	}
	if len(repo.stored) != 0 {
		t.Error("expected no log stream to be created when the plan limit denies")
	}
}

func TestLogStreamService_Update_MissingRecord_ReturnsNotFound(t *testing.T) {
	svc := NewLogStreamService(newFakeLogStreamRepoForTest())

	ls := validLogStream("siem-export")
	ls.ID = uuid.New()
	err := svc.Update(context.Background(), uuid.New(), ls)

	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestLogStreamService_Delete_MissingRecord_ReturnsNotFound(t *testing.T) {
	svc := NewLogStreamService(newFakeLogStreamRepoForTest())

	err := svc.Delete(context.Background(), uuid.New(), uuid.New())

	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestLogStreamService_Get_NotFound_ReturnsErrNotFound(t *testing.T) {
	svc := NewLogStreamService(newFakeLogStreamRepoForTest())

	_, err := svc.Get(context.Background(), uuid.New(), uuid.New())

	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

package resources

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type fakeUserRepoForTest struct {
	stored map[uuid.UUID]*domain.User
}

func newFakeUserRepoForTest() *fakeUserRepoForTest {
	return &fakeUserRepoForTest{stored: make(map[uuid.UUID]*domain.User)}
}

func (f *fakeUserRepoForTest) Create(_ context.Context, _ uuid.UUID, u *domain.User) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	copy := *u
	f.stored[u.ID] = &copy
	return nil
}
func (f *fakeUserRepoForTest) GetByID(_ context.Context, _ uuid.UUID, id uuid.UUID) (*domain.User, error) {
	return f.stored[id], nil
}
func (f *fakeUserRepoForTest) GetByEmail(_ context.Context, _ uuid.UUID, email string) (*domain.User, error) {
	for _, u := range f.stored {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, nil
}
func (f *fakeUserRepoForTest) GetTenantsByEmail(_ context.Context, _ string) ([]domain.TenantSummary, error) {
	return nil, nil
}
func (f *fakeUserRepoForTest) List(_ context.Context, _ uuid.UUID, _ domain.ListOptions) ([]domain.User, int, error) {
	return nil, 0, nil
}
func (f *fakeUserRepoForTest) Update(_ context.Context, _ uuid.UUID, u *domain.User) error {
	if _, ok := f.stored[u.ID]; !ok {
		return nil
	}
	copy := *u
	f.stored[u.ID] = &copy
	return nil
}
func (f *fakeUserRepoForTest) Delete(_ context.Context, _ uuid.UUID, id uuid.UUID) error {
	delete(f.stored, id)
	return nil
}

func TestUserService_Create_MissingEmail_ReturnsValidationError(t *testing.T) {
	svc := NewUserService(newFakeUserRepoForTest())

	err := svc.Create(context.Background(), uuid.New(), &domain.User{FirstName: "Ada"})

	var ve *domain.ValidationError
	if !errors.As(err, &ve) || ve.Field != "email" {
		t.Fatalf("expected validation error on email, got %v", err)
	}
}

func TestUserService_Create_MissingFirstName_ReturnsValidationError(t *testing.T) {
	svc := NewUserService(newFakeUserRepoForTest())

	err := svc.Create(context.Background(), uuid.New(), &domain.User{Email: "a@b.com"})

	var ve *domain.ValidationError
	if !errors.As(err, &ve) || ve.Field != "firstName" {
		t.Fatalf("expected validation error on firstName, got %v", err)
	}
}

func TestUserService_Create_TrimsFields(t *testing.T) {
	repo := newFakeUserRepoForTest()
	svc := NewUserService(repo)

	u := &domain.User{Email: "  a@b.com  ", FirstName: " Ada ", LastName: " Lovelace "}
	if err := svc.Create(context.Background(), uuid.New(), u); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := repo.stored[u.ID]
	if got.Email != "a@b.com" || got.FirstName != "Ada" || got.LastName != "Lovelace" {
		t.Errorf("expected trimmed fields, got %+v", got)
	}
}

func TestUserService_Create_ChecksLimiterBeforeCreating(t *testing.T) {
	repo := newFakeUserRepoForTest()
	svc := NewUserService(repo)
	svc.SetLimiter(denyingPlanLimiterForTest())

	err := svc.Create(context.Background(), uuid.New(), &domain.User{Email: "a@b.com", FirstName: "Ada"})

	if err == nil {
		t.Fatal("expected plan limit error")
	}
	if len(repo.stored) != 0 {
		t.Error("expected no user to be created when the plan limit denies")
	}
}

func TestUserService_Update_MissingRecord_ReturnsNotFound(t *testing.T) {
	svc := NewUserService(newFakeUserRepoForTest())

	err := svc.Update(context.Background(), uuid.New(), &domain.User{ID: uuid.New(), Email: "a@b.com", FirstName: "Ada"})

	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestUserService_Delete_MissingRecord_ReturnsNotFound(t *testing.T) {
	svc := NewUserService(newFakeUserRepoForTest())

	err := svc.Delete(context.Background(), uuid.New(), uuid.New())

	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestUserService_Get_NotFound_ReturnsErrNotFound(t *testing.T) {
	svc := NewUserService(newFakeUserRepoForTest())

	_, err := svc.Get(context.Background(), uuid.New(), uuid.New())

	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

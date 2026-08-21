package resources

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type fakeIdentityProviderRepoForTest struct {
	stored map[uuid.UUID]*domain.IdentityProvider
}

func newFakeIdentityProviderRepoForTest() *fakeIdentityProviderRepoForTest {
	return &fakeIdentityProviderRepoForTest{stored: make(map[uuid.UUID]*domain.IdentityProvider)}
}

func (f *fakeIdentityProviderRepoForTest) Create(_ context.Context, _ uuid.UUID, idp *domain.IdentityProvider) error {
	if idp.ID == uuid.Nil {
		idp.ID = uuid.New()
	}
	copy := *idp
	f.stored[idp.ID] = &copy
	return nil
}
func (f *fakeIdentityProviderRepoForTest) GetByID(_ context.Context, _ uuid.UUID, id uuid.UUID) (*domain.IdentityProvider, error) {
	return f.stored[id], nil
}
func (f *fakeIdentityProviderRepoForTest) List(_ context.Context, _ uuid.UUID, _ domain.ListOptions) ([]domain.IdentityProvider, int, error) {
	return nil, 0, nil
}
func (f *fakeIdentityProviderRepoForTest) Update(_ context.Context, _ uuid.UUID, idp *domain.IdentityProvider) error {
	if _, ok := f.stored[idp.ID]; !ok {
		return nil
	}
	copy := *idp
	f.stored[idp.ID] = &copy
	return nil
}
func (f *fakeIdentityProviderRepoForTest) Delete(_ context.Context, _ uuid.UUID, id uuid.UUID) error {
	delete(f.stored, id)
	return nil
}

func TestIdentityProviderService_Create_MissingName_ReturnsValidationError(t *testing.T) {
	svc := NewIdentityProviderService(newFakeIdentityProviderRepoForTest())

	err := svc.Create(context.Background(), uuid.New(), &domain.IdentityProvider{Type: "oidc_1.0"})

	var ve *domain.ValidationError
	if !errors.As(err, &ve) || ve.Field != "name" {
		t.Fatalf("expected validation error on name, got %v", err)
	}
}

func TestIdentityProviderService_Create_MissingType_ReturnsValidationError(t *testing.T) {
	svc := NewIdentityProviderService(newFakeIdentityProviderRepoForTest())

	err := svc.Create(context.Background(), uuid.New(), &domain.IdentityProvider{Name: "Okta"})

	var ve *domain.ValidationError
	if !errors.As(err, &ve) || ve.Field != "type" {
		t.Fatalf("expected validation error on type, got %v", err)
	}
}

func TestIdentityProviderService_Create_InvalidType_ReturnsValidationError(t *testing.T) {
	svc := NewIdentityProviderService(newFakeIdentityProviderRepoForTest())

	err := svc.Create(context.Background(), uuid.New(), &domain.IdentityProvider{Name: "Okta", Type: "ldap"})

	var ve *domain.ValidationError
	if !errors.As(err, &ve) || ve.Field != "type" {
		t.Fatalf("expected validation error on type, got %v", err)
	}
}

func TestIdentityProviderService_Update_MissingRecord_ReturnsNotFound(t *testing.T) {
	svc := NewIdentityProviderService(newFakeIdentityProviderRepoForTest())

	err := svc.Update(context.Background(), uuid.New(), &domain.IdentityProvider{ID: uuid.New(), Name: "Okta", Type: "oidc_1.0"})

	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestIdentityProviderService_Update_PreservesSecretWhenIncomingIsBlank(t *testing.T) {
	repo := newFakeIdentityProviderRepoForTest()
	svc := NewIdentityProviderService(repo)

	existing := &domain.IdentityProvider{
		Name: "Okta", Type: "oidc_1.0",
		Config: json.RawMessage(`{"clientId":"abc","clientSecret":"super-secret"}`),
	}
	if err := svc.Create(context.Background(), uuid.New(), existing); err != nil {
		t.Fatalf("setup create failed: %v", err)
	}

	update := &domain.IdentityProvider{
		ID: existing.ID, Name: "Okta", Type: "oidc_1.0",
		Config: json.RawMessage(`{"clientId":"abc","clientSecret":""}`),
	}
	if err := svc.Update(context.Background(), uuid.New(), update); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got map[string]string
	if err := json.Unmarshal(update.Config, &got); err != nil {
		t.Fatalf("failed to parse merged config: %v", err)
	}
	if got["clientSecret"] != "super-secret" {
		t.Errorf("expected existing clientSecret to be preserved when incoming value is blank, got %q", got["clientSecret"])
	}
}

func TestIdentityProviderService_Delete_MissingRecord_ReturnsNotFound(t *testing.T) {
	svc := NewIdentityProviderService(newFakeIdentityProviderRepoForTest())

	err := svc.Delete(context.Background(), uuid.New(), uuid.New())

	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestIdentityProviderService_Get_NotFound_ReturnsErrNotFound(t *testing.T) {
	svc := NewIdentityProviderService(newFakeIdentityProviderRepoForTest())

	_, err := svc.Get(context.Background(), uuid.New(), uuid.New())

	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

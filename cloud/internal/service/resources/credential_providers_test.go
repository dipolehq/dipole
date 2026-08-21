package resources

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/crypto"
	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/scope"
)

type fakeCredentialProviderRepo struct {
	stored map[uuid.UUID]*domain.CredentialProvider
}

func newFakeCredentialProviderRepo() *fakeCredentialProviderRepo {
	return &fakeCredentialProviderRepo{stored: make(map[uuid.UUID]*domain.CredentialProvider)}
}

func (f *fakeCredentialProviderRepo) Create(_ context.Context, _ uuid.UUID, cp *domain.CredentialProvider) error {
	if cp.ID == uuid.Nil {
		cp.ID = uuid.New()
	}
	copy := *cp
	f.stored[cp.ID] = &copy
	return nil
}
func (f *fakeCredentialProviderRepo) GetByID(_ context.Context, _ uuid.UUID, id uuid.UUID) (*domain.CredentialProvider, error) {
	cp, ok := f.stored[id]
	if !ok {
		return nil, nil
	}
	copy := *cp
	return &copy, nil
}
func (f *fakeCredentialProviderRepo) List(_ context.Context, _ uuid.UUID, _ domain.ListOptions) ([]domain.CredentialProvider, int, error) {
	return nil, 0, nil
}
func (f *fakeCredentialProviderRepo) Update(_ context.Context, _ uuid.UUID, cp *domain.CredentialProvider) error {
	copy := *cp
	f.stored[cp.ID] = &copy
	return nil
}
func (f *fakeCredentialProviderRepo) Delete(_ context.Context, _ uuid.UUID, id uuid.UUID) error {
	delete(f.stored, id)
	return nil
}
func (f *fakeCredentialProviderRepo) CountInactive(_ context.Context, _ uuid.UUID) int { return 0 }

func TestCredentialProviderService_Get_DecryptFailure_ReturnsError(t *testing.T) {
	repo := newFakeCredentialProviderRepo()
	tenantID := uuid.New()
	cp := &domain.CredentialProvider{Name: "Broken", Type: "api_key"}
	if err := repo.Create(context.Background(), tenantID, cp); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	wrongKey := []byte("00000000000000000000000000000000")[:32]
	encrypted, err := crypto.Encrypt([]byte(`{"apiKey":"secret"}`), wrongKey)
	if err != nil {
		t.Fatalf("encrypt with wrong key failed: %v", err)
	}
	repo.stored[cp.ID].ConfigEncrypted = encrypted

	svc := NewCredentialProviderService(repo, testEncryptionKey)
	_, _, err = svc.Get(context.Background(), tenantID, cp.ID)

	if err == nil {
		t.Fatal("expected an error when decryption fails, got nil (silently swallowed)")
	}
}

func TestCredentialProviderService_Get_NoConfig_ReturnsEmptyObject(t *testing.T) {
	repo := newFakeCredentialProviderRepo()
	tenantID := uuid.New()
	cp := &domain.CredentialProvider{Name: "No Config", Type: "api_key"}
	if err := repo.Create(context.Background(), tenantID, cp); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	svc := NewCredentialProviderService(repo, testEncryptionKey)
	_, config, err := svc.Get(context.Background(), tenantID, cp.ID)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(config) != "{}" {
		t.Errorf("expected empty object for a provider with no stored config, got %q", config)
	}
}

func TestCredentialProviderService_Get_ValidConfig_DecryptsSuccessfully(t *testing.T) {
	repo := newFakeCredentialProviderRepo()
	tenantID := uuid.New()
	cp := &domain.CredentialProvider{Name: "Valid", Type: "api_key"}

	svc := NewCredentialProviderService(repo, testEncryptionKey)
	if err := svc.Create(context.Background(), tenantID, cp, json.RawMessage(`{"apiKey":"secret-value"}`)); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	_, config, err := svc.Get(context.Background(), tenantID, cp.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]string
	if err := json.Unmarshal(config, &parsed); err != nil {
		t.Fatalf("decrypted config is not valid JSON: %v", err)
	}
	if parsed["apiKey"] != "secret-value" {
		t.Errorf("expected decrypted apiKey to round-trip, got %q", parsed["apiKey"])
	}
}

func TestCredentialProviderService_Get_NotFound_ReturnsErrNotFound(t *testing.T) {
	repo := newFakeCredentialProviderRepo()
	svc := NewCredentialProviderService(repo, testEncryptionKey)

	_, _, err := svc.Get(context.Background(), uuid.New(), uuid.New())
	if err != domain.ErrNotFound {
		t.Errorf("expected domain.ErrNotFound, got %v", err)
	}
}

func TestCredentialProviderService_Create_NilResourceSetID_DefaultsToWellKnownID(t *testing.T) {
	repo := newFakeCredentialProviderRepo()
	svc := NewCredentialProviderService(repo, testEncryptionKey)
	cp := &domain.CredentialProvider{Name: "Test", Type: "api_key"}

	if err := svc.Create(context.Background(), uuid.New(), cp, json.RawMessage(`{}`)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	stored := repo.stored[cp.ID]
	if stored.ResourceSetID == nil || *stored.ResourceSetID != scope.DefaultResourceSetID {
		t.Errorf("expected ResourceSetID to default to %s, got %v", scope.DefaultResourceSetID, stored.ResourceSetID)
	}
}

func TestCredentialProviderService_Update_OmittingResourceSetID_PreservesExisting(t *testing.T) {
	repo := newFakeCredentialProviderRepo()
	svc := NewCredentialProviderService(repo, testEncryptionKey)
	tenantID := uuid.New()
	customRS := uuid.New()

	cp := &domain.CredentialProvider{Name: "Test", Type: "api_key", ResourceSetID: &customRS}
	if err := svc.Create(context.Background(), tenantID, cp, json.RawMessage(`{}`)); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	update := &domain.CredentialProvider{ID: cp.ID, Name: "Test Renamed", Type: "api_key"}
	if err := svc.Update(context.Background(), tenantID, update, json.RawMessage(`{}`)); err != nil {
		t.Fatalf("update failed: %v", err)
	}

	stored := repo.stored[cp.ID]
	if stored.ResourceSetID == nil || *stored.ResourceSetID != customRS {
		t.Errorf("expected the existing custom ResourceSetID %s to be preserved, got %v", customRS, stored.ResourceSetID)
	}
}

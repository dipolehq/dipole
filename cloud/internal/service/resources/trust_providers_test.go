package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/crypto"
	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/platform"
)

var testEncryptionKey = []byte("01234567890123456789012345678901")[:32]

type fakeTrustProviderRepo struct {
	stored map[uuid.UUID]*domain.TrustProvider
}

func newFakeTrustProviderRepo() *fakeTrustProviderRepo {
	return &fakeTrustProviderRepo{stored: make(map[uuid.UUID]*domain.TrustProvider)}
}

func (f *fakeTrustProviderRepo) Create(_ context.Context, _ uuid.UUID, tp *domain.TrustProvider) error {
	copy := *tp
	f.stored[tp.ID] = &copy
	return nil
}
func (f *fakeTrustProviderRepo) GetByID(_ context.Context, _ uuid.UUID, id uuid.UUID) (*domain.TrustProvider, error) {
	tp, ok := f.stored[id]
	if !ok {
		return nil, nil
	}
	copy := *tp
	return &copy, nil
}
func (f *fakeTrustProviderRepo) List(_ context.Context, _ uuid.UUID, _ domain.ListOptions) ([]domain.TrustProvider, int, error) {
	return nil, 0, nil
}
func (f *fakeTrustProviderRepo) Update(_ context.Context, _ uuid.UUID, tp *domain.TrustProvider) error {
	if _, ok := f.stored[tp.ID]; !ok {
		return fmt.Errorf("not found")
	}
	copy := *tp
	f.stored[tp.ID] = &copy
	return nil
}
func (f *fakeTrustProviderRepo) Delete(_ context.Context, _ uuid.UUID, id uuid.UUID) error {
	delete(f.stored, id)
	return nil
}
func (f *fakeTrustProviderRepo) CountInactive(_ context.Context, _ uuid.UUID) int { return 0 }

type fakeTenantRepo struct {
	tenant *domain.Tenant
}

func (f *fakeTenantRepo) Create(_ context.Context, _ *domain.Tenant) error { return nil }
func (f *fakeTenantRepo) GetByID(_ context.Context, _ uuid.UUID) (*domain.Tenant, error) {
	return f.tenant, nil
}
func (f *fakeTenantRepo) GetBySubdomain(_ context.Context, _ string) (*domain.Tenant, error) {
	return f.tenant, nil
}
func (f *fakeTenantRepo) GetByAPITokenHash(_ context.Context, _ string) (*domain.Tenant, error) {
	return f.tenant, nil
}
func (f *fakeTenantRepo) List(_ context.Context, _ domain.ListOptions) ([]domain.Tenant, int, error) {
	return nil, 0, nil
}
func (f *fakeTenantRepo) Update(_ context.Context, _ *domain.Tenant) error { return nil }
func (f *fakeTenantRepo) Delete(_ context.Context, _ uuid.UUID) error      { return nil }

func newTestTrustProviderService() (*TrustProviderService, *fakeTrustProviderRepo) {
	repo := newFakeTrustProviderRepo()
	tenants := &fakeTenantRepo{tenant: &domain.Tenant{ID: uuid.New(), Subdomain: "acme"}}
	hosts := platform.NewHosts("dipolehq.dev", "production")
	svc := NewTrustProviderService(repo, tenants, "us-east-1", hosts, testEncryptionKey)
	return svc, repo
}

func TestTrustProviderService_Create_ExtractsAndEncryptsSymmetricKey(t *testing.T) {
	svc, repo := newTestTrustProviderService()

	tp := &domain.TrustProvider{
		Name:   "Test OIDC",
		Type:   "oidc_id_token",
		Config: json.RawMessage(`{"attestationMethod":"symmetric_key","symmetricKey":"c3VwZXItc2VjcmV0"}`),
	}

	if err := svc.Create(context.Background(), uuid.New(), tp); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stored := repo.stored[tp.ID]
	if stored == nil {
		t.Fatal("trust provider was not persisted")
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal(stored.Config, &cfg); err != nil {
		t.Fatalf("stored config is not valid JSON: %v", err)
	}
	if _, ok := cfg["symmetricKey"]; ok {
		t.Error("expected symmetricKey to be stripped out of stored Config")
	}
	if cfg["attestationMethod"] != "symmetric_key" {
		t.Errorf("expected non-secret fields to be preserved, got %v", cfg)
	}

	if len(stored.SymmetricKeyEncrypted) == 0 {
		t.Fatal("expected SymmetricKeyEncrypted to be populated")
	}
	decrypted, err := crypto.Decrypt(stored.SymmetricKeyEncrypted, testEncryptionKey)
	if err != nil {
		t.Fatalf("failed to decrypt stored symmetric key: %v", err)
	}
	if string(decrypted) != "c3VwZXItc2VjcmV0" {
		t.Errorf("decrypted symmetric key = %q, want %q", decrypted, "c3VwZXItc2VjcmV0")
	}
}

func TestTrustProviderService_Create_NoSymmetricKey_NoEncryptedValue(t *testing.T) {
	svc, repo := newTestTrustProviderService()

	tp := &domain.TrustProvider{
		Name:   "Test Discovery",
		Type:   "oidc_id_token",
		Config: json.RawMessage(`{"attestationMethod":"oidc_discovery","oidcEndpoint":"https://accounts.google.com"}`),
	}

	if err := svc.Create(context.Background(), uuid.New(), tp); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stored := repo.stored[tp.ID]
	if len(stored.SymmetricKeyEncrypted) != 0 {
		t.Error("expected no SymmetricKeyEncrypted for a provider with no symmetricKey in config")
	}
}

func TestTrustProviderService_Update_OmittingSymmetricKey_PreservesExisting(t *testing.T) {
	svc, repo := newTestTrustProviderService()
	tenantID := uuid.New()

	tp := &domain.TrustProvider{
		Name:   "Test OIDC",
		Type:   "oidc_id_token",
		Config: json.RawMessage(`{"attestationMethod":"symmetric_key","symmetricKey":"b3JpZ2luYWwtc2VjcmV0"}`),
	}
	if err := svc.Create(context.Background(), tenantID, tp); err != nil {
		t.Fatalf("create failed: %v", err)
	}
	originalEncrypted := repo.stored[tp.ID].SymmetricKeyEncrypted

	update := &domain.TrustProvider{
		ID:     tp.ID,
		Name:   "Test OIDC Renamed",
		Type:   "oidc_id_token",
		Config: json.RawMessage(`{"attestationMethod":"symmetric_key"}`), // no symmetricKey resent
	}
	if err := svc.Update(context.Background(), tenantID, update); err != nil {
		t.Fatalf("update failed: %v", err)
	}

	stored := repo.stored[tp.ID]
	if stored.Name != "Test OIDC Renamed" {
		t.Errorf("expected name to be updated, got %q", stored.Name)
	}
	decrypted, err := crypto.Decrypt(stored.SymmetricKeyEncrypted, testEncryptionKey)
	if err != nil {
		t.Fatalf("failed to decrypt preserved symmetric key: %v", err)
	}
	if string(decrypted) != "b3JpZ2luYWwtc2VjcmV0" {
		t.Errorf("expected the original secret to be preserved when omitted from update, got %q", decrypted)
	}
	_ = originalEncrypted
}

func TestTrustProviderService_Update_ProvidingNewSymmetricKey_ReplacesExisting(t *testing.T) {
	svc, repo := newTestTrustProviderService()
	tenantID := uuid.New()

	tp := &domain.TrustProvider{
		Name:   "Test OIDC",
		Type:   "oidc_id_token",
		Config: json.RawMessage(`{"attestationMethod":"symmetric_key","symmetricKey":"b2xkLXNlY3JldA=="}`),
	}
	if err := svc.Create(context.Background(), tenantID, tp); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	update := &domain.TrustProvider{
		ID:     tp.ID,
		Name:   "Test OIDC",
		Type:   "oidc_id_token",
		Config: json.RawMessage(`{"attestationMethod":"symmetric_key","symmetricKey":"bmV3LXNlY3JldA=="}`),
	}
	if err := svc.Update(context.Background(), tenantID, update); err != nil {
		t.Fatalf("update failed: %v", err)
	}

	decrypted, err := crypto.Decrypt(repo.stored[tp.ID].SymmetricKeyEncrypted, testEncryptionKey)
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}
	if string(decrypted) != "bmV3LXNlY3JldA==" {
		t.Errorf("expected new secret to replace old, got %q", decrypted)
	}
}

func TestTrustProviderService_Create_InvalidConfigJSON_ReturnsValidationError(t *testing.T) {
	svc, _ := newTestTrustProviderService()

	tp := &domain.TrustProvider{
		Name:   "Bad Config",
		Type:   "oidc_id_token",
		Config: json.RawMessage(`not-json`),
	}

	err := svc.Create(context.Background(), uuid.New(), tp)
	if err == nil {
		t.Fatal("expected an error for invalid config JSON")
	}
	var validationErr *domain.ValidationError
	if !isValidationError(err, &validationErr) {
		t.Errorf("expected a domain.ValidationError, got %T: %v", err, err)
	}
}

func isValidationError(err error, target **domain.ValidationError) bool {
	ve, ok := err.(*domain.ValidationError)
	if ok {
		*target = ve
	}
	return ok
}

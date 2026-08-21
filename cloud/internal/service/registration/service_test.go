// service_test.go tests RegistrationService.VerifyControllerSignature, the HMAC check that authenticates Agent Controller requests to /internal/controllers/token and /internal/controllers/health.
package registration

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/crypto"
	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type fakeControllerRepoForSigTest struct {
	controllers map[uuid.UUID]*domain.AgentController
}

func newFakeControllerRepoForSigTest() *fakeControllerRepoForSigTest {
	return &fakeControllerRepoForSigTest{controllers: make(map[uuid.UUID]*domain.AgentController)}
}

func (f *fakeControllerRepoForSigTest) Create(_ context.Context, _ uuid.UUID, ac *domain.AgentController) error {
	f.controllers[ac.ID] = ac
	return nil
}
func (f *fakeControllerRepoForSigTest) GetByID(_ context.Context, _ uuid.UUID, id uuid.UUID) (*domain.AgentController, error) {
	return f.controllers[id], nil
}
func (f *fakeControllerRepoForSigTest) List(_ context.Context, _ uuid.UUID, _ domain.ListOptions) ([]domain.AgentController, int, error) {
	return nil, 0, nil
}
func (f *fakeControllerRepoForSigTest) Update(_ context.Context, _ uuid.UUID, ac *domain.AgentController) error {
	f.controllers[ac.ID] = ac
	return nil
}
func (f *fakeControllerRepoForSigTest) Delete(_ context.Context, _ uuid.UUID, id uuid.UUID) error {
	delete(f.controllers, id)
	return nil
}
func (f *fakeControllerRepoForSigTest) ConsumeDeviceCode(_ context.Context, _ uuid.UUID, _ string, _ []byte) (*domain.AgentController, error) {
	return nil, nil
}

func testEncryptionKey() []byte {
	return []byte("01234567890123456789012345678901")
}

func newControllerWithSecret(t *testing.T, repo *fakeControllerRepoForSigTest, tenantID uuid.UUID, secret []byte) uuid.UUID {
	t.Helper()
	encKey := testEncryptionKey()
	secretEncrypted, err := crypto.Encrypt(secret, encKey)
	if err != nil {
		t.Fatalf("encrypt test secret: %v", err)
	}
	id := uuid.New()
	repo.controllers[id] = &domain.AgentController{
		ID:                    id,
		TenantID:              tenantID,
		Status:                domain.AgentStatusHealthy,
		SharedSecretEncrypted: secretEncrypted,
	}
	return id
}

func sign(secret, body []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func TestVerifyControllerSignature_ValidSignature_Succeeds(t *testing.T) {
	repo := newFakeControllerRepoForSigTest()
	tenantID := uuid.New()
	secret := []byte("controller-shared-secret")
	controllerID := newControllerWithSecret(t, repo, tenantID, secret)

	svc := NewRegistrationService(repo, nil, nil, testEncryptionKey())

	body := []byte(`{"tenant_id":"x","controller_id":"y"}`)
	sig := sign(secret, body)

	if err := svc.VerifyControllerSignature(context.Background(), tenantID, controllerID, body, sig); err != nil {
		t.Fatalf("expected valid signature to succeed, got %v", err)
	}
}

func TestVerifyControllerSignature_WrongSecret_Rejected(t *testing.T) {
	repo := newFakeControllerRepoForSigTest()
	tenantID := uuid.New()
	controllerID := newControllerWithSecret(t, repo, tenantID, []byte("real-secret"))

	svc := NewRegistrationService(repo, nil, nil, testEncryptionKey())

	body := []byte(`{"tenant_id":"x","controller_id":"y"}`)
	sig := sign([]byte("attacker-guessed-secret"), body)

	err := svc.VerifyControllerSignature(context.Background(), tenantID, controllerID, body, sig)
	var regErr *domain.RegistrationError
	if !errors.As(err, &regErr) {
		t.Fatalf("expected a RegistrationError for a wrong-secret signature, got %v", err)
	}
}

func TestVerifyControllerSignature_TamperedBody_Rejected(t *testing.T) {
	repo := newFakeControllerRepoForSigTest()
	tenantID := uuid.New()
	secret := []byte("controller-shared-secret")
	controllerID := newControllerWithSecret(t, repo, tenantID, secret)

	svc := NewRegistrationService(repo, nil, nil, testEncryptionKey())

	originalBody := []byte(`{"tenant_id":"x","controller_id":"y"}`)
	sig := sign(secret, originalBody)

	tamperedBody := []byte(`{"tenant_id":"x","controller_id":"z"}`)

	err := svc.VerifyControllerSignature(context.Background(), tenantID, controllerID, tamperedBody, sig)
	var regErr *domain.RegistrationError
	if !errors.As(err, &regErr) {
		t.Fatalf("expected a signature computed over a different body to be rejected, got %v", err)
	}
}

func TestVerifyControllerSignature_MissingSignature_Rejected(t *testing.T) {
	repo := newFakeControllerRepoForSigTest()
	tenantID := uuid.New()
	controllerID := newControllerWithSecret(t, repo, tenantID, []byte("real-secret"))

	svc := NewRegistrationService(repo, nil, nil, testEncryptionKey())

	err := svc.VerifyControllerSignature(context.Background(), tenantID, controllerID, []byte(`{}`), "")
	var regErr *domain.RegistrationError
	if !errors.As(err, &regErr) {
		t.Fatalf("expected an empty signature to be rejected, got %v", err)
	}
}

func TestVerifyControllerSignature_UnknownController_Rejected(t *testing.T) {
	repo := newFakeControllerRepoForSigTest()
	svc := NewRegistrationService(repo, nil, nil, testEncryptionKey())

	err := svc.VerifyControllerSignature(context.Background(), uuid.New(), uuid.New(), []byte(`{}`), "aabbcc")
	var regErr *domain.RegistrationError
	if !errors.As(err, &regErr) {
		t.Fatalf("expected an unknown controller to be rejected, got %v", err)
	}
}

func TestVerifyControllerSignature_NoSecretConfigured_Rejected(t *testing.T) {
	repo := newFakeControllerRepoForSigTest()
	tenantID := uuid.New()
	controllerID := uuid.New()
	repo.controllers[controllerID] = &domain.AgentController{
		ID:       controllerID,
		TenantID: tenantID,
		Status:   domain.AgentStatusHealthy,
	}

	svc := NewRegistrationService(repo, nil, nil, testEncryptionKey())

	err := svc.VerifyControllerSignature(context.Background(), tenantID, controllerID, []byte(`{}`), "aabbcc")
	var regErr *domain.RegistrationError
	if !errors.As(err, &regErr) {
		t.Fatalf("expected a controller with no shared secret to be rejected, got %v", err)
	}
}

func TestVerifyControllerSignature_InvalidHexSignature_Rejected(t *testing.T) {
	repo := newFakeControllerRepoForSigTest()
	tenantID := uuid.New()
	controllerID := newControllerWithSecret(t, repo, tenantID, []byte("real-secret"))

	svc := NewRegistrationService(repo, nil, nil, testEncryptionKey())

	err := svc.VerifyControllerSignature(context.Background(), tenantID, controllerID, []byte(`{}`), "not-valid-hex!!")
	var regErr *domain.RegistrationError
	if !errors.As(err, &regErr) {
		t.Fatalf("expected a non-hex signature to be rejected, got %v", err)
	}
}

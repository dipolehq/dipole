// internal_handlers_test.go tests that /internal/controllers/token and /internal/controllers/health reject requests with a missing or invalid controller signature before ever calling into RegistrationService's business logic.
package rest

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/crypto"
	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/service/registration"
)

type fakeControllerRepoForHandlerTest struct {
	controllers map[uuid.UUID]*domain.AgentController
}

func (f *fakeControllerRepoForHandlerTest) Create(_ context.Context, _ uuid.UUID, ac *domain.AgentController) error {
	f.controllers[ac.ID] = ac
	return nil
}
func (f *fakeControllerRepoForHandlerTest) GetByID(_ context.Context, _ uuid.UUID, id uuid.UUID) (*domain.AgentController, error) {
	return f.controllers[id], nil
}
func (f *fakeControllerRepoForHandlerTest) List(_ context.Context, _ uuid.UUID, _ domain.ListOptions) ([]domain.AgentController, int, error) {
	return nil, 0, nil
}
func (f *fakeControllerRepoForHandlerTest) Update(_ context.Context, _ uuid.UUID, ac *domain.AgentController) error {
	f.controllers[ac.ID] = ac
	return nil
}
func (f *fakeControllerRepoForHandlerTest) Delete(_ context.Context, _ uuid.UUID, id uuid.UUID) error {
	delete(f.controllers, id)
	return nil
}
func (f *fakeControllerRepoForHandlerTest) ConsumeDeviceCode(_ context.Context, _ uuid.UUID, _ string, _ []byte) (*domain.AgentController, error) {
	return nil, nil
}

type fakeProxyRepoForHandlerTest struct{}

func (f *fakeProxyRepoForHandlerTest) Create(_ context.Context, _ uuid.UUID, _ *domain.AgentProxy) error {
	return nil
}
func (f *fakeProxyRepoForHandlerTest) GetByID(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*domain.AgentProxy, error) {
	return nil, nil
}
func (f *fakeProxyRepoForHandlerTest) List(_ context.Context, _ uuid.UUID, _ domain.ListOptions) ([]domain.AgentProxy, int, error) {
	return nil, 0, nil
}
func (f *fakeProxyRepoForHandlerTest) UpdateHealth(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ string) error {
	return nil
}
func (f *fakeProxyRepoForHandlerTest) ListByController(_ context.Context, _ uuid.UUID, _ uuid.UUID) ([]domain.AgentProxy, error) {
	return nil, nil
}
func (f *fakeProxyRepoForHandlerTest) SetSessionTokenHash(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ string, _ time.Time) error {
	return nil
}
func (f *fakeProxyRepoForHandlerTest) GetBySessionTokenHash(_ context.Context, _ string) (*domain.AgentProxy, error) {
	return nil, nil
}

type fakeRegTokenRepoForHandlerTest struct{}

func (f *fakeRegTokenRepoForHandlerTest) Store(_ context.Context, _ uuid.UUID, _ string, _ uuid.UUID, _ time.Time) error {
	return nil
}
func (f *fakeRegTokenRepoForHandlerTest) Consume(_ context.Context, _ uuid.UUID, _ string) (*domain.RegistrationToken, error) {
	return nil, nil
}

func newSignedControllerTestSetup(t *testing.T) (*InternalHandlers, uuid.UUID, uuid.UUID, []byte) {
	t.Helper()
	encKey := []byte("01234567890123456789012345678901")
	secret := []byte("test-controller-secret")
	secretEncrypted, err := crypto.Encrypt(secret, encKey)
	if err != nil {
		t.Fatalf("encrypt test secret: %v", err)
	}

	tenantID := uuid.New()
	controllerID := uuid.New()

	controllerRepo := &fakeControllerRepoForHandlerTest{controllers: map[uuid.UUID]*domain.AgentController{
		controllerID: {
			ID:                    controllerID,
			TenantID:              tenantID,
			Status:                domain.AgentStatusHealthy,
			SharedSecretEncrypted: secretEncrypted,
		},
	}}

	regSvc := registration.NewRegistrationService(controllerRepo, &fakeProxyRepoForHandlerTest{}, &fakeRegTokenRepoForHandlerTest{}, encKey)
	return NewInternalHandlers(regSvc), tenantID, controllerID, secret
}

func TestHandleControllerToken_MissingSignature_Rejected(t *testing.T) {
	h, tenantID, controllerID, _ := newSignedControllerTestSetup(t)

	body := []byte(`{"tenant_id":"` + tenantID.String() + `","controller_id":"` + controllerID.String() + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/internal/controllers/token", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.handleControllerToken(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for a missing signature, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleControllerToken_WrongSignature_Rejected(t *testing.T) {
	h, tenantID, controllerID, _ := newSignedControllerTestSetup(t)

	body := []byte(`{"tenant_id":"` + tenantID.String() + `","controller_id":"` + controllerID.String() + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/internal/controllers/token", bytes.NewReader(body))
	req.Header.Set(controllerSignatureHeader, sign([]byte("wrong-secret"), body))
	w := httptest.NewRecorder()

	h.handleControllerToken(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for a wrong signature, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleControllerToken_ValidSignature_Succeeds(t *testing.T) {
	h, tenantID, controllerID, secret := newSignedControllerTestSetup(t)

	body := []byte(`{"tenant_id":"` + tenantID.String() + `","controller_id":"` + controllerID.String() + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/internal/controllers/token", bytes.NewReader(body))
	req.Header.Set(controllerSignatureHeader, sign(secret, body))
	w := httptest.NewRecorder()

	h.handleControllerToken(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for a valid signature, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleControllerHealth_MissingSignature_Rejected(t *testing.T) {
	h, tenantID, controllerID, _ := newSignedControllerTestSetup(t)

	body := []byte(`{"tenant_id":"` + tenantID.String() + `","controller_id":"` + controllerID.String() + `","status":"Healthy"}`)
	req := httptest.NewRequest(http.MethodPost, "/internal/controllers/health", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.handleControllerHealth(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for a missing signature, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleControllerHealth_ValidSignature_Succeeds(t *testing.T) {
	h, tenantID, controllerID, secret := newSignedControllerTestSetup(t)

	body := []byte(`{"tenant_id":"` + tenantID.String() + `","controller_id":"` + controllerID.String() + `","status":"Healthy"}`)
	req := httptest.NewRequest(http.MethodPost, "/internal/controllers/health", bytes.NewReader(body))
	req.Header.Set(controllerSignatureHeader, sign(secret, body))
	w := httptest.NewRecorder()

	h.handleControllerHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for a valid signature, got %d: %s", w.Code, w.Body.String())
	}
}

func sign(secret, body []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

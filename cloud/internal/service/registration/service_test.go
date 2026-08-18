/*
service_test.go tests the RegistrationService with focus on single-use token behavior.
*/
package registration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type mockControllerRepo struct {
	controller *domain.AgentController
	err        error
}

func (m *mockControllerRepo) Create(_ context.Context, _ uuid.UUID, _ *domain.AgentController) error {
	return nil
}
func (m *mockControllerRepo) GetByID(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*domain.AgentController, error) {
	return m.controller, m.err
}
func (m *mockControllerRepo) List(_ context.Context, _ uuid.UUID, _ domain.ListOptions) ([]domain.AgentController, int, error) {
	return nil, 0, nil
}
func (m *mockControllerRepo) Update(_ context.Context, _ uuid.UUID, _ *domain.AgentController) error {
	return nil
}
func (m *mockControllerRepo) Delete(_ context.Context, _ uuid.UUID, _ uuid.UUID) error { return nil }
func (m *mockControllerRepo) GetByDeviceCode(_ context.Context, _ uuid.UUID, _ string) (*domain.AgentController, error) {
	return m.controller, m.err
}

type mockProxyRepo struct {
	created         *domain.AgentProxy
	err             error
	sessionTokenErr error
	setTokenHashes  map[uuid.UUID]string
	setTokenExpiry  time.Time
}

func (m *mockProxyRepo) Create(_ context.Context, _ uuid.UUID, proxy *domain.AgentProxy) error {
	m.created = proxy
	return m.err
}
func (m *mockProxyRepo) GetByID(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*domain.AgentProxy, error) {
	return nil, nil
}
func (m *mockProxyRepo) UpdateHealth(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ string) error {
	return nil
}
func (m *mockProxyRepo) ListByController(_ context.Context, _ uuid.UUID, _ uuid.UUID) ([]domain.AgentProxy, error) {
	return nil, nil
}
func (m *mockProxyRepo) List(_ context.Context, _ uuid.UUID, _ domain.ListOptions) ([]domain.AgentProxy, int, error) {
	return nil, 0, nil
}
func (m *mockProxyRepo) SetSessionTokenHash(_ context.Context, _ uuid.UUID, proxyID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	if m.sessionTokenErr != nil {
		return m.sessionTokenErr
	}
	if m.setTokenHashes == nil {
		m.setTokenHashes = make(map[uuid.UUID]string)
	}
	m.setTokenHashes[proxyID] = tokenHash
	m.setTokenExpiry = expiresAt
	return nil
}
func (m *mockProxyRepo) GetBySessionTokenHash(_ context.Context, _ string) (*domain.AgentProxy, error) {
	return nil, nil
}

type mockTokenRepo struct {
	stored   []storedToken
	consumed *domain.RegistrationToken
	err      error
}

type storedToken struct {
	tenantID     uuid.UUID
	tokenHash    string
	controllerID uuid.UUID
	expiresAt    time.Time
}

func (m *mockTokenRepo) Store(_ context.Context, tenantID uuid.UUID, tokenHash string, controllerID uuid.UUID, expiresAt time.Time) error {
	m.stored = append(m.stored, storedToken{tenantID, tokenHash, controllerID, expiresAt})
	return nil
}

func (m *mockTokenRepo) Consume(_ context.Context, _ uuid.UUID, _ string) (*domain.RegistrationToken, error) {
	return m.consumed, m.err
}

func TestIssueProxyToken_StoresHash(t *testing.T) {
	tenantID := uuid.New()
	controllerID := uuid.New()

	ctrlRepo := &mockControllerRepo{
		controller: &domain.AgentController{
			ID:       controllerID,
			TenantID: tenantID,
			Status:   domain.AgentStatusHealthy,
		},
	}
	tokenRepo := &mockTokenRepo{}
	svc := NewRegistrationService(ctrlRepo, &mockProxyRepo{}, tokenRepo)

	token, err := svc.IssueProxyToken(context.Background(), tenantID, controllerID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	if len(tokenRepo.stored) != 1 {
		t.Fatalf("expected 1 stored hash, got %d", len(tokenRepo.stored))
	}

	hash := sha256.Sum256([]byte(token))
	expectedHash := hex.EncodeToString(hash[:])
	if tokenRepo.stored[0].tokenHash != expectedHash {
		t.Errorf("stored hash mismatch: got %s, want %s", tokenRepo.stored[0].tokenHash, expectedHash)
	}
	if tokenRepo.stored[0].tenantID != tenantID {
		t.Error("stored tenantID mismatch")
	}
	if tokenRepo.stored[0].controllerID != controllerID {
		t.Error("stored controllerID mismatch")
	}
	if time.Until(tokenRepo.stored[0].expiresAt) < 59*time.Minute {
		t.Error("expected expiry ~1 hour from now")
	}
}

func TestRegisterProxy_ConsumesToken(t *testing.T) {
	tenantID := uuid.New()
	controllerID := uuid.New()

	ctrlRepo := &mockControllerRepo{
		controller: &domain.AgentController{
			ID:       controllerID,
			TenantID: tenantID,
			Status:   domain.AgentStatusHealthy,
		},
	}
	proxyRepo := &mockProxyRepo{}
	tokenRepo := &mockTokenRepo{
		consumed: &domain.RegistrationToken{
			ID:           uuid.New(),
			TenantID:     tenantID,
			ControllerID: controllerID,
			Consumed:     true,
			ExpiresAt:    time.Now().Add(1 * time.Hour),
		},
	}
	svc := NewRegistrationService(ctrlRepo, proxyRepo, tokenRepo)

	token := tenantID.String() + ":" + controllerID.String() + ":abcdef1234567890"
	proxy, sessionToken, expiresAt, err := svc.RegisterProxy(context.Background(), tenantID, token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if proxy == nil {
		t.Fatal("expected proxy to be created")
	}
	if proxyRepo.created == nil {
		t.Fatal("expected proxy to be persisted")
	}
	if sessionToken == "" {
		t.Fatal("expected a non-empty session token")
	}
	if time.Until(expiresAt) < 59*time.Minute {
		t.Error("expected session token expiry ~1 hour from now")
	}
}

func TestRegisterProxy_IssuesAndStoresSessionTokenHash(t *testing.T) {
	tenantID := uuid.New()
	controllerID := uuid.New()

	ctrlRepo := &mockControllerRepo{
		controller: &domain.AgentController{
			ID:       controllerID,
			TenantID: tenantID,
			Status:   domain.AgentStatusHealthy,
		},
	}
	proxyRepo := &mockProxyRepo{}
	tokenRepo := &mockTokenRepo{
		consumed: &domain.RegistrationToken{
			ID:           uuid.New(),
			TenantID:     tenantID,
			ControllerID: controllerID,
			Consumed:     true,
			ExpiresAt:    time.Now().Add(1 * time.Hour),
		},
	}
	svc := NewRegistrationService(ctrlRepo, proxyRepo, tokenRepo)

	token := tenantID.String() + ":" + controllerID.String() + ":abcdef1234567890"
	proxy, sessionToken, expiresAt, err := svc.RegisterProxy(context.Background(), tenantID, token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	storedHash, ok := proxyRepo.setTokenHashes[proxy.ID]
	if !ok {
		t.Fatal("expected a session token hash to be stored for the new proxy")
	}

	expectedHash := hashSessionToken(sessionToken)
	if storedHash != expectedHash {
		t.Errorf("stored hash = %q, want %q (sha256 of returned token)", storedHash, expectedHash)
	}

	// The raw token must never be persisted anywhere — only its hash.
	if storedHash == sessionToken {
		t.Fatal("raw session token was stored instead of its hash")
	}

	if !proxyRepo.setTokenExpiry.Equal(expiresAt) {
		t.Errorf("stored expiry = %v, want %v (matching the returned expiresAt)", proxyRepo.setTokenExpiry, expiresAt)
	}
}

func TestRegisterProxy_SessionTokenStoreFailure_PropagatesError(t *testing.T) {
	tenantID := uuid.New()
	controllerID := uuid.New()

	ctrlRepo := &mockControllerRepo{
		controller: &domain.AgentController{
			ID:       controllerID,
			TenantID: tenantID,
			Status:   domain.AgentStatusHealthy,
		},
	}
	proxyRepo := &mockProxyRepo{sessionTokenErr: fmt.Errorf("db unavailable")}
	tokenRepo := &mockTokenRepo{
		consumed: &domain.RegistrationToken{
			ID:           uuid.New(),
			TenantID:     tenantID,
			ControllerID: controllerID,
			Consumed:     true,
			ExpiresAt:    time.Now().Add(1 * time.Hour),
		},
	}
	svc := NewRegistrationService(ctrlRepo, proxyRepo, tokenRepo)

	token := tenantID.String() + ":" + controllerID.String() + ":abcdef1234567890"
	_, sessionToken, expiresAt, err := svc.RegisterProxy(context.Background(), tenantID, token)
	if err == nil {
		t.Fatal("expected error when storing the session token hash fails")
	}
	if sessionToken != "" {
		t.Error("expected empty session token on failure")
	}
	if !expiresAt.IsZero() {
		t.Error("expected zero-value expiry on failure")
	}
}

func TestRegisterProxy_RejectsReplayedToken(t *testing.T) {
	tenantID := uuid.New()
	controllerID := uuid.New()

	ctrlRepo := &mockControllerRepo{
		controller: &domain.AgentController{
			ID:       controllerID,
			TenantID: tenantID,
			Status:   domain.AgentStatusHealthy,
		},
	}
	tokenRepo := &mockTokenRepo{
		consumed: nil,
		err:      fmt.Errorf("no rows"),
	}
	svc := NewRegistrationService(ctrlRepo, &mockProxyRepo{}, tokenRepo)

	token := tenantID.String() + ":" + controllerID.String() + ":abcdef1234567890"
	_, _, _, err := svc.RegisterProxy(context.Background(), tenantID, token)
	if err == nil {
		t.Fatal("expected error for replayed token")
	}

	var regErr *domain.RegistrationError
	if !errors.As(err, &regErr) {
		t.Fatalf("expected RegistrationError, got %T: %v", err, err)
	}
	if regErr.Message != "invalid or already consumed registration token" {
		t.Errorf("unexpected message: %s", regErr.Message)
	}
}

func TestRegisterProxy_RejectsExpiredToken(t *testing.T) {
	tenantID := uuid.New()
	controllerID := uuid.New()

	ctrlRepo := &mockControllerRepo{
		controller: &domain.AgentController{
			ID:       controllerID,
			TenantID: tenantID,
			Status:   domain.AgentStatusHealthy,
		},
	}
	tokenRepo := &mockTokenRepo{
		consumed: nil,
		err:      fmt.Errorf("no rows"),
	}
	svc := NewRegistrationService(ctrlRepo, &mockProxyRepo{}, tokenRepo)

	token := tenantID.String() + ":" + controllerID.String() + ":expired123"
	_, _, _, err := svc.RegisterProxy(context.Background(), tenantID, token)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestRefreshSessionToken_IssuesNewHashAndExpiry(t *testing.T) {
	tenantID := uuid.New()
	proxyID := uuid.New()

	proxyRepo := &mockProxyRepo{}
	svc := NewRegistrationService(&mockControllerRepo{}, proxyRepo, &mockTokenRepo{})

	sessionToken, expiresAt, err := svc.RefreshSessionToken(context.Background(), tenantID, proxyID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sessionToken == "" {
		t.Fatal("expected a non-empty session token")
	}
	if time.Until(expiresAt) < 59*time.Minute {
		t.Error("expected session token expiry ~1 hour from now")
	}

	storedHash, ok := proxyRepo.setTokenHashes[proxyID]
	if !ok {
		t.Fatal("expected a session token hash to be stored for the proxy")
	}
	if storedHash != hashSessionToken(sessionToken) {
		t.Error("stored hash does not match sha256 of the returned token")
	}
	if storedHash == sessionToken {
		t.Fatal("raw session token was stored instead of its hash")
	}
}

func TestRefreshSessionToken_StoreFailure_PropagatesError(t *testing.T) {
	proxyRepo := &mockProxyRepo{sessionTokenErr: fmt.Errorf("db unavailable")}
	svc := NewRegistrationService(&mockControllerRepo{}, proxyRepo, &mockTokenRepo{})

	sessionToken, expiresAt, err := svc.RefreshSessionToken(context.Background(), uuid.New(), uuid.New())
	if err == nil {
		t.Fatal("expected error when storing the refreshed session token hash fails")
	}
	if sessionToken != "" {
		t.Error("expected empty session token on failure")
	}
	if !expiresAt.IsZero() {
		t.Error("expected zero-value expiry on failure")
	}
}

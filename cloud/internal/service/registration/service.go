// Package registration handles Agent Controller and Proxy registration with Cloud: device codes, one-time proxy tokens, session token issuance/refresh, and health tracking.
package registration

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/crypto"
	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
)

const sessionTokenLifetime = 1 * time.Hour

type RegistrationService struct {
	controllerRepo repository.AgentControllerRepository
	proxyRepo      repository.AgentProxyRepository
	tokenRepo      repository.RegistrationTokenRepository
	encryptionKey  []byte
}

func NewRegistrationService(controllerRepo repository.AgentControllerRepository, proxyRepo repository.AgentProxyRepository, tokenRepo repository.RegistrationTokenRepository, encryptionKey []byte) *RegistrationService {
	return &RegistrationService{controllerRepo: controllerRepo, proxyRepo: proxyRepo, tokenRepo: tokenRepo, encryptionKey: encryptionKey}
}

func (s *RegistrationService) RegisterController(ctx context.Context, tenantID uuid.UUID, deviceCode string) (controller *domain.AgentController, sharedSecret string, err error) {
	sharedSecret, err = generateSharedSecret()
	if err != nil {
		return nil, "", fmt.Errorf("generate shared secret: %w", err)
	}

	secretEncrypted, err := crypto.Encrypt([]byte(sharedSecret), s.encryptionKey)
	if err != nil {
		return nil, "", fmt.Errorf("encrypt shared secret: %w", err)
	}

	controller, err = s.controllerRepo.ConsumeDeviceCode(ctx, tenantID, deviceCode, secretEncrypted)
	if err != nil {
		return nil, "", fmt.Errorf("consume device code: %w", err)
	}
	if controller == nil {
		return nil, "", domain.NewRegistrationError("invalid or expired device code")
	}

	return controller, sharedSecret, nil
}

func (s *RegistrationService) CreateController(ctx context.Context, tenantID uuid.UUID, name string) (*domain.AgentController, error) {
	code := generateDeviceCode()
	expiresAt := time.Now().Add(1 * time.Hour)

	controller := &domain.AgentController{
		ID:                  uuid.New(),
		TenantID:            tenantID,
		Name:                name,
		IsActive:            true,
		Status:              domain.AgentStatusUnregistered,
		DeviceCode:          code,
		DeviceCodeExpiresAt: &expiresAt,
		CreatedAt:           time.Now(),
	}

	if err := s.controllerRepo.Create(ctx, tenantID, controller); err != nil {
		return nil, fmt.Errorf("create controller: %w", err)
	}

	return controller, nil
}

func (s *RegistrationService) RegenerateDeviceCode(ctx context.Context, tenantID, controllerID uuid.UUID) (string, error) {
	controller, err := s.controllerRepo.GetByID(ctx, tenantID, controllerID)
	if err != nil {
		return "", fmt.Errorf("lookup controller: %w", err)
	}
	if controller == nil {
		return "", domain.NewRegistrationError("controller not found")
	}

	code := generateDeviceCode()
	expiresAt := time.Now().Add(1 * time.Hour)
	controller.DeviceCode = code
	controller.DeviceCodeExpiresAt = &expiresAt
	controller.Status = domain.AgentStatusUnregistered

	if err := s.controllerRepo.Update(ctx, tenantID, controller); err != nil {
		return "", fmt.Errorf("update controller: %w", err)
	}

	return code, nil
}

func (s *RegistrationService) IssueProxyToken(ctx context.Context, tenantID, controllerID uuid.UUID) (string, error) {
	controller, err := s.controllerRepo.GetByID(ctx, tenantID, controllerID)
	if err != nil {
		return "", fmt.Errorf("lookup controller: %w", err)
	}
	if controller == nil {
		return "", domain.NewRegistrationError("controller not found")
	}

	if controller.Status != domain.AgentStatusRegistered && controller.Status != domain.AgentStatusHealthy {
		return "", domain.NewRegistrationError("controller not registered")
	}

	rawBytes := make([]byte, 32)
	if _, err := rand.Read(rawBytes); err != nil {
		return "", fmt.Errorf("generate random token: %w", err)
	}
	token := fmt.Sprintf("%s:%s:%x", tenantID, controllerID, rawBytes)

	hash := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(hash[:])
	expiresAt := time.Now().Add(1 * time.Hour)

	if err := s.tokenRepo.Store(ctx, tenantID, tokenHash, controllerID, expiresAt); err != nil {
		return "", fmt.Errorf("store token hash: %w", err)
	}

	return token, nil
}

func (s *RegistrationService) RegisterProxy(ctx context.Context, tenantID uuid.UUID, registrationToken string) (*domain.AgentProxy, string, time.Time, error) {
	if registrationToken == "" {
		return nil, "", time.Time{}, domain.NewRegistrationError("empty registration token")
	}

	parts := strings.SplitN(registrationToken, ":", 3)
	if len(parts) != 3 {
		return nil, "", time.Time{}, domain.NewRegistrationError("invalid registration token format")
	}

	tokenTenantID, err := uuid.Parse(parts[0])
	if err != nil {
		return nil, "", time.Time{}, domain.NewRegistrationError("invalid tenant_id in token")
	}
	if tokenTenantID != tenantID {
		return nil, "", time.Time{}, domain.NewRegistrationError("token tenant mismatch")
	}

	controllerID, err := uuid.Parse(parts[1])
	if err != nil {
		return nil, "", time.Time{}, domain.NewRegistrationError("invalid controller_id in token")
	}

	hash := sha256.Sum256([]byte(registrationToken))
	tokenHash := hex.EncodeToString(hash[:])

	consumed, err := s.tokenRepo.Consume(ctx, tenantID, tokenHash)
	if err != nil || consumed == nil {
		return nil, "", time.Time{}, domain.NewRegistrationError("invalid or already consumed registration token")
	}

	controller, err := s.controllerRepo.GetByID(ctx, tenantID, controllerID)
	if err != nil {
		return nil, "", time.Time{}, fmt.Errorf("lookup controller: %w", err)
	}
	if controller == nil {
		return nil, "", time.Time{}, domain.NewRegistrationError("controller not found")
	}
	if controller.Status != domain.AgentStatusRegistered && controller.Status != domain.AgentStatusHealthy {
		return nil, "", time.Time{}, domain.NewRegistrationError("controller not in valid state for proxy registration")
	}

	now := time.Now()
	proxy := &domain.AgentProxy{
		ID:                uuid.New(),
		TenantID:          tenantID,
		AgentControllerID: &controllerID,
		Status:            domain.AgentStatusRegistered,
		RegisteredAt:      &now,
	}

	if err := s.proxyRepo.Create(ctx, tenantID, proxy); err != nil {
		return nil, "", time.Time{}, fmt.Errorf("persist proxy: %w", err)
	}

	sessionToken, expiresAt, err := s.issueSessionToken(ctx, tenantID, proxy.ID)
	if err != nil {
		return nil, "", time.Time{}, err
	}

	return proxy, sessionToken, expiresAt, nil
}

func (s *RegistrationService) RefreshSessionToken(ctx context.Context, tenantID, proxyID uuid.UUID) (string, time.Time, error) {
	return s.issueSessionToken(ctx, tenantID, proxyID)
}

func (s *RegistrationService) issueSessionToken(ctx context.Context, tenantID, proxyID uuid.UUID) (string, time.Time, error) {
	sessionToken, err := generateSessionToken()
	if err != nil {
		return "", time.Time{}, fmt.Errorf("generate session token: %w", err)
	}
	sessionTokenHash := hashSessionToken(sessionToken)
	expiresAt := time.Now().Add(sessionTokenLifetime)
	if err := s.proxyRepo.SetSessionTokenHash(ctx, tenantID, proxyID, sessionTokenHash, expiresAt); err != nil {
		return "", time.Time{}, fmt.Errorf("store session token: %w", err)
	}
	return sessionToken, expiresAt, nil
}

func generateSessionToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}
	return hex.EncodeToString(raw), nil
}

func hashSessionToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (s *RegistrationService) UpdateHealth(ctx context.Context, tenantID, controllerID uuid.UUID) error {
	controller, err := s.controllerRepo.GetByID(ctx, tenantID, controllerID)
	if err != nil {
		return fmt.Errorf("lookup controller: %w", err)
	}
	if controller == nil {
		return domain.NewRegistrationError("controller not found")
	}

	now := time.Now()
	controller.LastHealthAt = &now
	controller.Status = domain.AgentStatusHealthy

	if err := s.controllerRepo.Update(ctx, tenantID, controller); err != nil {
		return fmt.Errorf("update controller: %w", err)
	}
	return nil
}

func generateDeviceCode() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(900000))
	return fmt.Sprintf("%06d", n.Int64()+100000)
}

func generateSharedSecret() (string, error) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}
	return hex.EncodeToString(secret), nil
}

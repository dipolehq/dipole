/*
Package registration handles Agent Controller and Proxy registration with Cloud.

Validates device codes (6-digit, single-use, 1-hour expiry), issues proxy
registration tokens, and tracks agent health. Controllers register on startup,
proxies register via tokens obtained from their Controller.
*/
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

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
)

type RegistrationService struct {
	controllerRepo repository.AgentControllerRepository
	proxyRepo      repository.AgentProxyRepository
	tokenRepo      repository.RegistrationTokenRepository
	logger         interface{ Info(string, ...interface{}) }
}

func NewRegistrationService(controllerRepo repository.AgentControllerRepository, proxyRepo repository.AgentProxyRepository, tokenRepo repository.RegistrationTokenRepository) *RegistrationService {
	return &RegistrationService{controllerRepo: controllerRepo, proxyRepo: proxyRepo, tokenRepo: tokenRepo}
}

func (s *RegistrationService) RegisterController(ctx context.Context, tenantID uuid.UUID, deviceCode string) (*domain.AgentController, error) {
	controller, err := s.controllerRepo.GetByDeviceCode(ctx, tenantID, deviceCode)
	if err != nil {
		return nil, fmt.Errorf("invalid or expired device code")
	}

	if controller.Status != domain.AgentStatusUnregistered {
		return nil, fmt.Errorf("device code already used")
	}

	if controller.DeviceCodeExpiresAt != nil && time.Now().After(*controller.DeviceCodeExpiresAt) {
		return nil, fmt.Errorf("device code expired")
	}

	controller.Status = domain.AgentStatusRegistered
	controller.DeviceCode = ""

	// Generate shared secret for Kerberos attestation signing (HMAC-SHA256)
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, fmt.Errorf("generate shared secret: %w", err)
	}
	controller.SharedSecret = hex.EncodeToString(secret)

	if err := s.controllerRepo.Update(ctx, tenantID, controller); err != nil {
		return nil, fmt.Errorf("update controller: %w", err)
	}

	return controller, nil
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
		return "", fmt.Errorf("controller not found: %w", err)
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
		return "", fmt.Errorf("controller not found: %w", err)
	}

	if controller.Status != domain.AgentStatusRegistered && controller.Status != domain.AgentStatusHealthy {
		return "", fmt.Errorf("controller not registered")
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

// RegisterProxy validates and consumes a registration token, persists a new
// AgentProxy, and issues that proxy a bearer session token. The session
// token (not the one-time registration token) is what authenticates every
// subsequent gRPC call the proxy makes — see grpc.SessionAuthInterceptor.
// Only its SHA-256 hash is stored; the raw token is returned exactly once
// and never persisted or logged in full.
func (s *RegistrationService) RegisterProxy(ctx context.Context, tenantID uuid.UUID, registrationToken string) (*domain.AgentProxy, string, error) {
	if registrationToken == "" {
		return nil, "", domain.NewRegistrationError("empty registration token")
	}

	parts := strings.SplitN(registrationToken, ":", 3)
	if len(parts) != 3 {
		return nil, "", domain.NewRegistrationError("invalid registration token format")
	}

	tokenTenantID, err := uuid.Parse(parts[0])
	if err != nil {
		return nil, "", domain.NewRegistrationError("invalid tenant_id in token")
	}
	if tokenTenantID != tenantID {
		return nil, "", domain.NewRegistrationError("token tenant mismatch")
	}

	controllerID, err := uuid.Parse(parts[1])
	if err != nil {
		return nil, "", domain.NewRegistrationError("invalid controller_id in token")
	}

	// Verify token is valid, not consumed, not expired — and consume it atomically
	hash := sha256.Sum256([]byte(registrationToken))
	tokenHash := hex.EncodeToString(hash[:])

	consumed, err := s.tokenRepo.Consume(ctx, tenantID, tokenHash)
	if err != nil || consumed == nil {
		return nil, "", domain.NewRegistrationError("invalid or already consumed registration token")
	}

	// Verify controller exists and is registered/healthy
	controller, err := s.controllerRepo.GetByID(ctx, tenantID, controllerID)
	if err != nil {
		return nil, "", fmt.Errorf("lookup controller: %w", err)
	}
	if controller == nil {
		return nil, "", domain.NewRegistrationError("controller not found")
	}
	if controller.Status != domain.AgentStatusRegistered && controller.Status != domain.AgentStatusHealthy {
		return nil, "", domain.NewRegistrationError("controller not in valid state for proxy registration")
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
		return nil, "", fmt.Errorf("persist proxy: %w", err)
	}

	sessionToken, err := generateSessionToken()
	if err != nil {
		return nil, "", fmt.Errorf("generate session token: %w", err)
	}
	sessionTokenHash := hashSessionToken(sessionToken)
	if err := s.proxyRepo.SetSessionTokenHash(ctx, tenantID, proxy.ID, sessionTokenHash); err != nil {
		return nil, "", fmt.Errorf("store session token: %w", err)
	}

	return proxy, sessionToken, nil
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
		return err
	}

	now := time.Now()
	controller.LastHealthAt = &now
	controller.Status = domain.AgentStatusHealthy

	return s.controllerRepo.Update(ctx, tenantID, controller)
}

func generateDeviceCode() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(900000))
	return fmt.Sprintf("%06d", n.Int64()+100000)
}

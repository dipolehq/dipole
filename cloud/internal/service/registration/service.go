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
	logger         interface{ Info(string, ...interface{}) }
}

func NewRegistrationService(controllerRepo repository.AgentControllerRepository, proxyRepo repository.AgentProxyRepository) *RegistrationService {
	return &RegistrationService{controllerRepo: controllerRepo, proxyRepo: proxyRepo}
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

	// Generate token issued by Cloud — Cloud recognizes its own tokens
	rawBytes := make([]byte, 32)
	if _, err := rand.Read(rawBytes); err != nil {
		return "", fmt.Errorf("generate random token: %w", err)
	}
	token := fmt.Sprintf("%s:%s:%x", tenantID, controllerID, rawBytes)

	return token, nil
}

func (s *RegistrationService) RegisterProxy(ctx context.Context, tenantID uuid.UUID, registrationToken string) (*domain.AgentProxy, error) {
	if registrationToken == "" {
		return nil, fmt.Errorf("empty registration token")
	}

	// Parse token format: "tenantID:controllerID:random"
	parts := strings.SplitN(registrationToken, ":", 3)
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid registration token format")
	}

	tokenTenantID, err := uuid.Parse(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid tenant_id in token")
	}
	if tokenTenantID != tenantID {
		return nil, fmt.Errorf("token tenant mismatch")
	}

	controllerID, err := uuid.Parse(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid controller_id in token")
	}

	// Verify controller exists and is registered/healthy
	controller, err := s.controllerRepo.GetByID(ctx, tenantID, controllerID)
	if err != nil {
		return nil, fmt.Errorf("controller not found")
	}
	if controller == nil {
		return nil, fmt.Errorf("controller not found")
	}
	if controller.Status != domain.AgentStatusRegistered && controller.Status != domain.AgentStatusHealthy {
		return nil, fmt.Errorf("controller not in valid state for proxy registration")
	}

	// Create and persist proxy
	now := time.Now()
	proxy := &domain.AgentProxy{
		ID:                uuid.New(),
		TenantID:          tenantID,
		AgentControllerID: &controllerID,
		Status:            domain.AgentStatusRegistered,
		RegisteredAt:      &now,
	}

	if err := s.proxyRepo.Create(ctx, tenantID, proxy); err != nil {
		return nil, fmt.Errorf("persist proxy: %w", err)
	}

	return proxy, nil
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

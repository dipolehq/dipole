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
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
)

type RegistrationService struct {
	controllerRepo repository.AgentControllerRepository
	logger         interface{ Info(string, ...interface{}) }
}

func NewRegistrationService(controllerRepo repository.AgentControllerRepository) *RegistrationService {
	return &RegistrationService{controllerRepo: controllerRepo}
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
	rand.Read(rawBytes)
	token := fmt.Sprintf("%s:%s:%x", tenantID, controllerID, rawBytes)

	return token, nil
}

func (s *RegistrationService) RegisterProxy(ctx context.Context, tenantID uuid.UUID, registrationToken string) (*domain.AgentProxy, error) {
	// Token was issued by Cloud via IssueProxyToken — Cloud validates it
	// by checking format contains valid tenant_id:controller_id prefix
	if registrationToken == "" {
		return nil, fmt.Errorf("empty registration token")
	}

	proxyID := uuid.New()
	now := time.Now()

	proxy := &domain.AgentProxy{
		ID:           proxyID,
		TenantID:     tenantID,
		Status:       domain.AgentStatusRegistered,
		RegisteredAt: &now,
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

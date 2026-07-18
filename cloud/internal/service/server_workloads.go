// Package service — server_workloads.go implements business logic and
// validation for Server Workloads, sitting between transport and repository.
package service

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
)

type ServerWorkloadService struct {
	repo repository.ServerWorkloadRepository
}

func NewServerWorkloadService(repo repository.ServerWorkloadRepository) *ServerWorkloadService {
	return &ServerWorkloadService{repo: repo}
}

func (s *ServerWorkloadService) List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.ServerWorkload, int, error) {
	return s.repo.List(ctx, tenantID, opts)
}

func (s *ServerWorkloadService) Get(ctx context.Context, tenantID, id uuid.UUID) (*domain.ServerWorkload, error) {
	sw, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if sw == nil {
		return nil, domain.ErrNotFound
	}
	return sw, nil
}

func (s *ServerWorkloadService) Create(ctx context.Context, tenantID uuid.UUID, sw *domain.ServerWorkload) error {
	normalizeServerWorkload(sw)
	if err := validateServerWorkload(sw); err != nil {
		return err
	}
	return s.repo.Create(ctx, tenantID, sw)
}

func (s *ServerWorkloadService) Update(ctx context.Context, tenantID uuid.UUID, sw *domain.ServerWorkload) error {
	normalizeServerWorkload(sw)
	if err := validateServerWorkload(sw); err != nil {
		return err
	}

	existing, err := s.repo.GetByID(ctx, tenantID, sw.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return domain.ErrNotFound
	}
	return s.repo.Update(ctx, tenantID, sw)
}

func (s *ServerWorkloadService) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	existing, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return domain.ErrNotFound
	}
	return s.repo.Delete(ctx, tenantID, id)
}

func normalizeServerWorkload(sw *domain.ServerWorkload) {
	sw.Name = strings.TrimSpace(sw.Name)
	sw.Description = strings.TrimSpace(sw.Description)
	sw.Host = strings.TrimSpace(sw.Host)
	if sw.TransportProtocol == "" {
		sw.TransportProtocol = "TCP"
	}
}

func validateServerWorkload(sw *domain.ServerWorkload) error {
	if sw.Name == "" {
		return domain.NewValidationError("name", "name is required")
	}
	if sw.Host == "" {
		return domain.NewValidationError("host", "host is required")
	}
	if sw.Port < 1 || sw.Port > 65535 {
		return domain.NewValidationError("port", "port must be between 1 and 65535")
	}
	if sw.RequestedPort != 0 && (sw.RequestedPort < 1 || sw.RequestedPort > 65535) {
		return domain.NewValidationError("forwardToPort", "forward-to port must be between 1 and 65535")
	}
	return nil
}

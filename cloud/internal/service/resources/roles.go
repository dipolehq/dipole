// roles.go implements business logic and validation for Roles.
package resources

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
)

type RoleService struct {
	repo repository.RoleRepository
}

func NewRoleService(repo repository.RoleRepository) *RoleService {
	return &RoleService{repo: repo}
}

func (s *RoleService) List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.Role, int, error) {
	return s.repo.List(ctx, tenantID, opts)
}

func (s *RoleService) Get(ctx context.Context, tenantID, id uuid.UUID) (*domain.Role, error) {
	role, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, domain.ErrNotFound
	}
	return role, nil
}

func (s *RoleService) Create(ctx context.Context, tenantID uuid.UUID, role *domain.Role) error {
	normalizeRole(role)
	if err := validateRole(role); err != nil {
		return err
	}
	return s.repo.Create(ctx, tenantID, role)
}

func (s *RoleService) Update(ctx context.Context, tenantID uuid.UUID, role *domain.Role) error {
	normalizeRole(role)
	if err := validateRole(role); err != nil {
		return err
	}

	existing, err := s.repo.GetByID(ctx, tenantID, role.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return domain.ErrNotFound
	}
	return s.repo.Update(ctx, tenantID, role)
}

func (s *RoleService) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	existing, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return domain.ErrNotFound
	}
	return s.repo.Delete(ctx, tenantID, id)
}

func normalizeRole(role *domain.Role) {
	role.Name = strings.TrimSpace(role.Name)
	role.Description = strings.TrimSpace(role.Description)
}

func validateRole(role *domain.Role) error {
	if role.Name == "" {
		return domain.NewValidationError("name", "name is required")
	}
	return nil
}

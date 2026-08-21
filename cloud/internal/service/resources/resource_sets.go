// resource_sets.go implements business logic and validation for Resource Sets.
package resources

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
	"github.com/dipolehq/dipole/cloud/internal/scope"
	"github.com/dipolehq/dipole/cloud/internal/service"
)

type ResourceSetService struct {
	repo    repository.ResourceSetRepository
	limiter *service.PlanLimiter
}

func NewResourceSetService(repo repository.ResourceSetRepository) *ResourceSetService {
	return &ResourceSetService{repo: repo}
}

func (s *ResourceSetService) SetLimiter(l *service.PlanLimiter) { s.limiter = l }

func (s *ResourceSetService) List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.ResourceSet, int, error) {
	return s.repo.List(ctx, tenantID, opts)
}

func (s *ResourceSetService) Get(ctx context.Context, tenantID, id uuid.UUID) (*domain.ResourceSet, error) {
	rs, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if rs == nil {
		return nil, domain.ErrNotFound
	}
	return rs, nil
}

func (s *ResourceSetService) GetResourceCounts(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]domain.ResourceSetCounts, error) {
	return s.repo.GetResourceCounts(ctx, tenantID, ids)
}

func (s *ResourceSetService) Create(ctx context.Context, tenantID uuid.UUID, rs *domain.ResourceSet) error {
	if s.limiter != nil {
		if err := s.limiter.Check(ctx, tenantID, service.ResourceResourceSets); err != nil {
			return err
		}
	}
	normalizeResourceSet(rs)
	if err := validateResourceSet(rs); err != nil {
		return err
	}
	return s.repo.Create(ctx, tenantID, rs)
}

func (s *ResourceSetService) Update(ctx context.Context, tenantID uuid.UUID, rs *domain.ResourceSet) error {
	normalizeResourceSet(rs)
	if err := validateResourceSet(rs); err != nil {
		return err
	}

	existing, err := s.repo.GetByID(ctx, tenantID, rs.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return domain.ErrNotFound
	}
	return s.repo.Update(ctx, tenantID, rs)
}

func (s *ResourceSetService) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	existing, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return domain.ErrNotFound
	}
	if existing.ID == scope.DefaultResourceSetID {
		return domain.NewValidationError("id", "the Default Resource Set cannot be deleted")
	}
	return s.repo.Delete(ctx, tenantID, id)
}

func normalizeResourceSet(rs *domain.ResourceSet) {
	rs.Name = strings.TrimSpace(rs.Name)
	rs.Description = strings.TrimSpace(rs.Description)
}

func validateResourceSet(rs *domain.ResourceSet) error {
	if rs.Name == "" {
		return domain.NewValidationError("name", "name is required")
	}
	return nil
}

/*
Package resources — access_policies.go implements business logic and
validation for Access Policies.
*/
package resources

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
	"github.com/dipolehq/dipole/cloud/internal/service"
)

type AccessPolicyService struct {
	repo    repository.AccessPolicyRepository
	limiter *service.PlanLimiter
}

func NewAccessPolicyService(repo repository.AccessPolicyRepository) *AccessPolicyService {
	return &AccessPolicyService{repo: repo}
}

func (s *AccessPolicyService) SetLimiter(l *service.PlanLimiter) { s.limiter = l }

func (s *AccessPolicyService) List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.AccessPolicy, int, error) {
	return s.repo.List(ctx, tenantID, opts)
}

func (s *AccessPolicyService) Get(ctx context.Context, tenantID, id uuid.UUID) (*domain.AccessPolicy, error) {
	p, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, domain.ErrNotFound
	}
	return p, nil
}

func (s *AccessPolicyService) Create(ctx context.Context, tenantID uuid.UUID, p *domain.AccessPolicy) error {
	if s.limiter != nil {
		if err := s.limiter.Check(ctx, tenantID, service.ResourceAccessPolicies); err != nil {
			return err
		}
	}
	normalizeAccessPolicy(p)
	if err := validateAccessPolicy(p); err != nil {
		return err
	}
	return s.repo.Create(ctx, tenantID, p)
}

func (s *AccessPolicyService) Update(ctx context.Context, tenantID uuid.UUID, p *domain.AccessPolicy) error {
	normalizeAccessPolicy(p)
	if err := validateAccessPolicy(p); err != nil {
		return err
	}
	existing, err := s.repo.GetByID(ctx, tenantID, p.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return domain.ErrNotFound
	}
	return s.repo.Update(ctx, tenantID, p)
}

func (s *AccessPolicyService) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	existing, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return domain.ErrNotFound
	}
	return s.repo.Delete(ctx, tenantID, id)
}

func normalizeAccessPolicy(p *domain.AccessPolicy) {
	p.Name = strings.TrimSpace(p.Name)
	p.Description = strings.TrimSpace(p.Description)
}

func validateAccessPolicy(p *domain.AccessPolicy) error {
	if p.Name == "" {
		return domain.NewValidationError("name", "name is required")
	}
	if p.ClientWorkloadID == uuid.Nil {
		return domain.NewValidationError("clientWorkloadId", "client workload is required")
	}
	if p.ServerWorkloadID == uuid.Nil {
		return domain.NewValidationError("serverWorkloadId", "server workload is required")
	}
	return nil
}

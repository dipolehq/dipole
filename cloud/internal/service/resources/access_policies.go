// access_policies.go implements business logic and validation for Access Policies.
package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
	"github.com/dipolehq/dipole/cloud/internal/service"
)

type AccessPolicyService struct {
	repo                repository.AccessPolicyRepository
	clientWorkloads     repository.ClientWorkloadRepository
	serverWorkloads     repository.ServerWorkloadRepository
	trustProviders      repository.TrustProviderRepository
	credentialProviders repository.CredentialProviderRepository
	accessConditions    repository.AccessConditionRepository
	limiter             *service.PlanLimiter
}

func NewAccessPolicyService(
	repo repository.AccessPolicyRepository,
	clientWorkloads repository.ClientWorkloadRepository,
	serverWorkloads repository.ServerWorkloadRepository,
	trustProviders repository.TrustProviderRepository,
	credentialProviders repository.CredentialProviderRepository,
	accessConditions repository.AccessConditionRepository,
) *AccessPolicyService {
	return &AccessPolicyService{
		repo:                repo,
		clientWorkloads:     clientWorkloads,
		serverWorkloads:     serverWorkloads,
		trustProviders:      trustProviders,
		credentialProviders: credentialProviders,
		accessConditions:    accessConditions,
	}
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
	if err := s.validateReferencesBelongToTenant(ctx, tenantID, p); err != nil {
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
	if err := s.validateReferencesBelongToTenant(ctx, tenantID, p); err != nil {
		return err
	}
	return s.repo.Update(ctx, tenantID, p)
}

func (s *AccessPolicyService) validateReferencesBelongToTenant(ctx context.Context, tenantID uuid.UUID, p *domain.AccessPolicy) error {
	cw, err := s.clientWorkloads.GetByID(ctx, tenantID, p.ClientWorkloadID)
	if err != nil {
		return fmt.Errorf("resolve client workload: %w", err)
	}
	if cw == nil {
		return domain.NewValidationError("clientWorkloadId", "client workload not found")
	}

	sw, err := s.serverWorkloads.GetByID(ctx, tenantID, p.ServerWorkloadID)
	if err != nil {
		return fmt.Errorf("resolve server workload: %w", err)
	}
	if sw == nil {
		return domain.NewValidationError("serverWorkloadId", "server workload not found")
	}

	for _, id := range p.TrustProviderIDs {
		tp, err := s.trustProviders.GetByID(ctx, tenantID, id)
		if err != nil {
			return fmt.Errorf("resolve trust provider %s: %w", id, err)
		}
		if tp == nil {
			return domain.NewValidationError("trustProviderIds", fmt.Sprintf("trust provider %s not found", id))
		}
	}

	for _, pcp := range p.CredentialProviderIDs {
		cp, err := s.credentialProviders.GetByID(ctx, tenantID, pcp.CredentialProviderID)
		if err != nil {
			return fmt.Errorf("resolve credential provider %s: %w", pcp.CredentialProviderID, err)
		}
		if cp == nil {
			return domain.NewValidationError("credentialProviderIds", fmt.Sprintf("credential provider %s not found", pcp.CredentialProviderID))
		}
	}

	for _, id := range p.AccessConditionIDs {
		ac, err := s.accessConditions.GetByID(ctx, tenantID, id)
		if err != nil {
			return fmt.Errorf("resolve access condition %s: %w", id, err)
		}
		if ac == nil {
			return domain.NewValidationError("accessConditionIds", fmt.Sprintf("access condition %s not found", id))
		}
	}

	return nil
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

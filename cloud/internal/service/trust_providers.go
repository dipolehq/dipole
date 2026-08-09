/*
Package service — trust_providers.go implements business logic and
validation for Trust Providers, sitting between transport and repository.
*/
package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/platform"
	"github.com/dipolehq/dipole/cloud/internal/repository"
	"github.com/dipolehq/dipole/cloud/internal/service/trust"
)

// TrustProviderService is stateless with respect to tenant. The Edge SDK Client
// ID and Audience it generates embed the OWNING tenant's subdomain — these are
// the values a customer copies into their GitHub Actions workflow or CLI config
// — so the subdomain is looked up per call from the tenant that's creating the
// provider, not captured once at startup.
type TrustProviderService struct {
	repo    repository.TrustProviderRepository
	tenants repository.TenantRepository
	region  string
	hosts   *platform.Hosts
	limiter *PlanLimiter
}

func NewTrustProviderService(
	repo repository.TrustProviderRepository,
	tenants repository.TenantRepository,
	region string,
	hosts *platform.Hosts,
) *TrustProviderService {
	return &TrustProviderService{repo: repo, tenants: tenants, region: region, hosts: hosts}
}

func (s *TrustProviderService) SetLimiter(l *PlanLimiter) { s.limiter = l }

func (s *TrustProviderService) List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.TrustProvider, int, error) {
	return s.repo.List(ctx, tenantID, opts)
}

func (s *TrustProviderService) Get(ctx context.Context, tenantID, id uuid.UUID) (*domain.TrustProvider, error) {
	tp, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if tp == nil {
		return nil, domain.ErrNotFound
	}
	return tp, nil
}

func (s *TrustProviderService) Create(ctx context.Context, tenantID uuid.UUID, tp *domain.TrustProvider) error {
	if s.limiter != nil {
		if err := s.limiter.Check(ctx, tenantID, ResourceTrustProviders); err != nil {
			return err
		}
	}
	normalizeTrustProvider(tp)
	if err := validateTrustProvider(tp); err != nil {
		return err
	}

	tenant, err := s.tenants.GetByID(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("resolve tenant for edge sdk identifiers: %w", err)
	}
	if tenant == nil {
		return domain.ErrNotFound
	}

	// These two values are immutable after creation and get copied by the
	// customer into their CI/CD config, so refuse to persist a malformed one.
	audience := s.hosts.IdentityIssuerURL(tenant.Subdomain)
	if audience == "" {
		return fmt.Errorf(
			"cannot generate Edge SDK audience: tenant subdomain %q is not a valid hostname label",
			tenant.Subdomain)
	}

	tp.ID = uuid.New()
	tp.EdgeSDKClientID = trust.GenerateEdgeSDKClientID(s.region, tenant.Subdomain, tp.Type, tp.ID)
	tp.EdgeSDKAudience = audience
	return s.repo.Create(ctx, tenantID, tp)
}

func (s *TrustProviderService) Update(ctx context.Context, tenantID uuid.UUID, tp *domain.TrustProvider) error {
	normalizeTrustProvider(tp)
	if err := validateTrustProvider(tp); err != nil {
		return err
	}
	existing, err := s.repo.GetByID(ctx, tenantID, tp.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return domain.ErrNotFound
	}
	tp.EdgeSDKClientID = existing.EdgeSDKClientID
	tp.EdgeSDKAudience = existing.EdgeSDKAudience
	return s.repo.Update(ctx, tenantID, tp)
}

func (s *TrustProviderService) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	existing, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return domain.ErrNotFound
	}
	return s.repo.Delete(ctx, tenantID, id)
}

func normalizeTrustProvider(tp *domain.TrustProvider) {
	tp.Name = strings.TrimSpace(tp.Name)
	tp.Description = strings.TrimSpace(tp.Description)
}

func validateTrustProvider(tp *domain.TrustProvider) error {
	if tp.Name == "" {
		return domain.NewValidationError("name", "name is required")
	}
	if tp.Type == "" {
		return domain.NewValidationError("type", "trust provider type is required")
	}
	return nil
}

// Edge SDK identifier generation lives in service/trust/edge_sdk.go. It used to
// be duplicated here (with a second, unused exported copy in that package),
// which is how the two drifted apart during the subdomain pattern change.

// Package service — trust_providers.go implements business logic and
// validation for Trust Providers, sitting between transport and repository.
package service

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
)

type TrustProviderService struct {
	repo repository.TrustProviderRepository
}

func NewTrustProviderService(repo repository.TrustProviderRepository) *TrustProviderService {
	return &TrustProviderService{repo: repo}
}

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
	normalizeTrustProvider(tp)
	if err := validateTrustProvider(tp); err != nil {
		return err
	}
	tp.ID = uuid.New()
	tp.EdgeSDKClientID = generateEdgeSDKClientID(tenantID, tp)
	tp.EdgeSDKAudience = generateEdgeSDKAudience()
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

func generateEdgeSDKClientID(tenantID uuid.UUID, tp *domain.TrustProvider) string {
	return "dipole:ap-south-1:dev:identity:" + typeShortName(tp.Type) + ":" + tp.ID.String()
}

func generateEdgeSDKAudience() string {
	return "https://dev.id.ap-south-1.dipolehq.com"
}

func typeShortName(t string) string {
	switch t {
	case "kubernetes_service_account":
		return "kubernetes_serviceaccount_token"
	case "github_action":
		return "github_idtoken"
	case "gitlab_job":
		return "gitlab_idtoken"
	case "terraform_cloud":
		return "terraform_cloud_idtoken"
	case "oidc_id_token":
		return "oidc_id_token"
	default:
		return t
	}
}

// edge_validator.go implements EdgeTrustValidator for the Edge API, loading and delegating Trust Provider validation.
package trust

import (
	"context"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
)

type EdgeTrustValidatorService struct {
	tpRepo    repository.TrustProviderRepository
	validator *TrustValidationService
}

func NewEdgeTrustValidator(tpRepo repository.TrustProviderRepository, validator *TrustValidationService) *EdgeTrustValidatorService {
	return &EdgeTrustValidatorService{
		tpRepo:    tpRepo,
		validator: validator,
	}
}

func (s *EdgeTrustValidatorService) ValidateByID(tenantID uuid.UUID, trustProviderID uuid.UUID, evidence *domain.ClientEvidence) (*domain.TrustProviderResult, error) {
	ctx := context.Background()

	tp, err := s.tpRepo.GetByID(ctx, tenantID, trustProviderID)
	if err != nil || tp == nil {
		return &domain.TrustProviderResult{Attested: false, Reason: "TrustProviderNotFound"}, nil
	}

	if !tp.IsActive {
		return &domain.TrustProviderResult{Attested: false, Reason: "TrustProviderInactive"}, nil
	}

	return s.validator.Validate(ctx, tp, evidence)
}

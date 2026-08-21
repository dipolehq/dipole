// service.go provides TrustValidationService, dispatching validation to type-specific validators by Trust Provider type.
package trust

import (
	"context"
	"log/slog"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type TypeValidator interface {
	Validate(ctx context.Context, provider *domain.TrustProvider, evidence *domain.ClientEvidence, jwksCache *JWKSCache) (*domain.TrustProviderResult, error)
}

type TrustValidationService struct {
	jwksCache  *JWKSCache
	validators map[string]TypeValidator
	logger     *slog.Logger
}

func NewTrustValidationService(jwksCache *JWKSCache, logger *slog.Logger) *TrustValidationService {
	return &TrustValidationService{
		jwksCache:  jwksCache,
		validators: make(map[string]TypeValidator),
		logger:     logger,
	}
}

func (s *TrustValidationService) RegisterValidator(providerType string, v TypeValidator) {
	s.validators[providerType] = v
}

func (s *TrustValidationService) JWKSCache() *JWKSCache {
	return s.jwksCache
}

func (s *TrustValidationService) Validate(ctx context.Context, provider *domain.TrustProvider, evidence *domain.ClientEvidence) (*domain.TrustProviderResult, error) {
	v, ok := s.validators[provider.Type]
	if !ok {
		return &domain.TrustProviderResult{
			ProviderID:   provider.ID,
			ProviderName: provider.Name,
			Attested:     false,
			Reason:       "UnsupportedProviderType",
		}, nil
	}

	result, err := v.Validate(ctx, provider, evidence, s.jwksCache)
	if err != nil {
		return nil, err
	}

	result.ProviderID = provider.ID
	result.ProviderName = provider.Name
	return result, nil
}

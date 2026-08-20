// validator.go provides the TrustValidator interface for Trust Provider validation against the policy evaluation engine.
package trust

import (
	"context"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type TrustValidator interface {
	Validate(ctx context.Context, provider *domain.TrustProvider, evidence *domain.ClientEvidence) (*domain.TrustProviderResult, error)
}

type StubTrustValidator struct{}

func (s *StubTrustValidator) Validate(_ context.Context, provider *domain.TrustProvider, _ *domain.ClientEvidence) (*domain.TrustProviderResult, error) {
	return &domain.TrustProviderResult{
		ProviderID:   provider.ID,
		ProviderName: provider.Name,
		Attested:     true,
	}, nil
}

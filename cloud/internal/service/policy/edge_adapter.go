// edge_adapter.go adapts PolicyEvaluationService to the EdgePolicyEvaluator interface used by the Edge API handler.
package policy

import (
	"context"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type EdgePolicyAdapter struct {
	service *PolicyEvaluationService
}

func NewEdgePolicyAdapter(service *PolicyEvaluationService) *EdgePolicyAdapter {
	return &EdgePolicyAdapter{service: service}
}

func (a *EdgePolicyAdapter) Evaluate(ctx context.Context, tenantID uuid.UUID, evidence *domain.ClientEvidence, target *domain.ServerTarget) (*domain.EvaluationResult, error) {
	return a.service.Evaluate(ctx, tenantID, evidence, target)
}

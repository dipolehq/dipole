/*
edge_adapter.go adapts PolicyEvaluationService to the EdgePolicyEvaluator
interface used by the Edge API handler.

The Edge API transport defines EdgePolicyEvaluator without a context parameter.
This adapter bridges that gap by providing context.Background() — acceptable
because the Edge API HTTP handler's request context does not propagate into
this interface (by design of the transport layer).
*/
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

func (a *EdgePolicyAdapter) Evaluate(tenantID uuid.UUID, evidence *domain.ClientEvidence, target *domain.ServerTarget) (*domain.EvaluationResult, error) {
	return a.service.Evaluate(context.Background(), tenantID, evidence, target)
}

/*
Package conditions provides Access Condition evaluation for the policy engine.

ConditionEvaluator checks contextual conditions (time, GeoIP) attached to a policy.
StubConditionEvaluator always returns passed=true (replaced with real evaluators in Job 8).
*/
package conditions

import (
	"context"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type ConditionEvaluator interface {
	EvaluateAll(ctx context.Context, conditions []domain.AccessCondition, evidence *domain.ClientEvidence) (bool, string, error)
}

type StubConditionEvaluator struct{}

func (s *StubConditionEvaluator) EvaluateAll(_ context.Context, _ []domain.AccessCondition, _ *domain.ClientEvidence) (bool, string, error) {
	return true, "", nil
}

// evaluator.go implements Access Condition evaluation (time, GeoIP) for the policy engine; all conditions must pass (AND logic).
package conditions

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type Evaluator struct {
	timeChecker  *TimeConditionChecker
	geoIPChecker *GeoIPConditionChecker
}

func NewConditionEvaluator(geoIPDBPath string, logger *slog.Logger) *Evaluator {
	return &Evaluator{
		timeChecker:  &TimeConditionChecker{},
		geoIPChecker: NewGeoIPConditionChecker(geoIPDBPath, logger),
	}
}

func (e *Evaluator) EvaluateAll(_ context.Context, conditions []domain.AccessCondition, evidence *domain.ClientEvidence) (bool, string, error) {
	for _, cond := range conditions {
		passed, reason, err := e.evaluateOne(cond, evidence)
		if err != nil {
			return false, fmt.Sprintf("condition evaluation error: %v", err), nil
		}
		if !passed {
			return false, reason, nil
		}
	}
	return true, "", nil
}

func (e *Evaluator) evaluateOne(cond domain.AccessCondition, evidence *domain.ClientEvidence) (bool, string, error) {
	switch cond.Type {
	case "time_condition":
		var cfg TimeConditionConfig
		if err := json.Unmarshal(cond.Config, &cfg); err != nil {
			return false, "", fmt.Errorf("parse time condition config: %w", err)
		}
		return e.timeChecker.Check(&cfg)

	case "geoip_condition":
		var cfg GeoIPConditionConfig
		if err := json.Unmarshal(cond.Config, &cfg); err != nil {
			return false, "", fmt.Errorf("parse geoip condition config: %w", err)
		}
		return e.geoIPChecker.Check(evidence.SourceIP, &cfg)

	default:
		return false, fmt.Sprintf("unsupported condition type: %s", cond.Type), nil
	}
}

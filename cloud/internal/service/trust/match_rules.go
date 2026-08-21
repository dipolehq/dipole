// match_rules.go evaluates Trust Provider match rules (AND logic, dot-notation, wildcard) against JWT/attestation claims.
package trust

import (
	"fmt"
	"strings"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type MatchRule struct {
	Field string `json:"field"`
	Value string `json:"value"`
}

type MatchRuleFailure struct {
	Attribute     string
	ExpectedValue string
	ActualValue   string
}

func EvaluateMatchRules(rules []MatchRule, claims map[string]interface{}) (bool, *MatchRuleFailure) {
	for _, rule := range rules {
		actual, found := resolveClaimPath(claims, rule.Field)

		if rule.Value == "*" {
			if !found || actual == "" {
				return false, &MatchRuleFailure{
					Attribute:     rule.Field,
					ExpectedValue: "*",
					ActualValue:   "",
				}
			}
			continue
		}

		if !found || actual != rule.Value {
			return false, &MatchRuleFailure{
				Attribute:     rule.Field,
				ExpectedValue: rule.Value,
				ActualValue:   actual,
			}
		}
	}
	return true, nil
}

func resolveClaimPath(claims map[string]interface{}, path string) (string, bool) {
	if val, ok := claims[path]; ok {
		return fmt.Sprint(val), true
	}

	parts := strings.Split(path, ".")
	current := claims

	for i := 0; i < len(parts); i++ {
		key := strings.Join(parts[:i+1], ".")
		val, ok := current[key]
		if !ok {
			continue
		}

		remaining := parts[i+1:]
		if len(remaining) == 0 {
			return fmt.Sprint(val), true
		}

		nested, ok := val.(map[string]interface{})
		if !ok {
			return "", false
		}

		subPath := strings.Join(remaining, ".")
		return resolveClaimPath(nested, subPath)
	}

	return "", false
}

func toMatchRules(domainRules []domain.MatchRule) []MatchRule {
	rules := make([]MatchRule, len(domainRules))
	for i, r := range domainRules {
		rules[i] = MatchRule{Field: r.Attribute, Value: r.Value}
	}
	return rules
}

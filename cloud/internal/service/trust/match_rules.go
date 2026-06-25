/*
match_rules.go evaluates Trust Provider match rules against JWT claims.

EvaluateMatchRules checks ALL rules (AND logic) against a claims map. Supports
dot-notation for nested claims with special handling for compound keys like
"kubernetes.io". Wildcard value "*" matches any non-empty field. Returns
detailed failure info (attribute, expected, actual) on mismatch.
*/
package trust

import (
	"fmt"
	"strings"
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
	// Try the full path as a single key first (handles "kubernetes.io" as atomic key)
	if val, ok := claims[path]; ok {
		return fmt.Sprint(val), true
	}

	// Progressively split from the left, trying compound keys
	parts := strings.Split(path, ".")
	current := claims

	for i := 0; i < len(parts); i++ {
		// Try joining parts[0..i] as a single compound key
		key := strings.Join(parts[:i+1], ".")
		val, ok := current[key]
		if !ok {
			continue
		}

		// If this is the last segment, return the value
		remaining := parts[i+1:]
		if len(remaining) == 0 {
			return fmt.Sprint(val), true
		}

		// Otherwise, navigate into the nested map
		nested, ok := val.(map[string]interface{})
		if !ok {
			return "", false
		}

		// Recurse with remaining path
		subPath := strings.Join(remaining, ".")
		return resolveClaimPath(nested, subPath)
	}

	return "", false
}

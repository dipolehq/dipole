/*
match_rules.go evaluates Trust Provider match rules against JWT claims.

EvaluateMatchRules checks ALL rules (AND logic) against a claims map. Supports
dot-notation for nested claims with special handling for compound keys like
"kubernetes.io". Wildcard value "*" matches any non-empty field. Returns
detailed failure info (attribute, expected, actual) on mismatch.

# Claim value type semantics

rule.Value is always a plain string (it's user-configured Trust Provider
config), but the claim value it's compared against comes from decoded JSON
and can be any of the types encoding/json produces: string, float64 (JSON
has no distinct int type), bool, nil, []interface{}, or map[string]interface{}.
resolveClaimPath converts whatever it finds to a string via fmt.Sprint
before comparing, so match rules work correctly and predictably against
the JSON scalar types:
  - string: compared as-is (the common case: sub, repository, email, etc.)
  - number (float64): fmt.Sprint renders whole numbers without a decimal
    point ("42", not "42.000000") and fractional numbers with one
    ("42.5"), so a rule.Value of "42" correctly matches a claim of 42.
  - bool: renders as "true" or "false", matchable with rule.Value "true"/"false".
  - null: renders as "<nil>". No rule.Value will ever equal this except
    the literal string "<nil>", which is not a realistic customer input —
    in practice this means a null claim value can only satisfy a "*"
    (wildcard) rule if the claim key exists (found=true) with a non-empty
    stringified value... but "<nil>" IS non-empty, so a wildcard rule
    against an explicit null actually passes. This is almost certainly
    not what a customer configuring "field must be present and non-empty"
    intends, but it is what the code does today; documented here rather
    than silently changed, since "should a wildcard rule treat null as
    absent" is a product decision, not a bug fix.

# Array and object claim values are not usefully matchable

If the value at the resolved path is a JSON array or object rather than a
scalar, fmt.Sprint produces Go's internal representation, not JSON: e.g.
a claim value ["prod","web"] renders as the string "[prod web]", and
{"a":1} renders as "map[a:1]". These are deterministic (Go's fmt sorts
map keys for stable output) but not values any customer configuring a
Trust Provider would reasonably guess or write into a match rule's Value
field. There is no supported way today to match "array contains X" or
match on individual array elements — a match rule against an array-typed
claim can, in practice, only ever succeed by a customer copy-pasting the
exact "[elem1 elem2]" string, which is not a realistic or documented
usage pattern. If per-element or "contains" array matching is needed,
that is a product decision (what should it mean — exact match on the
whole array, any-element match, all-elements match?) requiring its own
design, not something this function does today.
*/
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

func toMatchRules(domainRules []domain.MatchRule) []MatchRule {
	rules := make([]MatchRule, len(domainRules))
	for i, r := range domainRules {
		rules[i] = MatchRule{Field: r.Attribute, Value: r.Value}
	}
	return rules
}

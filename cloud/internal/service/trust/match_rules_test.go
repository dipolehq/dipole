package trust

import "testing"

func TestEvaluateMatchRules_SimpleKey(t *testing.T) {
	claims := map[string]interface{}{"sub": "system:serviceaccount:prod:myapp"}
	rules := []MatchRule{{Field: "sub", Value: "system:serviceaccount:prod:myapp"}}

	passed, failure := EvaluateMatchRules(rules, claims)
	if !passed {
		t.Errorf("expected pass, got failure: %+v", failure)
	}
}

func TestEvaluateMatchRules_NestedK8sNamespace(t *testing.T) {
	claims := map[string]interface{}{
		"kubernetes.io": map[string]interface{}{
			"namespace": "production",
			"serviceaccount": map[string]interface{}{
				"name": "myapp-sa",
			},
		},
	}

	rules := []MatchRule{{Field: "kubernetes.io.namespace", Value: "production"}}
	passed, failure := EvaluateMatchRules(rules, claims)
	if !passed {
		t.Errorf("expected pass, got failure: %+v", failure)
	}
}

func TestEvaluateMatchRules_DeepNested(t *testing.T) {
	claims := map[string]interface{}{
		"kubernetes.io": map[string]interface{}{
			"serviceaccount": map[string]interface{}{
				"name": "myapp-sa",
			},
		},
	}

	rules := []MatchRule{{Field: "kubernetes.io.serviceaccount.name", Value: "myapp-sa"}}
	passed, failure := EvaluateMatchRules(rules, claims)
	if !passed {
		t.Errorf("expected pass, got failure: %+v", failure)
	}
}

func TestEvaluateMatchRules_MissingKey_Fails(t *testing.T) {
	claims := map[string]interface{}{"sub": "x"}
	rules := []MatchRule{{Field: "nonexistent", Value: "anything"}}

	passed, failure := EvaluateMatchRules(rules, claims)
	if passed {
		t.Error("expected failure for missing key")
	}
	if failure.Attribute != "nonexistent" {
		t.Errorf("attribute = %q, want %q", failure.Attribute, "nonexistent")
	}
}

func TestEvaluateMatchRules_Wildcard_PassesIfExists(t *testing.T) {
	claims := map[string]interface{}{"repository": "myorg/myrepo"}
	rules := []MatchRule{{Field: "repository", Value: "*"}}

	passed, _ := EvaluateMatchRules(rules, claims)
	if !passed {
		t.Error("expected wildcard to pass for existing non-empty field")
	}
}

func TestEvaluateMatchRules_Wildcard_FailsIfEmpty(t *testing.T) {
	claims := map[string]interface{}{"repository": ""}
	rules := []MatchRule{{Field: "repository", Value: "*"}}

	passed, _ := EvaluateMatchRules(rules, claims)
	if passed {
		t.Error("expected wildcard to fail for empty field")
	}
}

func TestEvaluateMatchRules_Wildcard_FailsIfMissing(t *testing.T) {
	claims := map[string]interface{}{"other": "x"}
	rules := []MatchRule{{Field: "repository", Value: "*"}}

	passed, _ := EvaluateMatchRules(rules, claims)
	if passed {
		t.Error("expected wildcard to fail for missing field")
	}
}

func TestEvaluateMatchRules_ANDLogic_OneFailsAllFail(t *testing.T) {
	claims := map[string]interface{}{
		"kubernetes.io": map[string]interface{}{
			"namespace": "staging",
			"serviceaccount": map[string]interface{}{
				"name": "myapp-sa",
			},
		},
	}

	rules := []MatchRule{
		{Field: "kubernetes.io.namespace", Value: "production"},
		{Field: "kubernetes.io.serviceaccount.name", Value: "myapp-sa"},
	}

	passed, failure := EvaluateMatchRules(rules, claims)
	if passed {
		t.Error("expected failure (namespace mismatch)")
	}
	if failure.Attribute != "kubernetes.io.namespace" {
		t.Errorf("attribute = %q, want %q", failure.Attribute, "kubernetes.io.namespace")
	}
	if failure.ExpectedValue != "production" {
		t.Errorf("expected = %q, want %q", failure.ExpectedValue, "production")
	}
	if failure.ActualValue != "staging" {
		t.Errorf("actual = %q, want %q", failure.ActualValue, "staging")
	}
}

func TestEvaluateMatchRules_ValueMismatch(t *testing.T) {
	claims := map[string]interface{}{"repository": "other/repo"}
	rules := []MatchRule{{Field: "repository", Value: "myorg/myrepo"}}

	passed, failure := EvaluateMatchRules(rules, claims)
	if passed {
		t.Error("expected failure for value mismatch")
	}
	if failure.ActualValue != "other/repo" {
		t.Errorf("actual = %q, want %q", failure.ActualValue, "other/repo")
	}
}

func TestEvaluateMatchRules_NumericClaim_IntegerValued(t *testing.T) {
	// JSON numbers decode to float64; an integer-valued number must
	// stringify without a decimal point ("42", not "42.000000") so a
	// rule.Value of "42" matches it.
	claims := map[string]interface{}{"count": float64(42)}
	rules := []MatchRule{{Field: "count", Value: "42"}}

	passed, failure := EvaluateMatchRules(rules, claims)
	if !passed {
		t.Errorf("expected pass for integer-valued float64 claim, got failure: %+v", failure)
	}
}

func TestEvaluateMatchRules_NumericClaim_FractionalValued(t *testing.T) {
	claims := map[string]interface{}{"score": float64(4.5)}
	rules := []MatchRule{{Field: "score", Value: "4.5"}}

	passed, failure := EvaluateMatchRules(rules, claims)
	if !passed {
		t.Errorf("expected pass for fractional float64 claim, got failure: %+v", failure)
	}
}

func TestEvaluateMatchRules_NumericClaim_Mismatch(t *testing.T) {
	claims := map[string]interface{}{"count": float64(42)}
	rules := []MatchRule{{Field: "count", Value: "43"}}

	passed, failure := EvaluateMatchRules(rules, claims)
	if passed {
		t.Error("expected failure for numeric mismatch")
	}
	if failure.ActualValue != "42" {
		t.Errorf("actual = %q, want %q", failure.ActualValue, "42")
	}
}

func TestEvaluateMatchRules_BooleanClaim_True(t *testing.T) {
	claims := map[string]interface{}{"active": true}
	rules := []MatchRule{{Field: "active", Value: "true"}}

	passed, failure := EvaluateMatchRules(rules, claims)
	if !passed {
		t.Errorf("expected pass for boolean true claim, got failure: %+v", failure)
	}
}

func TestEvaluateMatchRules_BooleanClaim_False(t *testing.T) {
	claims := map[string]interface{}{"active": false}
	rules := []MatchRule{{Field: "active", Value: "false"}}

	passed, failure := EvaluateMatchRules(rules, claims)
	if !passed {
		t.Errorf("expected pass for boolean false claim, got failure: %+v", failure)
	}
}

func TestEvaluateMatchRules_BooleanClaim_WildcardTreatsFalseAsNonEmpty(t *testing.T) {
	// Documents current behavior: the wildcard rule only checks that the
	// stringified value is non-empty, and "false" is a non-empty string
	// — so a wildcard rule against a boolean-false claim passes. This
	// mirrors the "*" rule's existing documented semantics (checks
	// presence + non-empty string, not truthiness).
	claims := map[string]interface{}{"active": false}
	rules := []MatchRule{{Field: "active", Value: "*"}}

	passed, _ := EvaluateMatchRules(rules, claims)
	if !passed {
		t.Error("expected wildcard to pass for a present boolean-false claim (non-empty string \"false\")")
	}
}

func TestEvaluateMatchRules_NullClaim_WildcardTreatsNullAsNonEmpty(t *testing.T) {
	// Documents current behavior: fmt.Sprint(nil) == "<nil>", which is a
	// non-empty string, so a wildcard rule against an explicit JSON null
	// claim value passes. See match_rules.go's package doc for why this
	// is left as documented behavior rather than silently changed.
	claims := map[string]interface{}{"maybe_present": nil}
	rules := []MatchRule{{Field: "maybe_present", Value: "*"}}

	passed, _ := EvaluateMatchRules(rules, claims)
	if !passed {
		t.Error("expected wildcard to pass for an explicit null claim (documented current behavior)")
	}
}

func TestEvaluateMatchRules_NullClaim_ExactValueNeverMatchesRealisticInput(t *testing.T) {
	claims := map[string]interface{}{"maybe_present": nil}
	rules := []MatchRule{{Field: "maybe_present", Value: "something"}}

	passed, failure := EvaluateMatchRules(rules, claims)
	if passed {
		t.Error("expected failure: no realistic rule.Value matches a null claim")
	}
	if failure.ActualValue != "<nil>" {
		t.Errorf("actual = %q, want %q (documents fmt.Sprint(nil) stringification)", failure.ActualValue, "<nil>")
	}
}

func TestEvaluateMatchRules_ArrayClaim_NotUsefullyMatchable(t *testing.T) {
	// Documents current behavior: an array-typed claim stringifies via
	// Go's fmt package, not JSON — "[prod web]", not "[\"prod\",\"web\"]"
	// — so no realistic customer-authored rule.Value can match it. This
	// is a documented limitation (see match_rules.go package doc), not
	// something this test asserts should change.
	claims := map[string]interface{}{"tags": []interface{}{"prod", "web"}}
	rules := []MatchRule{{Field: "tags", Value: "prod"}}

	passed, failure := EvaluateMatchRules(rules, claims)
	if passed {
		t.Error("expected failure: array claims don't match individual elements")
	}
	if failure.ActualValue != "[prod web]" {
		t.Errorf("actual = %q, want %q (documents Go fmt stringification of []interface{})", failure.ActualValue, "[prod web]")
	}
}

func TestEvaluateMatchRules_ArrayClaim_WildcardStillPassesIfNonEmpty(t *testing.T) {
	claims := map[string]interface{}{"tags": []interface{}{"prod", "web"}}
	rules := []MatchRule{{Field: "tags", Value: "*"}}

	passed, _ := EvaluateMatchRules(rules, claims)
	if !passed {
		t.Error("expected wildcard to pass for a present, non-empty array claim")
	}
}

func TestEvaluateMatchRules_ObjectClaim_NotUsefullyMatchable(t *testing.T) {
	// A JSON object claim value at the leaf of a match path (as opposed
	// to being an intermediate node navigated via dot-notation, which
	// resolveClaimPath handles separately) stringifies the same way:
	// Go's map representation, not JSON.
	claims := map[string]interface{}{"metadata": map[string]interface{}{"team": "platform"}}
	rules := []MatchRule{{Field: "metadata", Value: "platform"}}

	passed, failure := EvaluateMatchRules(rules, claims)
	if passed {
		t.Error("expected failure: object claims don't match nested field values directly")
	}
	if failure.ActualValue != "map[team:platform]" {
		t.Errorf("actual = %q, want %q (documents Go fmt stringification of map[string]interface{})", failure.ActualValue, "map[team:platform]")
	}
}

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

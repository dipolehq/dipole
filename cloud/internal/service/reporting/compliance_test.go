package reporting

import (
	"testing"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

func TestCompliance_PolicyWithoutTP_Violation(t *testing.T) {
	policy := domain.AccessPolicy{
		ID:                    uuid.New(),
		Name:                  "No TP Policy",
		IsActive:              true,
		TrustProviderIDs:      []uuid.UUID{},
		CredentialProviderIDs: []domain.PolicyCredentialProvider{{CredentialProviderID: uuid.New()}},
		AccessConditionIDs:    []uuid.UUID{uuid.New()},
	}

	rules := []ComplianceRule{
		{Type: "trust_provider_required", Enabled: true},
	}

	result := evaluatePolicy(policy, rules)
	if result.Compliant {
		t.Error("policy without TP should be non-compliant")
	}
	if len(result.Violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(result.Violations))
	}
}

func TestCompliance_PolicyWithTP_Compliant(t *testing.T) {
	policy := domain.AccessPolicy{
		ID:                    uuid.New(),
		Name:                  "Good Policy",
		IsActive:              true,
		TrustProviderIDs:      []uuid.UUID{uuid.New()},
		CredentialProviderIDs: []domain.PolicyCredentialProvider{{CredentialProviderID: uuid.New()}},
		AccessConditionIDs:    []uuid.UUID{uuid.New()},
	}

	rules := []ComplianceRule{
		{Type: "trust_provider_required", Enabled: true},
		{Type: "credential_provider_required", Enabled: true},
		{Type: "access_condition_required", Enabled: true},
	}

	result := evaluatePolicy(policy, rules)
	if !result.Compliant {
		t.Errorf("policy with all components should be compliant, violations: %v", result.Violations)
	}
}

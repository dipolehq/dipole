package service

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

func TestResourceSet_DefaultCannotBeDeleted(t *testing.T) {
	svc := &ResourceSetService{}
	err := svc.Delete(nil, uuid.New(), DefaultResourceSetID)
	if err == nil {
		t.Fatal("expected error when deleting default Resource Set")
	}
}

func TestResourceSet_ResolveDefault(t *testing.T) {
	id := ResolveResourceSetID(nil)
	if id != DefaultResourceSetID {
		t.Errorf("expected default RS ID, got %s", id)
	}
}

func TestResourceSet_ResolveProvided(t *testing.T) {
	custom := uuid.New()
	id := ResolveResourceSetID(&custom)
	if id != custom {
		t.Errorf("expected provided RS ID %s, got %s", custom, id)
	}
}

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

func TestRBAC_ReadOnlyBlocksWrite(t *testing.T) {
	svc := NewRBACService()
	perms := ReadOnlyPermissions()

	err := svc.CheckPermission(perms, "client_workloads", "POST")
	if err != ErrForbidden {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestRBAC_ReadOnlyAllowsRead(t *testing.T) {
	svc := NewRBACService()
	perms := ReadOnlyPermissions()

	err := svc.CheckPermission(perms, "client_workloads", "GET")
	if err != nil {
		t.Fatalf("expected nil (allowed), got %v", err)
	}
}

func TestRBAC_AdminAllowsWrite(t *testing.T) {
	svc := NewRBACService()
	perms := AdminPermissions()

	err := svc.CheckPermission(perms, "access_policies", "POST")
	if err != nil {
		t.Fatalf("expected nil (allowed), got %v", err)
	}
}

func TestRBAC_AdminBlocksRoleWrite(t *testing.T) {
	svc := NewRBACService()
	perms := AdminPermissions()

	err := svc.CheckPermission(perms, "roles", "PUT")
	if err != ErrForbidden {
		t.Fatalf("expected ErrForbidden for Admin writing roles, got %v", err)
	}
}

func TestSignOnPolicy_SSORequired_BlocksNonAdmin(t *testing.T) {
	svc := NewSignOnPolicyService()
	policy := &SignOnPolicy{RequireSSO: true}

	err := svc.CheckNativeLoginAllowed(policy, false)
	if err != ErrSSORequired {
		t.Fatalf("expected ErrSSORequired, got %v", err)
	}
}

func TestSignOnPolicy_SSORequired_AllowsSuperAdmin(t *testing.T) {
	svc := NewSignOnPolicyService()
	policy := &SignOnPolicy{RequireSSO: true}

	err := svc.CheckNativeLoginAllowed(policy, true)
	if err != nil {
		t.Fatalf("expected nil (Super Admin allowed), got %v", err)
	}
}

func TestSignOnPolicy_MFA_WithinGracePeriod_Allows(t *testing.T) {
	svc := NewSignOnPolicyService()
	policy := &SignOnPolicy{RequireMFA: true, MFAGracePeriodHours: 24}
	promptedAt := time.Now().UTC().Add(-1 * time.Hour)

	err := svc.CheckMFARequired(policy, nil, &promptedAt)
	if err != nil {
		t.Fatalf("expected nil (within grace period), got %v", err)
	}
}

func TestSignOnPolicy_MFA_GraceExpired_Blocks(t *testing.T) {
	svc := NewSignOnPolicyService()
	policy := &SignOnPolicy{RequireMFA: true, MFAGracePeriodHours: 24}
	promptedAt := time.Now().UTC().Add(-48 * time.Hour)

	err := svc.CheckMFARequired(policy, nil, &promptedAt)
	if err != ErrMFARequired {
		t.Fatalf("expected ErrMFARequired (grace expired), got %v", err)
	}
}

func TestSignOnPolicy_MFA_Configured_Allows(t *testing.T) {
	svc := NewSignOnPolicyService()
	policy := &SignOnPolicy{RequireMFA: true}
	configuredAt := time.Now().UTC().Add(-72 * time.Hour)

	err := svc.CheckMFARequired(policy, &configuredAt, nil)
	if err != nil {
		t.Fatalf("expected nil (MFA configured), got %v", err)
	}
}

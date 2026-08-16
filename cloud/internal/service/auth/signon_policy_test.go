package auth

import (
	"testing"
	"time"
)

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

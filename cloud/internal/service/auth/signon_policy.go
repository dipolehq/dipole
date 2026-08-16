// signon_policy.go enforces Sign-On Policy rules for admin login: require_sso and require_mfa checks.
package auth

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrSSORequired = errors.New("SSO required: native login disabled by Sign-On Policy")
	ErrMFARequired = errors.New("MFA required: provide verification code to complete login")
)

type SignOnPolicy struct {
	ID                  uuid.UUID
	TenantID            uuid.UUID
	RequireSSO          bool
	RequireMFA          bool
	MFAGracePeriodHours int
	AuthTimeoutMinutes  int
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type SignOnPolicyService struct{}

func NewSignOnPolicyService() *SignOnPolicyService {
	return &SignOnPolicyService{}
}

func (s *SignOnPolicyService) CheckNativeLoginAllowed(policy *SignOnPolicy, isSuperAdmin bool) error {
	if policy == nil {
		return nil
	}
	if policy.RequireSSO && !isSuperAdmin {
		return ErrSSORequired
	}
	return nil
}

func (s *SignOnPolicyService) CheckMFARequired(policy *SignOnPolicy, userMFAConfiguredAt *time.Time, userFirstPromptedAt *time.Time) error {
	if policy == nil || !policy.RequireMFA {
		return nil
	}

	// User already has MFA configured — no issue
	if userMFAConfiguredAt != nil {
		return nil
	}

	// MFA not configured yet — check if within grace period
	if userFirstPromptedAt != nil {
		gracePeriod := policy.MFAGracePeriodHours
		if gracePeriod == 0 {
			gracePeriod = 24
		}
		graceDeadline := userFirstPromptedAt.Add(time.Duration(gracePeriod) * time.Hour)
		if time.Now().UTC().Before(graceDeadline) {
			return nil // within grace period, allow login without MFA
		}
	}

	return ErrMFARequired
}

func DefaultSignOnPolicy(tenantID uuid.UUID) *SignOnPolicy {
	return &SignOnPolicy{
		ID:                  uuid.New(),
		TenantID:            tenantID,
		RequireSSO:          false,
		RequireMFA:          false,
		MFAGracePeriodHours: 24,
		AuthTimeoutMinutes:  480,
	}
}

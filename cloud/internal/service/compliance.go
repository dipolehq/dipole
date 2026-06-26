/*
compliance.go implements Global Policy Compliance checking.

ComplianceService evaluates Access Policies against org-wide compliance rules
(e.g., "all policies must have a Trust Provider"). Reports which policies
comply and which don't — advisory only, does not block policy creation.
*/
package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
)

type ComplianceRule struct {
	Type    string `json:"type"`
	Enabled bool   `json:"enabled"`
}

type GlobalPolicy struct {
	ID       uuid.UUID
	TenantID uuid.UUID
	Rules    []ComplianceRule
}

type PolicyComplianceResult struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	Compliant  bool      `json:"compliant"`
	Violations []string  `json:"violations"`
}

type ComplianceService struct {
	policyRepo repository.AccessPolicyRepository
}

func NewComplianceService(policyRepo repository.AccessPolicyRepository) *ComplianceService {
	return &ComplianceService{policyRepo: policyRepo}
}

func (s *ComplianceService) CheckCompliance(ctx context.Context, tenantID uuid.UUID, rules []ComplianceRule) ([]PolicyComplianceResult, error) {
	policies, _, err := s.policyRepo.List(ctx, tenantID, domain.ListOptions{Page: 1, PerPage: 1000})
	if err != nil {
		return nil, err
	}

	var results []PolicyComplianceResult
	for _, policy := range policies {
		if !policy.IsActive {
			continue
		}
		result := evaluatePolicy(policy, rules)
		results = append(results, result)
	}
	return results, nil
}

func evaluatePolicy(policy domain.AccessPolicy, rules []ComplianceRule) PolicyComplianceResult {
	result := PolicyComplianceResult{
		ID:         policy.ID,
		Name:       policy.Name,
		Compliant:  true,
		Violations: []string{},
	}

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		violation := checkRule(policy, rule)
		if violation != "" {
			result.Compliant = false
			result.Violations = append(result.Violations, violation)
		}
	}

	return result
}

func checkRule(policy domain.AccessPolicy, rule ComplianceRule) string {
	switch rule.Type {
	case "trust_provider_required":
		if len(policy.TrustProviderIDs) == 0 {
			return "trust_provider_required: policy has no Trust Provider"
		}
	case "access_condition_required":
		if len(policy.AccessConditionIDs) == 0 {
			return "access_condition_required: policy has no Access Condition"
		}
	case "credential_provider_required":
		if len(policy.CredentialProviderIDs) == 0 {
			return "credential_provider_required: policy has no Credential Provider"
		}
	}
	return ""
}

func ParseComplianceRules(raw json.RawMessage) ([]ComplianceRule, error) {
	var rules []ComplianceRule
	if err := json.Unmarshal(raw, &rules); err != nil {
		return nil, fmt.Errorf("parse compliance rules: %w", err)
	}
	return rules, nil
}

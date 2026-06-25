/*
kerberos_validator.go verifies Kerberos-attested workload identity.

The Agent Controller performs Kerberos authentication (it holds the keytab)
and signs the attested identity. Cloud validates the Controller's attestation
rather than raw Kerberos tickets directly.

KerberosValidator checks that Kerberos evidence contains a principal and
realm, then evaluates match rules against these fields. Only enabled when
DIPOLE_KERBEROS_ATTESTATION_ENABLED=true on the proxy.

Match rules: Principal, Realm/Domain, Source IP.
*/
package trust

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type KerberosValidator struct{}

type kerberosConfig struct {
	MatchRules []MatchRule `json:"match_rules"`
}

func (v *KerberosValidator) Validate(_ context.Context, provider *domain.TrustProvider, evidence *domain.ClientEvidence, _ *JWKSCache) (*domain.TrustProviderResult, error) {
	if evidence.Environment == nil {
		return &domain.TrustProviderResult{Attested: false, Reason: "NoDataFound"}, nil
	}

	principal := evidence.Environment["KERBEROS_PRINCIPAL"]
	realm := evidence.Environment["KERBEROS_REALM"]

	if principal == "" && realm == "" {
		return &domain.TrustProviderResult{Attested: false, Reason: "NoDataFound"}, nil
	}

	var cfg kerberosConfig
	if err := json.Unmarshal(provider.Config, &cfg); err != nil {
		return nil, fmt.Errorf("parse kerberos config: %w", err)
	}

	claims := map[string]interface{}{
		"principal": principal,
		"realm":     realm,
		"domain":    evidence.Environment["KERBEROS_DOMAIN"],
		"sourceIp":  evidence.SourceIP,
	}

	passed, failure := EvaluateMatchRules(cfg.MatchRules, claims)
	if !passed {
		return &domain.TrustProviderResult{
			Attested:      false,
			Reason:        "MatchRuleFailed",
			Attribute:     failure.Attribute,
			ExpectedValue: failure.ExpectedValue,
			ActualValue:   failure.ActualValue,
		}, nil
	}

	return &domain.TrustProviderResult{Attested: true}, nil
}

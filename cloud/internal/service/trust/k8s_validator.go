// k8s_validator.go validates Kubernetes service account JWTs against a configured OIDC endpoint.
package trust

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type K8sValidator struct{}

type k8sConfig struct {
	OIDCEndpoint string `json:"oidcEndpoint"`
}

func (v *K8sValidator) Validate(ctx context.Context, provider *domain.TrustProvider, evidence *domain.ClientEvidence, jwksCache *JWKSCache) (*domain.TrustProviderResult, error) {
	if evidence.K8s == nil || evidence.K8s.ServiceAccountToken == "" {
		return &domain.TrustProviderResult{Attested: false, Reason: "NoDataFound"}, nil
	}

	var cfg k8sConfig
	if err := json.Unmarshal(provider.Config, &cfg); err != nil {
		return nil, err
	}

	token := evidence.K8s.ServiceAccountToken

	claims, verr := verifyJWS(ctx, token, cachedKeySetResolver{cache: jwksCache, endpoint: cfg.OIDCEndpoint})
	if verr != nil {
		if verr.keySetUnavailable() {
			return &domain.TrustProviderResult{Attested: false, Reason: "NoDataFound"}, nil
		}
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidSignature"}, nil
	}

	iss, _ := claims["iss"].(string)
	expectedIssuer := strings.TrimRight(cfg.OIDCEndpoint, "/")
	if expectedIssuer == "" || strings.TrimRight(iss, "/") != expectedIssuer {
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidIssuer"}, nil
	}

	if provider.EdgeSDKAudience != "" {
		aud := fmt.Sprint(claims["aud"])
		if aud != provider.EdgeSDKAudience {
			return &domain.TrustProviderResult{Attested: false, Reason: "InvalidAudience"}, nil
		}
	}

	if expIsExpired(claims) {
		return &domain.TrustProviderResult{Attested: false, Reason: "TokenExpired"}, nil
	}

	if nbfNotYetValid(claims) {
		return &domain.TrustProviderResult{Attested: false, Reason: "TokenNotYetValid"}, nil
	}

	passed, failure := EvaluateMatchRules(toMatchRules(provider.MatchRules), claims)
	if !passed {
		return &domain.TrustProviderResult{
			Attested:      false,
			Reason:        "MatchRuleFailed",
			Attribute:     failure.Attribute,
			ExpectedValue: failure.ExpectedValue,
			ActualValue:   failure.ActualValue,
		}, nil
	}

	return &domain.TrustProviderResult{Attested: true, Claims: claims}, nil
}

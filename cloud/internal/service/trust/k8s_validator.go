/*
k8s_validator.go validates Kubernetes service account JWTs.

K8sValidator extracts the SA token from evidence, fetches JWKS from the
configured OIDC endpoint, verifies the JWT signature (RS256/ES256), checks
expiry, and evaluates match rules against token claims.
*/
package trust

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-jose/go-jose/v4"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type K8sValidator struct{}

type k8sConfig struct {
	OIDCEndpoint string      `json:"oidc_endpoint"`
	MatchRules   []MatchRule `json:"match_rules"`
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

	// Parse JWT without verification to get header
	jws, err := jose.ParseSigned(token, []jose.SignatureAlgorithm{jose.RS256, jose.ES256})
	if err != nil {
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidSignature"}, nil
	}

	// Fetch JWKS
	keySet, err := jwksCache.Get(ctx, cfg.OIDCEndpoint)
	if err != nil {
		return &domain.TrustProviderResult{Attested: false, Reason: "NoDataFound"}, nil
	}

	// Find key by kid
	kid := jws.Signatures[0].Header.KeyID
	keys := keySet.Key(kid)
	if len(keys) == 0 {
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidSignature"}, nil
	}

	// Verify signature
	payload, err := jws.Verify(keys[0])
	if err != nil {
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidSignature"}, nil
	}

	// Parse claims
	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidSignature"}, nil
	}

	// Check expiry
	if exp, ok := claims["exp"].(float64); ok {
		if int64(exp) < time.Now().Unix() {
			return &domain.TrustProviderResult{Attested: false, Reason: "TokenExpired"}, nil
		}
	}

	// Evaluate match rules
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

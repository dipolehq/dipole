// gcp_validator.go verifies GCP identity tokens from the Google metadata server via standard OIDC verification.
package trust

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type GCPValidator struct{}

type gcpConfig struct {
	OIDCEndpoint string `json:"oidcEndpoint"`
}

func (v *GCPValidator) Validate(ctx context.Context, provider *domain.TrustProvider, evidence *domain.ClientEvidence, jwksCache *JWKSCache) (*domain.TrustProviderResult, error) {
	if evidence.GCP == nil || evidence.GCP.IdentityToken == "" {
		return &domain.TrustProviderResult{Attested: false, Reason: "NoDataFound"}, nil
	}

	var cfg gcpConfig
	if err := json.Unmarshal(provider.Config, &cfg); err != nil {
		return nil, fmt.Errorf("parse gcp config: %w", err)
	}

	endpoint := cfg.OIDCEndpoint
	if endpoint == "" {
		endpoint = "https://accounts.google.com"
	}

	claims, verr := verifyJWS(ctx, evidence.GCP.IdentityToken, cachedKeySetResolver{cache: jwksCache, endpoint: endpoint})
	if verr != nil {
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidSignature"}, nil
	}

	iss, _ := claims["iss"].(string)
	expectedIssuer := strings.TrimRight(endpoint, "/")
	if expectedIssuer == "" || strings.TrimRight(iss, "/") != expectedIssuer {
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidIssuer"}, nil
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

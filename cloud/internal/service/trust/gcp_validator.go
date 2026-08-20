/*
gcp_validator.go verifies GCP identity tokens from the Google metadata server.

GCPValidator validates Google-signed identity tokens using standard OIDC
verification (Google's JWKS endpoint). This is effectively OIDC validation
with Google as the issuer. Evaluates match rules against token claims (email).
*/
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

	// Check issuer: same rationale and trailing-slash normalization as
	// K8sValidator/OIDCValidator's oidc_discovery path — binds the
	// verified signature to the specific endpoint this Trust Provider is
	// configured to trust (Google's own endpoint by default), rather
	// than any issuer whose JWKS happens to be reachable there.
	iss, _ := claims["iss"].(string)
	expectedIssuer := strings.TrimRight(endpoint, "/")
	if expectedIssuer == "" || strings.TrimRight(iss, "/") != expectedIssuer {
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidIssuer"}, nil
	}

	if expIsExpired(claims) {
		return &domain.TrustProviderResult{Attested: false, Reason: "TokenExpired"}, nil
	}

	// Check not-before: reject tokens that aren't valid yet (RFC 7519 §4.1.5)
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

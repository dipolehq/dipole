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

	// Check issuer: the token must have been issued by the OIDC endpoint
	// this Trust Provider is configured to trust. Without this check, a
	// signature-valid JWT from ANY issuer whose JWKS happens to be
	// reachable at cfg.OIDCEndpoint (or whose kid collides) would pass —
	// this is the check that binds the verified signature to the
	// specific K8s cluster the customer configured.
	//
	// Trailing slashes are trimmed from both sides before comparing:
	// cfg.OIDCEndpoint is raw customer input (fetchJWKSURI already
	// tolerates a trailing slash there), while the iss claim emitted by
	// the actual issuer normally has none. Without normalizing, a
	// customer who configured "https://cluster.example.com/" would see
	// every legitimate token rejected as InvalidIssuer. An empty
	// cfg.OIDCEndpoint is rejected explicitly so a misconfigured Trust
	// Provider (empty expected issuer) can never match a token that's
	// simply missing its iss claim (both would otherwise trim to "").
	iss, _ := claims["iss"].(string)
	expectedIssuer := strings.TrimRight(cfg.OIDCEndpoint, "/")
	if expectedIssuer == "" || strings.TrimRight(iss, "/") != expectedIssuer {
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidIssuer"}, nil
	}

	// Check audience matches edge_sdk_audience, same as GitHubValidator and
	// OIDCValidator. Only enforced if configured, matching those validators'
	// existing behavior (see Part 1's Open Questions on whether this should
	// become mandatory across all three).
	if provider.EdgeSDKAudience != "" {
		aud := fmt.Sprint(claims["aud"])
		if aud != provider.EdgeSDKAudience {
			return &domain.TrustProviderResult{Attested: false, Reason: "InvalidAudience"}, nil
		}
	}

	// Check expiry
	if expIsExpired(claims) {
		return &domain.TrustProviderResult{Attested: false, Reason: "TokenExpired"}, nil
	}

	// Check not-before: reject tokens that aren't valid yet (RFC 7519 §4.1.5)
	if nbfNotYetValid(claims) {
		return &domain.TrustProviderResult{Attested: false, Reason: "TokenNotYetValid"}, nil
	}

	// Evaluate match rules
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

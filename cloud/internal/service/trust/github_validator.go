// github_validator.go validates GitHub Actions OIDC tokens against the hardcoded GitHub OIDC endpoint.
package trust

import (
	"context"
	"fmt"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

const githubOIDCEndpoint = "https://token.actions.githubusercontent.com"

type GitHubValidator struct{}

func (v *GitHubValidator) Validate(ctx context.Context, provider *domain.TrustProvider, evidence *domain.ClientEvidence, jwksCache *JWKSCache) (*domain.TrustProviderResult, error) {
	if evidence.GitHub == nil || evidence.GitHub.IdentityToken == "" {
		return &domain.TrustProviderResult{Attested: false, Reason: "NoDataFound"}, nil
	}

	token := evidence.GitHub.IdentityToken

	claims, verr := verifyJWS(ctx, token, cachedKeySetResolver{cache: jwksCache, endpoint: githubOIDCEndpoint})
	if verr != nil {
		if verr.keySetUnavailable() {
			return &domain.TrustProviderResult{Attested: false, Reason: "NoDataFound"}, nil
		}
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidSignature"}, nil
	}

	if iss, _ := claims["iss"].(string); iss != githubOIDCEndpoint {
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

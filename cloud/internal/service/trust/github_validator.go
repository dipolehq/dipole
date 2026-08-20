/*
github_validator.go validates GitHub Actions OIDC tokens.

GitHubValidator uses the hardcoded GitHub OIDC endpoint to fetch JWKS, verifies
the JWT signature, checks audience matches the Trust Provider's edge_sdk_audience,
checks expiry, and evaluates match rules (repository, actor, workflow).
*/
package trust

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-jose/go-jose/v4"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

const githubOIDCEndpoint = "https://token.actions.githubusercontent.com"

type GitHubValidator struct{}

func (v *GitHubValidator) Validate(ctx context.Context, provider *domain.TrustProvider, evidence *domain.ClientEvidence, jwksCache *JWKSCache) (*domain.TrustProviderResult, error) {
	if evidence.GitHub == nil || evidence.GitHub.IdentityToken == "" {
		return &domain.TrustProviderResult{Attested: false, Reason: "NoDataFound"}, nil
	}

	token := evidence.GitHub.IdentityToken

	jws, err := jose.ParseSigned(token, []jose.SignatureAlgorithm{jose.RS256, jose.ES256})
	if err != nil {
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidSignature"}, nil
	}

	keySet, err := jwksCache.Get(ctx, githubOIDCEndpoint)
	if err != nil {
		return &domain.TrustProviderResult{Attested: false, Reason: "NoDataFound"}, nil
	}

	kid := jws.Signatures[0].Header.KeyID
	keys := keySet.Key(kid)
	if len(keys) == 0 {
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidSignature"}, nil
	}

	payload, err := jws.Verify(keys[0])
	if err != nil {
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidSignature"}, nil
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidSignature"}, nil
	}

	// Check issuer: GitHub Actions tokens are always issued by
	// githubOIDCEndpoint, so a mismatch here means the signature was
	// verified against the wrong key material (e.g. a kid collision) or
	// the token was never a genuine GitHub Actions token to begin with.
	if iss, _ := claims["iss"].(string); iss != githubOIDCEndpoint {
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidIssuer"}, nil
	}

	// Check audience matches edge_sdk_audience
	if provider.EdgeSDKAudience != "" {
		aud := fmt.Sprint(claims["aud"])
		if aud != provider.EdgeSDKAudience {
			return &domain.TrustProviderResult{Attested: false, Reason: "InvalidAudience"}, nil
		}
	}

	// Check expiry
	if exp, ok := claims["exp"].(float64); ok {
		if int64(exp) < time.Now().Unix() {
			return &domain.TrustProviderResult{Attested: false, Reason: "TokenExpired"}, nil
		}
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

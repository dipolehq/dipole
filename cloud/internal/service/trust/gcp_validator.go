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
	"time"

	"github.com/go-jose/go-jose/v4"

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

	claims, err := verifyOIDCToken(ctx, evidence.GCP.IdentityToken, endpoint, jwksCache)
	if err != nil {
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidSignature"}, nil
	}

	if exp, ok := claims["exp"].(float64); ok {
		if int64(exp) < time.Now().Unix() {
			return &domain.TrustProviderResult{Attested: false, Reason: "TokenExpired"}, nil
		}
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

	return &domain.TrustProviderResult{Attested: true}, nil
}

func verifyOIDCToken(ctx context.Context, token, oidcEndpoint string, jwksCache *JWKSCache) (map[string]interface{}, error) {
	jws, err := jose.ParseSigned(token, []jose.SignatureAlgorithm{jose.RS256, jose.ES256})
	if err != nil {
		return nil, err
	}

	keySet, err := jwksCache.Get(ctx, oidcEndpoint)
	if err != nil {
		return nil, err
	}

	kid := jws.Signatures[0].Header.KeyID
	keys := keySet.Key(kid)
	if len(keys) == 0 {
		return nil, fmt.Errorf("key not found: %s", kid)
	}

	payload, err := jws.Verify(keys[0])
	if err != nil {
		return nil, err
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, err
	}
	return claims, nil
}

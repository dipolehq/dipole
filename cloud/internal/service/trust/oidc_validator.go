/*
oidc_validator.go is the generic OIDC validator covering GitLab, Terraform Cloud,
and any standards-compliant OIDC provider.

OIDCValidator supports three attestation methods: oidc_discovery (fetch JWKS from
endpoint), upload_jwks (JWKS stored in config), symmetric_key (HMAC-SHA256).
Used for gitlab_job, terraform_cloud, and oidc_id_token trust provider types.
*/
package trust

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-jose/go-jose/v4"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type OIDCValidator struct{}

type oidcConfig struct {
	AttestationMethod string      `json:"attestation_method"`
	OIDCEndpoint      string      `json:"oidc_endpoint"`
	JWKS              string      `json:"jwks"`
	SymmetricKey      string      `json:"symmetric_key"`
	MatchRules        []MatchRule `json:"match_rules"`
}

func (v *OIDCValidator) Validate(ctx context.Context, provider *domain.TrustProvider, evidence *domain.ClientEvidence, jwksCache *JWKSCache) (*domain.TrustProviderResult, error) {
	token := extractOIDCToken(provider.Type, evidence)
	if token == "" {
		return &domain.TrustProviderResult{Attested: false, Reason: "NoDataFound"}, nil
	}

	var cfg oidcConfig
	if err := json.Unmarshal(provider.Config, &cfg); err != nil {
		return nil, err
	}

	var claims map[string]interface{}
	var err error

	switch cfg.AttestationMethod {
	case "oidc_discovery", "":
		claims, err = v.verifyWithDiscovery(ctx, token, cfg.OIDCEndpoint, jwksCache)
	case "upload_jwks":
		claims, err = v.verifyWithUploadedJWKS(token, cfg.JWKS)
	case "symmetric_key":
		claims, err = v.verifyWithSymmetricKey(token, cfg.SymmetricKey)
	default:
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidSignature"}, nil
	}

	if err != nil {
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidSignature"}, nil
	}

	// Check audience if configured
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

func (v *OIDCValidator) verifyWithDiscovery(ctx context.Context, token, oidcEndpoint string, jwksCache *JWKSCache) (map[string]interface{}, error) {
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

func (v *OIDCValidator) verifyWithUploadedJWKS(token, jwksJSON string) (map[string]interface{}, error) {
	var keySet jose.JSONWebKeySet
	if err := json.Unmarshal([]byte(jwksJSON), &keySet); err != nil {
		return nil, err
	}

	jws, err := jose.ParseSigned(token, []jose.SignatureAlgorithm{jose.RS256, jose.ES256})
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

func (v *OIDCValidator) verifyWithSymmetricKey(token, keyBase64 string) (map[string]interface{}, error) {
	key, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		return nil, fmt.Errorf("decode symmetric key: %w", err)
	}

	// Split JWT into parts
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format")
	}

	// Verify HMAC-SHA256
	signingInput := parts[0] + "." + parts[1]
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("decode signature: %w", err)
	}

	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(signingInput))
	expected := mac.Sum(nil)

	if !hmac.Equal(sig, expected) {
		return nil, fmt.Errorf("signature mismatch")
	}

	// Decode payload
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, err
	}
	return claims, nil
}

func extractOIDCToken(providerType string, evidence *domain.ClientEvidence) string {
	switch providerType {
	case "gitlab_job":
		if evidence.GitLab != nil {
			return evidence.GitLab.IdentityToken
		}
	case "terraform_cloud":
		if evidence.Terraform != nil {
			return evidence.Terraform.IdentityToken
		}
	case "oidc_id_token":
		// Generic OIDC — check multiple sources
		if evidence.GitHub != nil && evidence.GitHub.IdentityToken != "" {
			return evidence.GitHub.IdentityToken
		}
		if evidence.GitLab != nil && evidence.GitLab.IdentityToken != "" {
			return evidence.GitLab.IdentityToken
		}
		if evidence.Terraform != nil && evidence.Terraform.IdentityToken != "" {
			return evidence.Terraform.IdentityToken
		}
		if evidence.Environment != nil && evidence.Environment["OIDC_ID_TOKEN"] != "" {
			return evidence.Environment["OIDC_ID_TOKEN"]
		}
	}
	return ""
}

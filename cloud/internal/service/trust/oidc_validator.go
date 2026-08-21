// oidc_validator.go is the generic OIDC validator (oidc_discovery, upload_jwks, symmetric_key) covering GitLab, Terraform Cloud, and any standards-compliant OIDC provider.
package trust

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dipolehq/dipole/cloud/internal/crypto"
	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type OIDCValidator struct {
	encryptionKey []byte
}

func NewOIDCValidator(encryptionKey []byte) *OIDCValidator {
	return &OIDCValidator{encryptionKey: encryptionKey}
}

type oidcConfig struct {
	AttestationMethod string `json:"attestationMethod"`
	OIDCEndpoint      string `json:"oidcEndpoint"`
	JWKS              string `json:"jwks"`
	PublicKey         string `json:"publicKey"`
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
		var verr *jwsVerifyError
		claims, verr = verifyJWS(ctx, token, cachedKeySetResolver{cache: jwksCache, endpoint: cfg.OIDCEndpoint})
		if verr != nil {
			err = verr
		}
	case "upload_jwks":
		var verr *jwsVerifyError
		claims, verr = verifyJWS(ctx, token, staticKeySetResolver{jwksJSON: cfg.JWKS})
		if verr != nil {
			err = verr
		}
	case "symmetric_key":
		var symmetricKey string
		symmetricKey, err = v.decryptSymmetricKey(provider)
		if err == nil {
			claims, err = v.verifyWithSymmetricKey(token, symmetricKey)
		}
	default:
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidSignature"}, nil
	}

	if err != nil {
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidSignature"}, nil
	}

	if cfg.AttestationMethod == "oidc_discovery" || cfg.AttestationMethod == "" {
		iss, _ := claims["iss"].(string)
		expectedIssuer := strings.TrimRight(cfg.OIDCEndpoint, "/")
		if expectedIssuer == "" || strings.TrimRight(iss, "/") != expectedIssuer {
			return &domain.TrustProviderResult{Attested: false, Reason: "InvalidIssuer"}, nil
		}
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

func (v *OIDCValidator) decryptSymmetricKey(provider *domain.TrustProvider) (string, error) {
	if len(provider.SymmetricKeyEncrypted) == 0 {
		return "", fmt.Errorf("trust provider has no symmetric key configured")
	}
	plaintext, err := crypto.Decrypt(provider.SymmetricKeyEncrypted, v.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("decrypt symmetric key: %w", err)
	}
	return string(plaintext), nil
}

func (v *OIDCValidator) verifyWithSymmetricKey(token, keyBase64 string) (map[string]interface{}, error) {
	key, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		return nil, fmt.Errorf("decode symmetric key: %w", err)
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format")
	}

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

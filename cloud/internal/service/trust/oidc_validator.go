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

	"github.com/dipolehq/dipole/cloud/internal/crypto"
	"github.com/dipolehq/dipole/cloud/internal/domain"
)

// OIDCValidator needs the tenant encryption key because the symmetric_key
// attestation method's HMAC shared secret is stored encrypted
// (TrustProvider.SymmetricKeyEncrypted), not in the plaintext Config blob
// — the same reason KerberosValidator holds an encryption key for its
// AgentController shared secret.
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
		claims, err = v.verifyWithDiscovery(ctx, token, cfg.OIDCEndpoint, jwksCache)
	case "upload_jwks":
		claims, err = v.verifyWithUploadedJWKS(token, cfg.JWKS)
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

	// Check issuer for oidc_discovery: the token must have been issued by
	// the OIDC endpoint this Trust Provider trusts, not merely signed by
	// a key that happens to be reachable at that endpoint's JWKS. For
	// upload_jwks and symmetric_key there's no single "endpoint" the
	// customer configured to compare against — the trust anchor there is
	// the uploaded key material or shared secret itself, so issuer
	// enforcement for those methods is expressed via match rules instead.
	//
	// Trailing slashes are trimmed from both sides before comparing:
	// cfg.OIDCEndpoint is raw customer input (FetchJWKS already tolerates
	// a trailing slash there), while the iss claim emitted by the actual
	// issuer normally has none — without normalizing, a customer who
	// configured "https://gitlab.example.com/" would see every
	// legitimate token rejected as InvalidIssuer. An empty
	// cfg.OIDCEndpoint is rejected explicitly so a misconfigured Trust
	// Provider can never match a token that's simply missing its iss
	// claim (both would otherwise trim to "").
	if cfg.AttestationMethod == "oidc_discovery" || cfg.AttestationMethod == "" {
		iss, _ := claims["iss"].(string)
		expectedIssuer := strings.TrimRight(cfg.OIDCEndpoint, "/")
		if expectedIssuer == "" || strings.TrimRight(iss, "/") != expectedIssuer {
			return &domain.TrustProviderResult{Attested: false, Reason: "InvalidIssuer"}, nil
		}
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

// decryptSymmetricKey returns the base64-encoded HMAC shared secret for a
// symmetric_key-attestation trust provider, decrypting it from
// SymmetricKeyEncrypted. There is no legacy plaintext fallback: by the
// time a Trust Provider using this attestation method reaches Validate,
// the resources.TrustProviderService/repository backfill has already
// moved its secret out of Config and into this encrypted field.
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

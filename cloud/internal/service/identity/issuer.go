// issuer.go implements the OIDC Identity Issuance Service: signs and verifies JWTs (ES256/RS256), and serves discovery and JWKS.
package identity

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/google/uuid"
)

type IdentityIssuer struct {
	platformIssuerURL string

	es256Key *ecdsa.PrivateKey
	es256Kid string
	rs256Key *rsa.PrivateKey
	rs256Kid string
}

type IssuanceRequest struct {
	Subject      string
	Audience     string
	Lifetime     time.Duration
	Algorithm    string
	CustomClaims map[string]interface{}
	TrustClaims  map[string]interface{}
	IssuerURL    string
}

func NewIdentityIssuer(platformIssuerURL string) (*IdentityIssuer, error) {
	es256Key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate ES256 key: %w", err)
	}

	rs256Key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generate RS256 key: %w", err)
	}

	return &IdentityIssuer{
		platformIssuerURL: platformIssuerURL,
		es256Key:          es256Key,
		es256Kid:          uuid.New().String()[:8],
		rs256Key:          rs256Key,
		rs256Kid:          uuid.New().String()[:8],
	}, nil
}

func (i *IdentityIssuer) VerifyTokenSignature(token string) error {
	jws, err := jose.ParseSigned(token, []jose.SignatureAlgorithm{jose.ES256, jose.RS256})
	if err != nil {
		return fmt.Errorf("invalid JWT format")
	}

	alg := jws.Signatures[0].Header.Algorithm

	var pubKey interface{}
	switch alg {
	case string(jose.ES256):
		pubKey = i.es256Key.Public()
	case string(jose.RS256):
		pubKey = i.rs256Key.Public()
	default:
		return fmt.Errorf("unsupported algorithm: %s", alg)
	}

	if _, err := jws.Verify(pubKey); err != nil {
		return fmt.Errorf("%s signature verification failed", alg)
	}

	return nil
}

func (i *IdentityIssuer) IssueToken(req IssuanceRequest) (string, error) {
	subject, err := ResolveDynamicValue(req.Subject, req.TrustClaims)
	if err != nil {
		return "", fmt.Errorf("resolve subject: %w", err)
	}

	resolvedClaims := make(map[string]interface{})
	for k, v := range req.CustomClaims {
		if strVal, ok := v.(string); ok {
			resolved, err := ResolveDynamicValue(strVal, req.TrustClaims)
			if err != nil {
				return "", fmt.Errorf("resolve claim %q: %w", k, err)
			}
			resolvedClaims[k] = resolved
		} else {
			resolvedClaims[k] = v
		}
	}

	issuerURL := req.IssuerURL
	if issuerURL == "" {
		issuerURL = i.platformIssuerURL
	}

	now := time.Now()
	payload := map[string]interface{}{
		"iss": issuerURL,
		"sub": subject,
		"aud": req.Audience,
		"iat": now.Unix(),
		"nbf": now.Unix(),
		"exp": now.Add(req.Lifetime).Unix(),
		"jti": uuid.New().String(),
	}
	for k, v := range resolvedClaims {
		payload[k] = v
	}

	alg := req.Algorithm
	if alg == "" {
		alg = "ES256"
	}

	var key crypto.Signer
	var kid string
	switch alg {
	case "ES256":
		key = i.es256Key
		kid = i.es256Kid
	case "RS256":
		key = i.rs256Key
		kid = i.rs256Kid
	default:
		return "", fmt.Errorf("unsupported algorithm: %s", alg)
	}

	return signJWT(payload, key, alg, kid)
}

func (i *IdentityIssuer) DiscoveryDocument(issuerURL string) map[string]interface{} {
	if issuerURL == "" {
		issuerURL = i.platformIssuerURL
	}
	return map[string]interface{}{
		"issuer":                                issuerURL,
		"jwks_uri":                              issuerURL + "/.well-known/openid-configuration/jwks",
		"id_token_signing_alg_values_supported": []string{"ES256", "RS256"},
		"response_types_supported":              []string{"id_token"},
		"subject_types_supported":               []string{"public"},
	}
}

func (i *IdentityIssuer) JWKS() map[string]interface{} {
	keys := []map[string]interface{}{}

	ecPubBytes, _ := i.es256Key.PublicKey.Bytes()
	coordSize := (i.es256Key.PublicKey.Curve.Params().BitSize + 7) / 8
	xBytes := ecPubBytes[1 : 1+coordSize]
	yBytes := ecPubBytes[1+coordSize:]

	keys = append(keys, map[string]interface{}{
		"kty": "EC",
		"kid": i.es256Kid,
		"crv": "P-256",
		"x":   base64.RawURLEncoding.EncodeToString(xBytes),
		"y":   base64.RawURLEncoding.EncodeToString(yBytes),
		"use": "sig",
		"alg": "ES256",
	})

	keys = append(keys, map[string]interface{}{
		"kty": "RSA",
		"kid": i.rs256Kid,
		"n":   base64.RawURLEncoding.EncodeToString(i.rs256Key.PublicKey.N.Bytes()),
		"e":   base64URLEncodeInt(i.rs256Key.PublicKey.E),
		"use": "sig",
		"alg": "RS256",
	})

	return map[string]interface{}{"keys": keys}
}

func base64URLEncodeInt(n int) string {
	b := big.NewInt(int64(n)).Bytes()
	return base64.RawURLEncoding.EncodeToString(b)
}

func signJWT(payload map[string]interface{}, key crypto.Signer, alg, kid string) (string, error) {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	var sigAlg jose.SignatureAlgorithm
	switch alg {
	case "ES256":
		sigAlg = jose.ES256
	case "RS256":
		sigAlg = jose.RS256
	default:
		return "", fmt.Errorf("unsupported algorithm: %s", alg)
	}

	jwk := jose.JSONWebKey{Key: key, KeyID: kid, Algorithm: alg, Use: "sig"}
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: sigAlg, Key: jwk}, (&jose.SignerOptions{}).WithType("JWT"))
	if err != nil {
		return "", err
	}

	obj, err := signer.Sign(payloadJSON)
	if err != nil {
		return "", err
	}

	return obj.CompactSerialize()
}

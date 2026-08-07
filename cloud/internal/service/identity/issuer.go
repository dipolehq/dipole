/*
Package identity provides the OIDC Identity Issuance Service.

IdentityIssuer acts as an OIDC provider — signs JWTs with ES256 or RS256,
serves discovery and JWKS endpoints, and supports dynamic claim resolution.
Used by credential providers (AWS STS, Azure WIF, GCP WIF) that exchange
our OIDC tokens for native cloud credentials.

Issuer URL pattern: https://<tenantSubdomain>.id.<region>.dipolehq.com
*/
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
	"strings"
	"time"

	"github.com/google/uuid"
)

type IdentityIssuer struct {
	issuerURL string
	es256Key  *ecdsa.PrivateKey
	es256Kid  string
	rs256Key  *rsa.PrivateKey
	rs256Kid  string
}

type IssuanceRequest struct {
	Subject      string
	Audience     string
	Lifetime     time.Duration
	Algorithm    string
	CustomClaims map[string]interface{}
	TrustClaims  map[string]interface{}
}

func NewIdentityIssuer(tenantSubdomain, region string) (*IdentityIssuer, error) {
	es256Key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate ES256 key: %w", err)
	}

	rs256Key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generate RS256 key: %w", err)
	}

	return &IdentityIssuer{
		issuerURL: fmt.Sprintf("https://%s.id.dipolehq.com", tenantSubdomain),
		es256Key:  es256Key,
		es256Kid:  uuid.New().String()[:8],
		rs256Key:  rs256Key,
		rs256Kid:  uuid.New().String()[:8],
	}, nil
}

func (i *IdentityIssuer) IssuerURL() string {
	return i.issuerURL
}

// VerifyTokenSignature verifies a JWT was signed by this issuer's keys.
func (i *IdentityIssuer) VerifyTokenSignature(token string) error {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return fmt.Errorf("invalid JWT format")
	}

	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return fmt.Errorf("decode header: %w", err)
	}
	var header struct {
		Alg string `json:"alg"`
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return fmt.Errorf("parse header: %w", err)
	}

	signingInput := parts[0] + "." + parts[1]
	sigBytes, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}

	hash := crypto.SHA256.New()
	hash.Write([]byte(signingInput))
	digest := hash.Sum(nil)

	switch header.Alg {
	case "ES256":
		if !ecdsa.VerifyASN1(i.es256Key.Public().(*ecdsa.PublicKey), digest, ecdsaFixedToASN1(sigBytes, 32)) {
			return fmt.Errorf("ES256 signature verification failed")
		}
	case "RS256":
		if err := rsa.VerifyPKCS1v15(i.rs256Key.Public().(*rsa.PublicKey), crypto.SHA256, digest, sigBytes); err != nil {
			return fmt.Errorf("RS256 signature verification failed")
		}
	default:
		return fmt.Errorf("unsupported algorithm: %s", header.Alg)
	}

	return nil
}

// ecdsaFixedToASN1 converts a fixed-size ECDSA signature (r || s) to ASN.1 DER format.
func ecdsaFixedToASN1(sig []byte, keySize int) []byte {
	if len(sig) != keySize*2 {
		return sig
	}
	r := new(big.Int).SetBytes(sig[:keySize])
	s := new(big.Int).SetBytes(sig[keySize:])

	// ASN.1 DER: SEQUENCE { INTEGER r, INTEGER s }
	rBytes := r.Bytes()
	sBytes := s.Bytes()

	// Add leading zero if high bit set (positive integer encoding)
	if len(rBytes) > 0 && rBytes[0]&0x80 != 0 {
		rBytes = append([]byte{0}, rBytes...)
	}
	if len(sBytes) > 0 && sBytes[0]&0x80 != 0 {
		sBytes = append([]byte{0}, sBytes...)
	}

	// Build DER: 0x30 <len> 0x02 <rlen> <r> 0x02 <slen> <s>
	inner := make([]byte, 0, 2+len(rBytes)+2+len(sBytes))
	inner = append(inner, 0x02, byte(len(rBytes)))
	inner = append(inner, rBytes...)
	inner = append(inner, 0x02, byte(len(sBytes)))
	inner = append(inner, sBytes...)

	out := make([]byte, 0, 2+len(inner))
	out = append(out, 0x30, byte(len(inner)))
	out = append(out, inner...)
	return out
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

	now := time.Now()
	payload := map[string]interface{}{
		"iss": i.issuerURL,
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

func (i *IdentityIssuer) DiscoveryDocument() map[string]interface{} {
	return map[string]interface{}{
		"issuer":                                i.issuerURL,
		"jwks_uri":                              i.issuerURL + "/.well-known/openid-configuration/jwks",
		"token_endpoint":                        i.issuerURL + "/connect/token",
		"id_token_signing_alg_values_supported": []string{"ES256", "RS256"},
		"response_types_supported":              []string{"id_token"},
		"subject_types_supported":               []string{"public"},
	}
}

func (i *IdentityIssuer) JWKS() map[string]interface{} {
	keys := []map[string]interface{}{}

	// ES256 key — extract X, Y from uncompressed public key bytes (0x04 || X || Y)
	ecPubBytes := elliptic.Marshal(i.es256Key.PublicKey.Curve, i.es256Key.PublicKey.X, i.es256Key.PublicKey.Y)
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

	// RS256 key
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
	header := map[string]string{"alg": alg, "kid": kid, "typ": "JWT"}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signingInput := headerB64 + "." + payloadB64

	var signature []byte
	switch alg {
	case "ES256":
		hash := crypto.SHA256.New()
		hash.Write([]byte(signingInput))
		sig, err := ecdsa.SignASN1(rand.Reader, key.(*ecdsa.PrivateKey), hash.Sum(nil))
		if err != nil {
			return "", err
		}
		signature = ecdsaASN1ToFixed(sig, 32)
	case "RS256":
		hash := crypto.SHA256.New()
		hash.Write([]byte(signingInput))
		sig, err := rsa.SignPKCS1v15(rand.Reader, key.(*rsa.PrivateKey), crypto.SHA256, hash.Sum(nil))
		if err != nil {
			return "", err
		}
		signature = sig
	default:
		return "", fmt.Errorf("unsupported algorithm: %s", alg)
	}

	sigB64 := base64.RawURLEncoding.EncodeToString(signature)
	return signingInput + "." + sigB64, nil
}

// ecdsaASN1ToFixed converts ASN.1 DER-encoded ECDSA signature to fixed-size
// format (r || s, each padded to keySize bytes) as required by JWS.
func ecdsaASN1ToFixed(asn1Sig []byte, keySize int) []byte {
	// Parse ASN.1: SEQUENCE { INTEGER r, INTEGER s }
	r, s := parseECDSASignature(asn1Sig)
	out := make([]byte, keySize*2)
	rBytes := r.Bytes()
	sBytes := s.Bytes()
	copy(out[keySize-len(rBytes):keySize], rBytes)
	copy(out[2*keySize-len(sBytes):], sBytes)
	return out
}

func parseECDSASignature(sig []byte) (*big.Int, *big.Int) {
	// ASN.1 DER: 0x30 <len> 0x02 <rlen> <r> 0x02 <slen> <s>
	r := new(big.Int)
	s := new(big.Int)
	if len(sig) < 6 {
		return r, s
	}
	// Skip outer SEQUENCE tag and length
	pos := 2
	// Parse r INTEGER
	if sig[pos] != 0x02 {
		return r, s
	}
	pos++
	rLen := int(sig[pos])
	pos++
	r.SetBytes(sig[pos : pos+rLen])
	pos += rLen
	// Parse s INTEGER
	if sig[pos] != 0x02 {
		return r, s
	}
	pos++
	sLen := int(sig[pos])
	pos++
	s.SetBytes(sig[pos : pos+sLen])
	return r, s
}

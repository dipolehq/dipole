// certificate_validator.go verifies certificate chain-of-trust plus proof-of-possession (stateless, bounded-freshness, not nonce-based) over a signed timestamp.
package trust

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"strconv"
	"time"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type CertificateValidator struct{}

type certificateConfig struct {
	TrustAnchorPEM string `json:"trustAnchorPem"`
}

const certificateChallengePrefix = "dipole-certificate-challenge-v1\x00"

const certificateClockSkew = 180 * time.Second

const certificateMaxAge = 5 * time.Minute

func (v *CertificateValidator) Validate(_ context.Context, provider *domain.TrustProvider, evidence *domain.ClientEvidence, _ *JWKSCache) (*domain.TrustProviderResult, error) {
	attest := evidence.CertificateAttestation
	if attest == nil || attest.CertificatePEM == "" {
		return &domain.TrustProviderResult{Attested: false, Reason: "NoDataFound"}, nil
	}

	var cfg certificateConfig
	if err := json.Unmarshal(provider.Config, &cfg); err != nil {
		return nil, fmt.Errorf("parse certificate config: %w", err)
	}

	cert, err := parseCertificatePEM(attest.CertificatePEM)
	if err != nil {
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidSignature"}, nil
	}

	roots, err := buildTrustPool(cfg.TrustAnchorPEM)
	if err != nil {
		return nil, fmt.Errorf("parse trust anchor: %w", err)
	}

	_, err = cert.Verify(x509.VerifyOptions{
		Roots:     roots,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	})
	if err != nil {
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidSignature"}, nil
	}

	if err := verifyCertificateChallenge(cert, attest.Timestamp, attest.Signature); err != nil {
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidSignature"}, nil
	}

	if fresh, reason := certificateAttestationFresh(attest.Timestamp); !fresh {
		return &domain.TrustProviderResult{Attested: false, Reason: reason}, nil
	}

	subjectFields := map[string]interface{}{
		"subject.cn":           cert.Subject.CommonName,
		"subject.organization": firstOrEmpty(cert.Subject.Organization),
		"subject.ou":           firstOrEmpty(cert.Subject.OrganizationalUnit),
		"subject.country":      firstOrEmpty(cert.Subject.Country),
		"subject.locality":     firstOrEmpty(cert.Subject.Locality),
		"subject.province":     firstOrEmpty(cert.Subject.Province),
	}

	if len(provider.MatchRules) == 0 {
		return &domain.TrustProviderResult{Attested: true, Claims: subjectFields}, nil
	}

	passed, failure := EvaluateMatchRules(toMatchRules(provider.MatchRules), subjectFields)
	if !passed {
		return &domain.TrustProviderResult{
			Attested:      false,
			Reason:        "MatchRuleFailed",
			Attribute:     failure.Attribute,
			ExpectedValue: failure.ExpectedValue,
			ActualValue:   failure.ActualValue,
		}, nil
	}

	return &domain.TrustProviderResult{Attested: true, Claims: subjectFields}, nil
}

func parseCertificatePEM(pemStr string) (*x509.Certificate, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("no PEM block found")
	}
	return x509.ParseCertificate(block.Bytes)
}

func buildCertificateChallenge(certDER []byte, timestamp int64) []byte {
	certHash := sha256.Sum256(certDER)
	challenge := certificateChallengePrefix +
		hex.EncodeToString(certHash[:]) + "\x00" +
		strconv.FormatInt(timestamp, 10)
	return []byte(challenge)
}

func verifyCertificateChallenge(cert *x509.Certificate, timestamp int64, signature []byte) error {
	challenge := buildCertificateChallenge(cert.Raw, timestamp)
	digest := sha256.Sum256(challenge)

	switch pub := cert.PublicKey.(type) {
	case *rsa.PublicKey:
		return rsa.VerifyPKCS1v15(pub, crypto.SHA256, digest[:], signature)
	case *ecdsa.PublicKey:
		if !ecdsa.VerifyASN1(pub, digest[:], signature) {
			return fmt.Errorf("ecdsa signature verification failed")
		}
		return nil
	default:
		return fmt.Errorf("unsupported public key type %T", pub)
	}
}

func certificateAttestationFresh(timestamp int64) (ok bool, reason string) {
	if timestamp <= 0 {
		return false, "MissingTimestamp"
	}
	signedAt := time.Unix(timestamp, 0)
	now := time.Now().UTC()

	if signedAt.After(now.Add(certificateClockSkew)) {
		return false, "AttestationNotYetValid"
	}
	if signedAt.Add(certificateMaxAge).Before(now) {
		return false, "AttestationExpired"
	}
	return true, ""
}

func buildTrustPool(trustAnchorPEM string) (*x509.CertPool, error) {
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM([]byte(trustAnchorPEM)) {
		return nil, fmt.Errorf("failed to parse trust anchor PEM")
	}
	return pool, nil
}

func firstOrEmpty(s []string) string {
	if len(s) > 0 {
		return s[0]
	}
	return ""
}

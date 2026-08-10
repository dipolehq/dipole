/*
certificate_validator.go verifies Certificate Signed Attestation documents.

CertificateValidator verifies that a signed attestation document was signed
by a certificate that chains to a configured trust anchor. This Trust Provider
validates ONLY the cryptographic signature — confirming
that a trusted certificate signed the attestation document. Match rules on
subject fields are also supported for additional filtering.
*/
package trust

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type CertificateValidator struct{}

type certificateConfig struct {
	TrustAnchorPEM string `json:"trustAnchorPem"`
}

func (v *CertificateValidator) Validate(_ context.Context, provider *domain.TrustProvider, evidence *domain.ClientEvidence, _ *JWKSCache) (*domain.TrustProviderResult, error) {
	if evidence.Environment == nil {
		return &domain.TrustProviderResult{Attested: false, Reason: "NoDataFound"}, nil
	}

	certPEM := evidence.Environment["ATTESTATION_CERTIFICATE"]
	if certPEM == "" {
		return &domain.TrustProviderResult{Attested: false, Reason: "NoDataFound"}, nil
	}

	var cfg certificateConfig
	if err := json.Unmarshal(provider.Config, &cfg); err != nil {
		return nil, fmt.Errorf("parse certificate config: %w", err)
	}

	cert, err := parseCertificatePEM(certPEM)
	if err != nil {
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidSignature"}, nil
	}

	roots, err := buildTrustPool(cfg.TrustAnchorPEM)
	if err != nil {
		return nil, fmt.Errorf("parse trust anchor: %w", err)
	}

	_, err = cert.Verify(x509.VerifyOptions{Roots: roots})
	if err != nil {
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidSignature"}, nil
	}

	if len(provider.MatchRules) == 0 {
		return &domain.TrustProviderResult{Attested: true}, nil
	}

	subjectFields := map[string]interface{}{
		"subject.cn":           cert.Subject.CommonName,
		"subject.organization": firstOrEmpty(cert.Subject.Organization),
		"subject.ou":           firstOrEmpty(cert.Subject.OrganizationalUnit),
		"subject.country":      firstOrEmpty(cert.Subject.Country),
		"subject.locality":     firstOrEmpty(cert.Subject.Locality),
		"subject.province":     firstOrEmpty(cert.Subject.Province),
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

	return &domain.TrustProviderResult{Attested: true}, nil
}

func parseCertificatePEM(pemStr string) (*x509.Certificate, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("no PEM block found")
	}
	return x509.ParseCertificate(block.Bytes)
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

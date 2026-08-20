// aws_imds_validator.go verifies EC2 instance identity documents via customer-supplied AWS signing certificate.
package trust

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type AWSIMDSValidator struct{}

type awsIMDSConfig struct {
	Certificate string `json:"certificate"`
}

func (v *AWSIMDSValidator) Validate(_ context.Context, provider *domain.TrustProvider, evidence *domain.ClientEvidence, _ *JWKSCache) (*domain.TrustProviderResult, error) {
	if evidence.AWS == nil || evidence.AWS.InstanceIdentityDocument == "" || evidence.AWS.InstanceIdentitySignature == "" {
		return &domain.TrustProviderResult{Attested: false, Reason: "NoDataFound"}, nil
	}

	var cfg awsIMDSConfig
	if err := json.Unmarshal(provider.Config, &cfg); err != nil {
		return nil, fmt.Errorf("parse aws_imds config: %w", err)
	}

	docBytes, err := base64.StdEncoding.DecodeString(evidence.AWS.InstanceIdentityDocument)
	if err != nil {
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidSignature"}, nil
	}

	sigBytes, err := base64.StdEncoding.DecodeString(evidence.AWS.InstanceIdentitySignature)
	if err != nil {
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidSignature"}, nil
	}

	cert, err := parseCertificateFromPEM(cfg.Certificate)
	if err != nil {
		return nil, fmt.Errorf("parse aws certificate: %w", err)
	}

	pubKey, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("aws certificate does not contain an RSA public key")
	}

	now := time.Now()
	if now.Before(cert.NotBefore) {
		return &domain.TrustProviderResult{Attested: false, Reason: "CertificateNotYetValid"}, nil
	}
	if now.After(cert.NotAfter) {
		return &domain.TrustProviderResult{Attested: false, Reason: "CertificateExpired"}, nil
	}

	hash := sha256.Sum256(docBytes)
	if err := rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, hash[:], sigBytes); err != nil {
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidSignature"}, nil
	}

	var docFields map[string]interface{}
	if err := json.Unmarshal(docBytes, &docFields); err != nil {
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidSignature"}, nil
	}

	passed, failure := EvaluateMatchRules(toMatchRules(provider.MatchRules), docFields)
	if !passed {
		return &domain.TrustProviderResult{
			Attested:      false,
			Reason:        "MatchRuleFailed",
			Attribute:     failure.Attribute,
			ExpectedValue: failure.ExpectedValue,
			ActualValue:   failure.ActualValue,
		}, nil
	}

	return &domain.TrustProviderResult{Attested: true, Claims: docFields}, nil
}

func parseCertificateFromPEM(pemStr string) (*x509.Certificate, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("no PEM block found in certificate")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse x509 certificate: %w", err)
	}

	return cert, nil
}

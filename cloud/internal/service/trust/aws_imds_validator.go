/*
aws_imds_validator.go verifies EC2 instance identity documents.

AWSIMDSValidator base64-decodes the identity document and RSA signature from
evidence, verifies the signature using the AWS public certificate stored in
the Trust Provider config, then evaluates match rules against document fields
(accountId, instanceId, region, etc.).

The signing certificate is customer-supplied config (there is no single
global AWS public certificate to hardcode as a default trust anchor: AWS
publishes distinct certificates per verification method — PKCS7/DSA,
base64-encoded RSA, RSA-2048 — and per partition (standard, GovCloud,
China), documented at
https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/regions-certs.html.
Embedding an incomplete or outdated subset would silently break customers
in a region/partition not covered, which is worse than requiring upload.
What this validator DOES enforce is that the configured certificate is
currently within its own stated validity window (NotBefore/NotAfter) —
a customer using a rotated-out or not-yet-valid certificate is a
misconfiguration that should be reported clearly (CertificateExpired /
CertificateNotYetValid) rather than surfacing as an opaque signature
failure.
*/
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

	// A signature that verifies against an expired or not-yet-valid
	// certificate is not proof the document came from AWS's currently
	// operative signing key — reject with a specific, actionable reason
	// rather than letting this surface as an indistinguishable
	// InvalidSignature. Checked before signature verification since a
	// cert outside its validity window shouldn't be trusted regardless
	// of what it signs.
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

/*
aws_imds_validator.go verifies EC2 instance identity documents.

AWSIMDSValidator base64-decodes the identity document and RSA signature from
evidence, verifies the signature using the AWS public certificate stored in
the Trust Provider config, then evaluates match rules against document fields
(accountId, instanceId, region, etc.).
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

	pubKey, err := parseRSAPublicKeyFromPEM(cfg.Certificate)
	if err != nil {
		return nil, fmt.Errorf("parse aws certificate: %w", err)
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

	return &domain.TrustProviderResult{Attested: true}, nil
}

func parseRSAPublicKeyFromPEM(pemStr string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("no PEM block found in certificate")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse x509 certificate: %w", err)
	}

	rsaKey, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("certificate does not contain RSA public key")
	}

	return rsaKey, nil
}

/*
azure_validator.go verifies Azure instance identity via IMDS attested documents.

AzureValidator parses the PKCS#7 signed attestation document from Azure IMDS,
verifies the cryptographic signature against system roots (or an optional
pinned CA), extracts compute metadata from the verified payload, and evaluates
match rules against fields (subscriptionId, vmId, sku).
*/
package trust

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"go.mozilla.org/pkcs7"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type AzureValidator struct{}

type azureConfig struct {
	SigningCAPEM string `json:"signingCaPem"`
}

func (v *AzureValidator) Validate(_ context.Context, provider *domain.TrustProvider, evidence *domain.ClientEvidence, _ *JWKSCache) (*domain.TrustProviderResult, error) {
	if evidence.Azure == nil || evidence.Azure.Signature == "" {
		return &domain.TrustProviderResult{Attested: false, Reason: "NoDataFound"}, nil
	}

	var cfg azureConfig
	if err := json.Unmarshal(provider.Config, &cfg); err != nil {
		return nil, fmt.Errorf("parse azure config: %w", err)
	}

	computeFields, err := verifyAndExtractAzureAttestation(evidence.Azure.Signature, cfg.SigningCAPEM)
	if err != nil {
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidSignature"}, nil
	}

	if len(provider.MatchRules) == 0 {
		return &domain.TrustProviderResult{Attested: true, Claims: computeFields}, nil
	}

	passed, failure := EvaluateMatchRules(toMatchRules(provider.MatchRules), computeFields)
	if !passed {
		return &domain.TrustProviderResult{
			Attested:      false,
			Reason:        "MatchRuleFailed",
			Attribute:     failure.Attribute,
			ExpectedValue: failure.ExpectedValue,
			ActualValue:   failure.ActualValue,
		}, nil
	}

	return &domain.TrustProviderResult{Attested: true, Claims: computeFields}, nil
}

// verifyAndExtractAzureAttestation parses the PKCS#7 signed attestation,
// verifies the cryptographic signature, and returns the verified compute
// metadata payload.
func verifyAndExtractAzureAttestation(signatureBase64, signingCAPEM string) (map[string]interface{}, error) {
	sigBytes, err := base64.StdEncoding.DecodeString(signatureBase64)
	if err != nil {
		return nil, fmt.Errorf("decode signature base64: %w", err)
	}

	p7, err := pkcs7.Parse(sigBytes)
	if err != nil {
		return nil, fmt.Errorf("parse PKCS#7: %w", err)
	}

	if signingCAPEM != "" {
		certPool := x509.NewCertPool()
		if !certPool.AppendCertsFromPEM([]byte(signingCAPEM)) {
			return nil, fmt.Errorf("failed to parse signing CA PEM")
		}
		if err := p7.VerifyWithChain(certPool); err != nil {
			return nil, fmt.Errorf("PKCS#7 signature verification failed: %w", err)
		}
	} else {
		if err := p7.Verify(); err != nil {
			return nil, fmt.Errorf("PKCS#7 signature verification failed: %w", err)
		}
	}

	var fields map[string]interface{}
	if err := json.Unmarshal(p7.Content, &fields); err != nil {
		return nil, fmt.Errorf("parse verified compute metadata: %w", err)
	}

	return fields, nil
}

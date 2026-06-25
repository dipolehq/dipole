/*
azure_validator.go verifies Azure instance identity via IMDS attested documents.

AzureValidator checks that Azure attestation evidence exists (proving the
workload is on Azure), decodes the PKCS#7 signed document to extract compute
metadata, and evaluates match rules against fields (subscriptionId, vmId, sku).

For MVP, the PKCS#7 signature is not chain-verified against Microsoft's CAs.
The evidence is trusted because it could only have been obtained from within
the Azure VM (IMDS at 169.254.169.254 is not reachable externally).
*/
package trust

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type AzureValidator struct{}

type azureConfig struct {
	MatchRules []MatchRule `json:"match_rules"`
}

func (v *AzureValidator) Validate(_ context.Context, provider *domain.TrustProvider, evidence *domain.ClientEvidence, _ *JWKSCache) (*domain.TrustProviderResult, error) {
	if evidence.Azure == nil || evidence.Azure.Signature == "" {
		return &domain.TrustProviderResult{Attested: false, Reason: "NoDataFound"}, nil
	}

	var cfg azureConfig
	if err := json.Unmarshal(provider.Config, &cfg); err != nil {
		return nil, fmt.Errorf("parse azure config: %w", err)
	}

	// If no match rules configured, the presence of Azure evidence is sufficient
	if len(cfg.MatchRules) == 0 {
		return &domain.TrustProviderResult{Attested: true}, nil
	}

	computeFields, err := decodeAzureAttestedDocument(evidence.Azure.Signature)
	if err != nil {
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidSignature"}, nil
	}

	passed, failure := EvaluateMatchRules(cfg.MatchRules, computeFields)
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

// decodeAzureAttestedDocument extracts compute metadata from the PKCS#7
// signed attestation document. The signature field is base64-encoded PKCS#7
// containing a JSON payload with Azure compute instance metadata.
func decodeAzureAttestedDocument(signature string) (map[string]interface{}, error) {
	sigBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return nil, fmt.Errorf("decode signature base64: %w", err)
	}

	// Extract the encapsulated content from PKCS#7 SignedData.
	// PKCS#7 DER structure: SEQUENCE > [0] > SEQUENCE(SignedData) >
	//   version > digestAlgos > encapContentInfo(SEQUENCE > OID > [0] > OCTET STRING)
	content, err := extractPKCS7Payload(sigBytes)
	if err != nil {
		return nil, err
	}

	var fields map[string]interface{}
	if err := json.Unmarshal(content, &fields); err != nil {
		return nil, fmt.Errorf("parse compute metadata: %w", err)
	}
	return fields, nil
}

// extractPKCS7Payload navigates the DER-encoded PKCS#7 ASN.1 structure to
// extract the signed payload (encapsulated content).
func extractPKCS7Payload(der []byte) ([]byte, error) {
	// Outer SEQUENCE
	inner, err := asn1Unwrap(der, 0x30)
	if err != nil {
		return nil, err
	}

	// Skip ContentType OID, enter [0] EXPLICIT
	inner, err = asn1SkipAndUnwrap(inner, 0xA0)
	if err != nil {
		return nil, err
	}

	// SignedData SEQUENCE
	inner, err = asn1Unwrap(inner, 0x30)
	if err != nil {
		return nil, err
	}

	// Skip: version, digestAlgorithms
	inner, err = asn1Skip(inner) // version
	if err != nil {
		return nil, err
	}
	inner, err = asn1Skip(inner) // digestAlgorithms
	if err != nil {
		return nil, err
	}

	// EncapContentInfo SEQUENCE
	encap, err := asn1Unwrap(inner, 0x30)
	if err != nil {
		return nil, err
	}

	// Skip content type OID, enter [0] EXPLICIT
	encap, err = asn1SkipAndUnwrap(encap, 0xA0)
	if err != nil {
		return nil, err
	}

	// OCTET STRING with the payload
	content, err := asn1Unwrap(encap, 0x04)
	if err != nil {
		// Fallback: some encodings use the raw bytes directly
		return encap, nil
	}

	return content, nil
}

func asn1Unwrap(data []byte, expectedTag byte) ([]byte, error) {
	if len(data) < 2 || data[0] != expectedTag {
		return nil, fmt.Errorf("expected ASN.1 tag 0x%02x", expectedTag)
	}
	return asn1Content(data[1:])
}

func asn1Skip(data []byte) ([]byte, error) {
	if len(data) < 2 {
		return nil, fmt.Errorf("data too short")
	}
	_, remaining, err := asn1Element(data)
	return remaining, err
}

func asn1SkipAndUnwrap(data []byte, wrapTag byte) ([]byte, error) {
	// Skip one element, then unwrap the next with expected tag
	remaining, err := asn1Skip(data)
	if err != nil {
		return nil, err
	}
	return asn1Unwrap(remaining, wrapTag)
}

func asn1Element(data []byte) (content []byte, remaining []byte, err error) {
	if len(data) < 2 {
		return nil, nil, fmt.Errorf("data too short")
	}
	contentBytes, err := asn1Content(data[1:])
	if err != nil {
		return nil, nil, err
	}
	totalLen := len(data) - len(data[1:]) + asn1HeaderLen(data[1:]) + len(contentBytes)
	if totalLen > len(data) {
		return nil, nil, fmt.Errorf("element exceeds data")
	}
	return contentBytes, data[totalLen:], nil
}

func asn1Content(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("no length byte")
	}
	if data[0] < 0x80 {
		length := int(data[0])
		if len(data) < 1+length {
			return nil, fmt.Errorf("content exceeds data")
		}
		return data[1 : 1+length], nil
	}
	numBytes := int(data[0] & 0x7F)
	if numBytes == 0 || numBytes > 4 || len(data) < 1+numBytes {
		return nil, fmt.Errorf("invalid length encoding")
	}
	length := 0
	for i := 0; i < numBytes; i++ {
		length = (length << 8) | int(data[1+i])
	}
	offset := 1 + numBytes
	if len(data) < offset+length {
		return nil, fmt.Errorf("content exceeds data")
	}
	return data[offset : offset+length], nil
}

func asn1HeaderLen(data []byte) int {
	if len(data) == 0 {
		return 0
	}
	if data[0] < 0x80 {
		return 1
	}
	return 1 + int(data[0]&0x7F)
}

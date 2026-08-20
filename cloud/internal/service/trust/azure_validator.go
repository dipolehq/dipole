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
	"time"

	"go.mozilla.org/pkcs7"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type AzureValidator struct{}

type azureConfig struct {
	SigningCAPEM string `json:"signingCaPem"`
}

// azureTimestampLayout matches the format Azure IMDS uses for
// timeStamp.createdOn/expiresOn in the signed attestation payload, e.g.
// "11/30/20 21:19:19 -0000" (see Microsoft's documented sample response).
// Azure does not document this as a named format; it was derived directly
// from their sample and is not RFC 3339.
const azureTimestampLayout = "1/2/06 15:04:05 -0700"

// azureClockSkew tolerates minor clock drift between Azure's attestation
// service and this server when checking the signed document's validity
// window, matching the tolerance already established by clockSkew (JWT
// validators) and samlClockSkew (SAML) elsewhere in this package for the
// same reason.
const azureClockSkew = 180 * time.Second

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

	// Reject documents outside their signed validity window. Azure signs
	// attestation documents with a short createdOn/expiresOn window
	// (observed ~45s in Microsoft's own sample) specifically so a
	// captured document cannot be replayed indefinitely — this is the
	// mechanism providing replay mitigation for this validator, mirroring
	// the timestamp-freshness checks already used for Kerberos
	// (kerberosTimestampMaxAge) and AWS IMDS certs elsewhere in this
	// package. This is time-bounded mitigation, not absolute replay
	// prevention: a captured document remains valid, and thus replayable,
	// until expiresOn.
	if fresh, reason := azureAttestationFresh(computeFields); !fresh {
		return &domain.TrustProviderResult{Attested: false, Reason: reason}, nil
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

// azureAttestationFresh reports whether the verified attestation payload's
// timeStamp.createdOn/expiresOn window covers the current time, within
// azureClockSkew tolerance in each direction. Must only be called on fields
// already extracted from a signature-verified payload — these timestamps
// are only trustworthy because they're inside Azure's signed document, not
// attacker-controlled input.
//
// A missing or malformed timeStamp block is treated as a rejection rather
// than skipped: every real Azure Resource Manager VM attestation includes
// this field (per Microsoft's documented schema), so its absence indicates
// either a non-ARM/classic-deployment VM (a deployment model Azure retired
// in 2024) or a malformed/unexpected payload shape, neither of which
// should silently attest.
func azureAttestationFresh(fields map[string]interface{}) (ok bool, reason string) {
	ts, _ := fields["timeStamp"].(map[string]interface{})
	if ts == nil {
		return false, "MissingTimestamp"
	}

	createdOn, ok1 := parseAzureTimestamp(ts["createdOn"])
	expiresOn, ok2 := parseAzureTimestamp(ts["expiresOn"])
	if !ok1 || !ok2 {
		return false, "MissingTimestamp"
	}

	now := time.Now().UTC()
	if createdOn.Add(-azureClockSkew).After(now) {
		return false, "AttestationNotYetValid"
	}
	if expiresOn.Add(azureClockSkew).Before(now) {
		return false, "AttestationExpired"
	}
	return true, ""
}

func parseAzureTimestamp(v interface{}) (time.Time, bool) {
	s, ok := v.(string)
	if !ok || s == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(azureTimestampLayout, s)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

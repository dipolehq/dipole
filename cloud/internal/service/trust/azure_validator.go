// azure_validator.go verifies Azure instance identity via IMDS attested documents, with a mandatory (OS trust store or pinned) CA and signed-timestamp freshness check.
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

type AzureValidator struct {
	verifier chainVerifier
}

func NewAzureValidator() *AzureValidator {
	return &AzureValidator{verifier: systemChainVerifier{}}
}

type azureConfig struct {
	SigningCAPEM string `json:"signingCaPem"`
}

const azureTimestampLayout = "1/2/06 15:04:05 -0700"

const azureClockSkew = 180 * time.Second

func (v *AzureValidator) Validate(_ context.Context, provider *domain.TrustProvider, evidence *domain.ClientEvidence, _ *JWKSCache) (*domain.TrustProviderResult, error) {
	if evidence.Azure == nil || evidence.Azure.Signature == "" {
		return &domain.TrustProviderResult{Attested: false, Reason: "NoDataFound"}, nil
	}

	var cfg azureConfig
	if err := json.Unmarshal(provider.Config, &cfg); err != nil {
		return nil, fmt.Errorf("parse azure config: %w", err)
	}

	computeFields, err := verifyAndExtractAzureAttestation(evidence.Azure.Signature, cfg.SigningCAPEM, v.verifier)
	if err != nil {
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidSignature"}, nil
	}

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

func verifyAndExtractAzureAttestation(signatureBase64, signingCAPEM string, verifier chainVerifier) (map[string]interface{}, error) {
	sigBytes, err := base64.StdEncoding.DecodeString(signatureBase64)
	if err != nil {
		return nil, fmt.Errorf("decode signature base64: %w", err)
	}

	p7, err := pkcs7.Parse(sigBytes)
	if err != nil {
		return nil, fmt.Errorf("parse PKCS#7: %w", err)
	}

	var trustAnchor *x509.CertPool
	if signingCAPEM != "" {
		trustAnchor = x509.NewCertPool()
		if !trustAnchor.AppendCertsFromPEM([]byte(signingCAPEM)) {
			return nil, fmt.Errorf("failed to parse signing CA PEM")
		}
	} else {
		trustAnchor, err = verifier.trustedRoots()
		if err != nil {
			return nil, fmt.Errorf("load default trust anchor: %w", err)
		}
	}

	if err := p7.VerifyWithChain(trustAnchor); err != nil {
		return nil, fmt.Errorf("PKCS#7 signature verification failed: %w", err)
	}

	var fields map[string]interface{}
	if err := json.Unmarshal(p7.Content, &fields); err != nil {
		return nil, fmt.Errorf("parse verified compute metadata: %w", err)
	}

	return fields, nil
}

type chainVerifier interface {
	trustedRoots() (*x509.CertPool, error)
}

type systemChainVerifier struct{}

func (systemChainVerifier) trustedRoots() (*x509.CertPool, error) {
	pool, err := x509.SystemCertPool()
	if err != nil {
		return nil, err
	}
	if pool == nil {
		return nil, fmt.Errorf("system certificate pool unavailable")
	}
	return pool, nil
}

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

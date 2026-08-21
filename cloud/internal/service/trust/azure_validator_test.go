/*
azure_validator_test.go tests Azure IMDS PKCS#7 signature verification.
*/
package trust

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"testing"
	"time"

	"go.mozilla.org/pkcs7"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

// freshAzureTimeStamp returns a `timeStamp` JSON fragment (Azure's
// "1/2/06 15:04:05 -0700"-style format) whose createdOn/expiresOn window
// covers the current moment, for use in test payloads that must pass the
// freshness check.
func freshAzureTimeStamp() string {
	now := time.Now().UTC()
	created := now.Add(-10 * time.Second).Format(azureTimestampLayout)
	expires := now.Add(1 * time.Minute).Format(azureTimestampLayout)
	return `"timeStamp":{"createdOn":"` + created + `","expiresOn":"` + expires + `"}`
}

func TestAzureValidator_ValidPKCS7_Attested(t *testing.T) {
	caKey, caCert, caPEM := generateTestCA(t)
	payload := `{"subscriptionId":"sub-123","vmId":"vm-456","sku":"Standard_D2s_v3",` + freshAzureTimeStamp() + `}`
	sig := signPKCS7(t, []byte(payload), caKey, caCert)
	sigBase64 := base64.StdEncoding.EncodeToString(sig)

	provider := &domain.TrustProvider{
		Config:     json.RawMessage(`{"signingCaPem":"` + escapePEM(caPEM) + `"}`),
		MatchRules: []domain.MatchRule{{Attribute: "subscriptionId", Value: "sub-123"}},
	}
	evidence := &domain.ClientEvidence{
		Azure: &domain.AzureEvidence{Signature: sigBase64},
	}

	v := &AzureValidator{verifier: systemChainVerifier{}}
	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Attested {
		t.Errorf("expected attested=true, got reason: %s", result.Reason)
	}
}

func TestAzureValidator_InvalidSignature_NotAttested(t *testing.T) {
	provider := &domain.TrustProvider{
		Config: json.RawMessage(`{}`),
	}
	evidence := &domain.ClientEvidence{
		Azure: &domain.AzureEvidence{Signature: base64.StdEncoding.EncodeToString([]byte("not valid pkcs7"))},
	}

	v := &AzureValidator{verifier: systemChainVerifier{}}
	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Attested {
		t.Error("expected attested=false for invalid PKCS#7")
	}
	if result.Reason != "InvalidSignature" {
		t.Errorf("expected reason 'InvalidSignature', got '%s'", result.Reason)
	}
}

func TestAzureValidator_TamperedPayload_NotAttested(t *testing.T) {
	caKey, caCert, caPEM := generateTestCA(t)
	payload := `{"subscriptionId":"sub-123","vmId":"vm-456"}`
	sig := signPKCS7(t, []byte(payload), caKey, caCert)

	// Tamper with the signature bytes (flip a byte in the middle)
	sig[len(sig)/2] ^= 0xFF

	sigBase64 := base64.StdEncoding.EncodeToString(sig)

	provider := &domain.TrustProvider{
		Config: json.RawMessage(`{"signingCaPem":"` + escapePEM(caPEM) + `"}`),
	}
	evidence := &domain.ClientEvidence{
		Azure: &domain.AzureEvidence{Signature: sigBase64},
	}

	v := &AzureValidator{verifier: systemChainVerifier{}}
	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Attested {
		t.Error("expected attested=false for tampered PKCS#7")
	}
}

func TestAzureValidator_MatchRuleFails_NotAttested(t *testing.T) {
	caKey, caCert, caPEM := generateTestCA(t)
	payload := `{"subscriptionId":"sub-999","vmId":"vm-456",` + freshAzureTimeStamp() + `}`
	sig := signPKCS7(t, []byte(payload), caKey, caCert)
	sigBase64 := base64.StdEncoding.EncodeToString(sig)

	provider := &domain.TrustProvider{
		Config:     json.RawMessage(`{"signingCaPem":"` + escapePEM(caPEM) + `"}`),
		MatchRules: []domain.MatchRule{{Attribute: "subscriptionId", Value: "sub-123"}},
	}
	evidence := &domain.ClientEvidence{
		Azure: &domain.AzureEvidence{Signature: sigBase64},
	}

	v := &AzureValidator{verifier: systemChainVerifier{}}
	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Attested {
		t.Error("expected attested=false when match rule fails")
	}
	if result.Reason != "MatchRuleFailed" {
		t.Errorf("expected reason 'MatchRuleFailed', got '%s'", result.Reason)
	}
	if result.Attribute != "subscriptionId" {
		t.Errorf("expected attribute 'subscriptionId', got '%s'", result.Attribute)
	}
}

func TestAzureValidator_NoEvidence_NotAttested(t *testing.T) {
	provider := &domain.TrustProvider{
		Config: json.RawMessage(`{}`),
	}
	evidence := &domain.ClientEvidence{}

	v := &AzureValidator{verifier: systemChainVerifier{}}
	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Attested {
		t.Error("expected attested=false when no Azure evidence")
	}
	if result.Reason != "NoDataFound" {
		t.Errorf("expected reason 'NoDataFound', got '%s'", result.Reason)
	}
}

// --- Test helpers ---

func generateTestCA(t *testing.T) (*ecdsa.PrivateKey, *x509.Certificate, string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Test Azure CA"},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		t.Fatal(err)
	}

	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	return key, cert, string(pemBytes)
}

func signPKCS7(t *testing.T, content []byte, key *ecdsa.PrivateKey, cert *x509.Certificate) []byte {
	t.Helper()
	signedData, err := pkcs7.NewSignedData(content)
	if err != nil {
		t.Fatal(err)
	}
	if err := signedData.AddSigner(cert, key, pkcs7.SignerInfoConfig{}); err != nil {
		t.Fatal(err)
	}
	der, err := signedData.Finish()
	if err != nil {
		t.Fatal(err)
	}
	return der
}

func escapePEM(pemStr string) string {
	result := ""
	for _, c := range pemStr {
		if c == '\n' {
			result += "\\n"
		} else if c == '"' {
			result += "\\\""
		} else {
			result += string(c)
		}
	}
	return result
}

func TestAzureValidator_ExpiredAttestation_NotAttested(t *testing.T) {
	caKey, caCert, caPEM := generateTestCA(t)
	now := time.Now().UTC()
	// A document whose validity window closed 10 minutes ago — simulates
	// a captured/replayed attestation being presented well after Azure's
	// short signing window expired.
	created := now.Add(-15 * time.Minute).Format(azureTimestampLayout)
	expires := now.Add(-10 * time.Minute).Format(azureTimestampLayout)
	payload := `{"subscriptionId":"sub-123","vmId":"vm-456","timeStamp":{"createdOn":"` + created + `","expiresOn":"` + expires + `"}}`
	sig := signPKCS7(t, []byte(payload), caKey, caCert)
	sigBase64 := base64.StdEncoding.EncodeToString(sig)

	provider := &domain.TrustProvider{
		Config: json.RawMessage(`{"signingCaPem":"` + escapePEM(caPEM) + `"}`),
	}
	evidence := &domain.ClientEvidence{
		Azure: &domain.AzureEvidence{Signature: sigBase64},
	}

	v := &AzureValidator{verifier: systemChainVerifier{}}
	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Attested {
		t.Error("expected attested=false for expired attestation window")
	}
	if result.Reason != "AttestationExpired" {
		t.Errorf("expected reason 'AttestationExpired', got '%s'", result.Reason)
	}
}

func TestAzureValidator_NotYetValidAttestation_NotAttested(t *testing.T) {
	caKey, caCert, caPEM := generateTestCA(t)
	now := time.Now().UTC()
	// A document whose validity window doesn't start for another 10
	// minutes — clock-skewed or forged createdOn far in the future.
	created := now.Add(10 * time.Minute).Format(azureTimestampLayout)
	expires := now.Add(15 * time.Minute).Format(azureTimestampLayout)
	payload := `{"subscriptionId":"sub-123","vmId":"vm-456","timeStamp":{"createdOn":"` + created + `","expiresOn":"` + expires + `"}}`
	sig := signPKCS7(t, []byte(payload), caKey, caCert)
	sigBase64 := base64.StdEncoding.EncodeToString(sig)

	provider := &domain.TrustProvider{
		Config: json.RawMessage(`{"signingCaPem":"` + escapePEM(caPEM) + `"}`),
	}
	evidence := &domain.ClientEvidence{
		Azure: &domain.AzureEvidence{Signature: sigBase64},
	}

	v := &AzureValidator{verifier: systemChainVerifier{}}
	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Attested {
		t.Error("expected attested=false for not-yet-valid attestation window")
	}
	if result.Reason != "AttestationNotYetValid" {
		t.Errorf("expected reason 'AttestationNotYetValid', got '%s'", result.Reason)
	}
}

func TestAzureValidator_MissingTimestamp_NotAttested(t *testing.T) {
	// A validly signed document that omits the timeStamp block entirely
	// (e.g. a classic/non-ARM deployment, or an unexpected payload
	// shape) must not attest — there's no basis to judge freshness.
	caKey, caCert, caPEM := generateTestCA(t)
	payload := `{"subscriptionId":"sub-123","vmId":"vm-456"}`
	sig := signPKCS7(t, []byte(payload), caKey, caCert)
	sigBase64 := base64.StdEncoding.EncodeToString(sig)

	provider := &domain.TrustProvider{
		Config: json.RawMessage(`{"signingCaPem":"` + escapePEM(caPEM) + `"}`),
	}
	evidence := &domain.ClientEvidence{
		Azure: &domain.AzureEvidence{Signature: sigBase64},
	}

	v := &AzureValidator{verifier: systemChainVerifier{}}
	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Attested {
		t.Error("expected attested=false when timeStamp block is missing")
	}
	if result.Reason != "MissingTimestamp" {
		t.Errorf("expected reason 'MissingTimestamp', got '%s'", result.Reason)
	}
}

// fakeSystemChainVerifier lets a test substitute a fixed pool of "system"
// roots, instead of depending on whatever CA bundle happens to be
// installed on the machine actually running the test.
type fakeSystemChainVerifier struct {
	pool *x509.CertPool
	err  error
}

func (f fakeSystemChainVerifier) trustedRoots() (*x509.CertPool, error) {
	return f.pool, f.err
}

func TestAzureValidator_NoCAConfigured_SelfSignedRejected(t *testing.T) {
	// Regression test for the MEDIUM finding this fix closes: a Trust
	// Provider configured WITHOUT signingCaPem must no longer accept an
	// arbitrary self-signed PKCS#7 document. Before this fix,
	// verifyAndExtractAzureAttestation fell back to pkcs7.Verify(),
	// which per that library's own doc comment "initializes an empty
	// trust store, effectively disabling certificate verification" -
	// meaning anyone could self-sign a document with a throwaway
	// certificate (exactly what generateTestCA below produces, chosen
	// specifically because it is NOT a certificate any real trust store
	// would contain) and attest successfully.
	//
	// The verifier here simulates a real deployment's system trust
	// store (which contains real public roots, none of which signed
	// this attacker's throwaway CA) rather than an empty one - proving
	// this closes the gap via the actual mandatory-default code path,
	// not merely via an artificially empty pool.
	caKey, caCert, _ := generateTestCA(t)
	payload := `{"subscriptionId":"sub-123","vmId":"vm-456",` + freshAzureTimeStamp() + `}`
	sig := signPKCS7(t, []byte(payload), caKey, caCert)
	sigBase64 := base64.StdEncoding.EncodeToString(sig)

	// A trust pool standing in for "the real system CA bundle" -
	// populated with a different CA than the one that signed the
	// attestation above, so it genuinely exercises "attacker's CA is
	// not in the trust store" rather than an empty-pool edge case.
	_, unrelatedCert, _ := generateTestCA(t)
	systemPool := x509.NewCertPool()
	systemPool.AddCert(unrelatedCert)

	provider := &domain.TrustProvider{
		// No signingCaPem configured - this is the exact scenario the
		// old code silently disabled verification for.
		Config: json.RawMessage(`{}`),
	}
	evidence := &domain.ClientEvidence{
		Azure: &domain.AzureEvidence{Signature: sigBase64},
	}

	v := &AzureValidator{verifier: fakeSystemChainVerifier{pool: systemPool}}
	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Attested {
		t.Error("expected attested=false: a self-signed PKCS#7 document must not verify against the default (system) trust store when it isn't the CA that signed it")
	}
	if result.Reason != "InvalidSignature" {
		t.Errorf("expected reason 'InvalidSignature', got '%s'", result.Reason)
	}
}

func TestAzureValidator_NoCAConfigured_SystemPoolUnavailable_Rejected(t *testing.T) {
	// If the default trust anchor can't be loaded at all (e.g.
	// x509.SystemCertPool() failing, or returning a nil pool on a
	// platform Go doesn't support for this), the validator must fail
	// closed rather than silently falling back to no verification.
	caKey, caCert, _ := generateTestCA(t)
	payload := `{"subscriptionId":"sub-123","vmId":"vm-456",` + freshAzureTimeStamp() + `}`
	sig := signPKCS7(t, []byte(payload), caKey, caCert)
	sigBase64 := base64.StdEncoding.EncodeToString(sig)

	provider := &domain.TrustProvider{
		Config: json.RawMessage(`{}`),
	}
	evidence := &domain.ClientEvidence{
		Azure: &domain.AzureEvidence{Signature: sigBase64},
	}

	v := &AzureValidator{verifier: fakeSystemChainVerifier{err: fmt.Errorf("system pool unavailable")}}
	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Attested {
		t.Error("expected attested=false when the default trust anchor cannot be loaded")
	}
	if result.Reason != "InvalidSignature" {
		t.Errorf("expected reason 'InvalidSignature', got '%s'", result.Reason)
	}
}

func TestAzureValidator_NoCAConfigured_ValidAgainstSystemPool_Attested(t *testing.T) {
	// The positive case: when no signingCaPem is configured, a document
	// signed by a CA that genuinely IS in the (simulated) system trust
	// store must still attest. Proves the default-trust-anchor path is
	// not merely fail-closed but actually functions as a real
	// verification path, not a permanently-broken one.
	caKey, caCert, _ := generateTestCA(t)
	payload := `{"subscriptionId":"sub-123","vmId":"vm-456",` + freshAzureTimeStamp() + `}`
	sig := signPKCS7(t, []byte(payload), caKey, caCert)
	sigBase64 := base64.StdEncoding.EncodeToString(sig)

	systemPool := x509.NewCertPool()
	systemPool.AddCert(caCert)

	provider := &domain.TrustProvider{
		Config: json.RawMessage(`{}`),
	}
	evidence := &domain.ClientEvidence{
		Azure: &domain.AzureEvidence{Signature: sigBase64},
	}

	v := &AzureValidator{verifier: fakeSystemChainVerifier{pool: systemPool}}
	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Attested {
		t.Errorf("expected attested=true when signer chains to the default trust store, got reason: %s", result.Reason)
	}
}

func TestAzureValidator_SigningCAPEM_OverridesSystemPool(t *testing.T) {
	// When signingCaPem IS configured, it must be used instead of the
	// default trust store - proving explicit configuration still
	// overrides the new mandatory default, per the documented design
	// (override, not merely an additional check).
	caKey, caCert, caPEM := generateTestCA(t)
	payload := `{"subscriptionId":"sub-123","vmId":"vm-456",` + freshAzureTimeStamp() + `}`
	sig := signPKCS7(t, []byte(payload), caKey, caCert)
	sigBase64 := base64.StdEncoding.EncodeToString(sig)

	// A "system" pool that does NOT contain this CA - if the validator
	// incorrectly consulted the system pool instead of signingCaPem,
	// this would fail; it must succeed because signingCaPem overrides.
	_, unrelatedCert, _ := generateTestCA(t)
	systemPool := x509.NewCertPool()
	systemPool.AddCert(unrelatedCert)

	provider := &domain.TrustProvider{
		Config: json.RawMessage(`{"signingCaPem":"` + escapePEM(caPEM) + `"}`),
	}
	evidence := &domain.ClientEvidence{
		Azure: &domain.AzureEvidence{Signature: sigBase64},
	}

	v := &AzureValidator{verifier: fakeSystemChainVerifier{pool: systemPool}}
	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Attested {
		t.Errorf("expected attested=true: signingCaPem should override the (non-matching) default trust store, got reason: %s", result.Reason)
	}
}

func TestAzureValidator_WithinClockSkew_Attested(t *testing.T) {
	// A document whose expiresOn is slightly in the past, but still
	// within azureClockSkew tolerance, must still attest — otherwise
	// minor clock drift between Azure and this server would cause
	// spurious rejections for legitimate, freshly-issued documents.
	caKey, caCert, caPEM := generateTestCA(t)
	now := time.Now().UTC()
	created := now.Add(-1 * time.Minute).Format(azureTimestampLayout)
	expires := now.Add(-30 * time.Second).Format(azureTimestampLayout) // within 180s skew
	payload := `{"subscriptionId":"sub-123","vmId":"vm-456","timeStamp":{"createdOn":"` + created + `","expiresOn":"` + expires + `"}}`
	sig := signPKCS7(t, []byte(payload), caKey, caCert)
	sigBase64 := base64.StdEncoding.EncodeToString(sig)

	provider := &domain.TrustProvider{
		Config: json.RawMessage(`{"signingCaPem":"` + escapePEM(caPEM) + `"}`),
	}
	evidence := &domain.ClientEvidence{
		Azure: &domain.AzureEvidence{Signature: sigBase64},
	}

	v := &AzureValidator{verifier: systemChainVerifier{}}
	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Attested {
		t.Errorf("expected attested=true within clock skew tolerance, got reason: %s", result.Reason)
	}
}

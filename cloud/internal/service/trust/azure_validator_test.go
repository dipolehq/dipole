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
	"math/big"
	"testing"
	"time"

	"go.mozilla.org/pkcs7"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

func TestAzureValidator_ValidPKCS7_Attested(t *testing.T) {
	caKey, caCert, caPEM := generateTestCA(t)
	payload := `{"subscriptionId":"sub-123","vmId":"vm-456","sku":"Standard_D2s_v3"}`
	sig := signPKCS7(t, []byte(payload), caKey, caCert)
	sigBase64 := base64.StdEncoding.EncodeToString(sig)

	provider := &domain.TrustProvider{
		Config:     json.RawMessage(`{"signingCaPem":"` + escapePEM(caPEM) + `"}`),
		MatchRules: []domain.MatchRule{{Attribute: "subscriptionId", Value: "sub-123"}},
	}
	evidence := &domain.ClientEvidence{
		Azure: &domain.AzureEvidence{Signature: sigBase64},
	}

	v := &AzureValidator{}
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

	v := &AzureValidator{}
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

	v := &AzureValidator{}
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
	payload := `{"subscriptionId":"sub-999","vmId":"vm-456"}`
	sig := signPKCS7(t, []byte(payload), caKey, caCert)
	sigBase64 := base64.StdEncoding.EncodeToString(sig)

	provider := &domain.TrustProvider{
		Config:     json.RawMessage(`{"signingCaPem":"` + escapePEM(caPEM) + `"}`),
		MatchRules: []domain.MatchRule{{Attribute: "subscriptionId", Value: "sub-123"}},
	}
	evidence := &domain.ClientEvidence{
		Azure: &domain.AzureEvidence{Signature: sigBase64},
	}

	v := &AzureValidator{}
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

	v := &AzureValidator{}
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

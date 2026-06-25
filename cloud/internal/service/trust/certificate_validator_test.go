package trust

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

func generateCACert(t *testing.T) (*ecdsa.PrivateKey, string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Test CA"},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	return key, string(certPEM)
}

func generateLeafCert(t *testing.T, caKey *ecdsa.PrivateKey, caPEM string, cn string) string {
	t.Helper()

	block, _ := pem.Decode([]byte(caPEM))
	caCert, _ := x509.ParseCertificate(block.Bytes)

	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: cn, Organization: []string{"TestOrg"}},
		NotBefore:    time.Now().Add(-1 * time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, caCert, &leafKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}

	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER}))
}

func TestCertificateValidator_ValidChain_Attested(t *testing.T) {
	caKey, caPEM := generateCACert(t)
	leafPEM := generateLeafCert(t, caKey, caPEM, "my-workload")

	cfg, _ := json.Marshal(certificateConfig{
		TrustAnchorPEM: caPEM,
		MatchRules:     []MatchRule{{Field: "subject.cn", Value: "my-workload"}},
	})

	provider := &domain.TrustProvider{ID: uuid.New(), Config: cfg}
	evidence := &domain.ClientEvidence{
		Environment: map[string]string{"ATTESTATION_CERTIFICATE": leafPEM},
	}

	v := &CertificateValidator{}
	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Attested {
		t.Fatalf("expected attested, got reason: %s", result.Reason)
	}
}

func TestCertificateValidator_WrongCA_InvalidSignature(t *testing.T) {
	caKey, caPEM := generateCACert(t)
	_, otherCAPEM := generateCACert(t)
	leafPEM := generateLeafCert(t, caKey, caPEM, "my-workload")

	cfg, _ := json.Marshal(certificateConfig{TrustAnchorPEM: otherCAPEM})
	provider := &domain.TrustProvider{ID: uuid.New(), Config: cfg}
	evidence := &domain.ClientEvidence{
		Environment: map[string]string{"ATTESTATION_CERTIFICATE": leafPEM},
	}

	v := &CertificateValidator{}
	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Attested {
		t.Fatal("expected denied")
	}
	if result.Reason != "InvalidSignature" {
		t.Fatalf("expected InvalidSignature, got %s", result.Reason)
	}
}

func TestCertificateValidator_MatchRuleFailed_WrongCN(t *testing.T) {
	caKey, caPEM := generateCACert(t)
	leafPEM := generateLeafCert(t, caKey, caPEM, "other-workload")

	cfg, _ := json.Marshal(certificateConfig{
		TrustAnchorPEM: caPEM,
		MatchRules:     []MatchRule{{Field: "subject.cn", Value: "my-workload"}},
	})

	provider := &domain.TrustProvider{ID: uuid.New(), Config: cfg}
	evidence := &domain.ClientEvidence{
		Environment: map[string]string{"ATTESTATION_CERTIFICATE": leafPEM},
	}

	v := &CertificateValidator{}
	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Attested {
		t.Fatal("expected denied")
	}
	if result.Reason != "MatchRuleFailed" {
		t.Fatalf("expected MatchRuleFailed, got %s", result.Reason)
	}
	if result.Attribute != "subject.cn" {
		t.Fatalf("expected attribute subject.cn, got %s", result.Attribute)
	}
}

func TestCertificateValidator_NoEvidence_NoDataFound(t *testing.T) {
	v := &CertificateValidator{}
	result, err := v.Validate(context.Background(), &domain.TrustProvider{}, &domain.ClientEvidence{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Reason != "NoDataFound" {
		t.Fatalf("expected NoDataFound, got %s", result.Reason)
	}
}

func TestCertificateValidator_NoMatchRules_ChainOnly(t *testing.T) {
	caKey, caPEM := generateCACert(t)
	leafPEM := generateLeafCert(t, caKey, caPEM, "anything")

	cfg, _ := json.Marshal(certificateConfig{TrustAnchorPEM: caPEM})
	provider := &domain.TrustProvider{ID: uuid.New(), Config: cfg}
	evidence := &domain.ClientEvidence{
		Environment: map[string]string{"ATTESTATION_CERTIFICATE": leafPEM},
	}

	v := &CertificateValidator{}
	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Attested {
		t.Fatalf("expected attested with no match rules, got reason: %s", result.Reason)
	}
}

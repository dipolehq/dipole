package trust

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

func generateAWSTestCert(t *testing.T) (*rsa.PrivateKey, string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "AWS Test Cert"},
		NotBefore:    time.Now().Add(-1 * time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	return key, string(certPEM)
}

func signDocument(t *testing.T, key *rsa.PrivateKey, doc []byte) string {
	t.Helper()
	hash := sha256.Sum256(doc)
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hash[:])
	if err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(sig)
}

func TestAWSIMDSValidator_ValidSignature_Attested(t *testing.T) {
	key, certPEM := generateAWSTestCert(t)

	doc := []byte(`{"accountId":"123456789012","region":"ap-south-1","instanceId":"i-0abc"}`)
	docB64 := base64.StdEncoding.EncodeToString(doc)
	sigB64 := signDocument(t, key, doc)

	cfg, _ := json.Marshal(awsIMDSConfig{
		Certificate: certPEM,
	})

	provider := &domain.TrustProvider{
		ID:         uuid.New(),
		Config:     cfg,
		MatchRules: []domain.MatchRule{{Attribute: "accountId", Value: "123456789012"}},
	}
	evidence := &domain.ClientEvidence{
		AWS: &domain.AWSEvidence{
			InstanceIdentityDocument:  docB64,
			InstanceIdentitySignature: sigB64,
		},
	}

	v := &AWSIMDSValidator{}
	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Attested {
		t.Fatalf("expected attested, got reason: %s", result.Reason)
	}
}

func TestAWSIMDSValidator_TamperedDocument_InvalidSignature(t *testing.T) {
	key, certPEM := generateAWSTestCert(t)

	doc := []byte(`{"accountId":"123456789012","region":"ap-south-1"}`)
	sigB64 := signDocument(t, key, doc)

	// Tamper the document
	tampered := []byte(`{"accountId":"999999999999","region":"ap-south-1"}`)
	docB64 := base64.StdEncoding.EncodeToString(tampered)

	cfg, _ := json.Marshal(awsIMDSConfig{Certificate: certPEM})
	provider := &domain.TrustProvider{ID: uuid.New(), Config: cfg}
	evidence := &domain.ClientEvidence{
		AWS: &domain.AWSEvidence{
			InstanceIdentityDocument:  docB64,
			InstanceIdentitySignature: sigB64,
		},
	}

	v := &AWSIMDSValidator{}
	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Attested {
		t.Fatal("expected denied, got attested")
	}
	if result.Reason != "InvalidSignature" {
		t.Fatalf("expected InvalidSignature, got %s", result.Reason)
	}
}

func TestAWSIMDSValidator_MatchRuleFailed(t *testing.T) {
	key, certPEM := generateAWSTestCert(t)

	doc := []byte(`{"accountId":"123456789012","region":"us-east-1"}`)
	docB64 := base64.StdEncoding.EncodeToString(doc)
	sigB64 := signDocument(t, key, doc)

	cfg, _ := json.Marshal(awsIMDSConfig{
		Certificate: certPEM,
	})

	provider := &domain.TrustProvider{
		ID:         uuid.New(),
		Config:     cfg,
		MatchRules: []domain.MatchRule{{Attribute: "accountId", Value: "999999999999"}},
	}
	evidence := &domain.ClientEvidence{
		AWS: &domain.AWSEvidence{
			InstanceIdentityDocument:  docB64,
			InstanceIdentitySignature: sigB64,
		},
	}

	v := &AWSIMDSValidator{}
	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Attested {
		t.Fatal("expected denied, got attested")
	}
	if result.Reason != "MatchRuleFailed" {
		t.Fatalf("expected MatchRuleFailed, got %s", result.Reason)
	}
	if result.Attribute != "accountId" {
		t.Fatalf("expected attribute accountId, got %s", result.Attribute)
	}
	if result.ExpectedValue != "999999999999" {
		t.Fatalf("expected value 999999999999, got %s", result.ExpectedValue)
	}
	if result.ActualValue != "123456789012" {
		t.Fatalf("expected actual 123456789012, got %s", result.ActualValue)
	}
}

func TestAWSIMDSValidator_NoEvidence_NoDataFound(t *testing.T) {
	v := &AWSIMDSValidator{}
	result, err := v.Validate(context.Background(), &domain.TrustProvider{}, &domain.ClientEvidence{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Reason != "NoDataFound" {
		t.Fatalf("expected NoDataFound, got %s", result.Reason)
	}
}

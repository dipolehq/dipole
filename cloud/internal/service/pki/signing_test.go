// signing_test.go tests CSR validation and leaf certificate issuance in isolation, independent of the gRPC transport layer.
package pki

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"testing"
	"time"

	"github.com/google/uuid"
)

func newSignableService(t *testing.T) (*TenantCAService, uuid.UUID) {
	t.Helper()
	encKey := make([]byte, 32)
	if _, err := rand.Read(encKey); err != nil {
		t.Fatalf("generate encryption key: %v", err)
	}

	repo := &mockTenantCARepo{}
	svc := NewTenantCAService(repo, encKey)

	tenantID := uuid.New()
	if _, _, err := svc.GetOrCreateTenantCA(context.Background(), tenantID); err != nil {
		t.Fatalf("seed tenant CA: %v", err)
	}
	return svc, tenantID
}

func buildCSR(t *testing.T, sans ...string) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	der, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{DNSNames: sans}, key)
	if err != nil {
		t.Fatalf("create CSR: %v", err)
	}
	return der
}

func TestSignCSR_HappyPath(t *testing.T) {
	svc, tenantID := newSignableService(t)

	resp, err := svc.SignCSR(context.Background(), SignCSRRequest{
		TenantID: tenantID,
		CSRDER:   buildCSR(t, "api.stripe.com"),
		Hostname: "api.stripe.com",
	})
	if err != nil {
		t.Fatalf("SignCSR: %v", err)
	}

	leaf, err := x509.ParseCertificate(resp.CertDER)
	if err != nil {
		t.Fatalf("parse leaf cert: %v", err)
	}
	if leaf.IsCA {
		t.Error("leaf certificate must not be a CA")
	}
	if len(leaf.DNSNames) != 1 || leaf.DNSNames[0] != "api.stripe.com" {
		t.Errorf("DNSNames = %v, want [api.stripe.com]", leaf.DNSNames)
	}
	if leaf.KeyUsage&x509.KeyUsageDigitalSignature == 0 {
		t.Error("missing KeyUsageDigitalSignature")
	}
	if len(leaf.ExtKeyUsage) != 1 || leaf.ExtKeyUsage[0] != x509.ExtKeyUsageServerAuth {
		t.Errorf("ExtKeyUsage = %v, want [ServerAuth]", leaf.ExtKeyUsage)
	}
	if resp.NotAfter.Before(time.Now()) {
		t.Error("NotAfter is in the past")
	}
}

func TestSignCSR_RejectsWildcardHostname(t *testing.T) {
	svc, tenantID := newSignableService(t)

	_, err := svc.SignCSR(context.Background(), SignCSRRequest{
		TenantID: tenantID,
		CSRDER:   buildCSR(t, "*.stripe.com"),
		Hostname: "*.stripe.com",
	})
	if err == nil {
		t.Fatal("expected error for wildcard hostname")
	}
}

func TestSignCSR_RejectsWildcardSAN(t *testing.T) {
	svc, tenantID := newSignableService(t)

	_, err := svc.SignCSR(context.Background(), SignCSRRequest{
		TenantID: tenantID,
		CSRDER:   buildCSR(t, "api.stripe.com", "*.stripe.com"),
		Hostname: "api.stripe.com",
	})
	if err == nil {
		t.Fatal("expected error for wildcard SAN in CSR")
	}
}

func TestSignCSR_RejectsMismatchedSAN(t *testing.T) {
	svc, tenantID := newSignableService(t)

	_, err := svc.SignCSR(context.Background(), SignCSRRequest{
		TenantID: tenantID,
		CSRDER:   buildCSR(t, "other.example.com"),
		Hostname: "api.stripe.com",
	})
	if err == nil {
		t.Fatal("expected error when CSR SAN does not contain the requested hostname")
	}
}

func TestSignCSR_RejectsEmptyHostname(t *testing.T) {
	svc, tenantID := newSignableService(t)

	_, err := svc.SignCSR(context.Background(), SignCSRRequest{
		TenantID: tenantID,
		CSRDER:   buildCSR(t, "api.stripe.com"),
		Hostname: "",
	})
	if err == nil {
		t.Fatal("expected error for empty hostname")
	}
}

func TestSignCSR_RejectsCSRWithNoSANs(t *testing.T) {
	svc, tenantID := newSignableService(t)

	_, err := svc.SignCSR(context.Background(), SignCSRRequest{
		TenantID: tenantID,
		CSRDER:   buildCSR(t),
		Hostname: "api.stripe.com",
	})
	if err == nil {
		t.Fatal("expected error for CSR with no DNS SANs")
	}
}

func TestSignCSR_RejectsCAExtensionInCSR(t *testing.T) {
	svc, tenantID := newSignableService(t)

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	basicConstraints, err := asn1.Marshal(struct {
		IsCA bool `asn1:"optional"`
	}{IsCA: true})
	if err != nil {
		t.Fatalf("marshal basic constraints: %v", err)
	}

	template := &x509.CertificateRequest{
		DNSNames: []string{"api.stripe.com"},
		ExtraExtensions: []pkix.Extension{
			{Id: asn1.ObjectIdentifier{2, 5, 29, 19}, Value: basicConstraints},
		},
	}
	der, err := x509.CreateCertificateRequest(rand.Reader, template, key)
	if err != nil {
		t.Fatalf("create CSR: %v", err)
	}

	_, err = svc.SignCSR(context.Background(), SignCSRRequest{
		TenantID: tenantID,
		CSRDER:   der,
		Hostname: "api.stripe.com",
	})
	if err == nil {
		t.Fatal("expected error for CSR containing a BasicConstraints extension")
	}
}

func TestSignCSR_RejectsInvalidCSRBytes(t *testing.T) {
	svc, tenantID := newSignableService(t)

	_, err := svc.SignCSR(context.Background(), SignCSRRequest{
		TenantID: tenantID,
		CSRDER:   []byte("not a real CSR"),
		Hostname: "api.stripe.com",
	})
	if err == nil {
		t.Fatal("expected error for malformed CSR bytes")
	}
}

func TestSignCSR_NoLazyCAGeneration(t *testing.T) {
	encKey := make([]byte, 32)
	rand.Read(encKey)
	repo := &mockTenantCARepo{}
	svc := NewTenantCAService(repo, encKey)

	_, err := svc.SignCSR(context.Background(), SignCSRRequest{
		TenantID: uuid.New(),
		CSRDER:   buildCSR(t, "api.stripe.com"),
		Hostname: "api.stripe.com",
	})
	if err == nil {
		t.Fatal("expected error when no tenant CA exists")
	}
	if repo.stored {
		t.Fatal("SignCSR must not create a CA as a side effect")
	}
}

func TestSignCSR_LeafSignedByTenantCA(t *testing.T) {
	svc, tenantID := newSignableService(t)
	certPEM, err := svc.GetTenantCACert(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("GetTenantCACert: %v", err)
	}

	resp, err := svc.SignCSR(context.Background(), SignCSRRequest{
		TenantID: tenantID,
		CSRDER:   buildCSR(t, "api.stripe.com"),
		Hostname: "api.stripe.com",
	})
	if err != nil {
		t.Fatalf("SignCSR: %v", err)
	}

	leaf, err := x509.ParseCertificate(resp.CertDER)
	if err != nil {
		t.Fatalf("parse leaf: %v", err)
	}

	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		t.Fatal("failed to decode CA cert PEM")
	}
	caCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("parse CA cert: %v", err)
	}

	if err := leaf.CheckSignatureFrom(caCert); err != nil {
		t.Fatalf("leaf not signed by tenant CA: %v", err)
	}
}

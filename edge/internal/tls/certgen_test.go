package tls

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"testing"
	"time"
)

// testCA is a self-signed CA used only within tests to sign CSRs, standing
// in for Cloud's real signing service via a fakeSigner.
type testCA struct {
	cert *x509.Certificate
	key  *ecdsa.PrivateKey
}

func newTestCA(t *testing.T) *testCA {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate CA key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Test Root CA"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		MaxPathLen:            1,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create CA cert: %v", err)
	}

	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse CA cert: %v", err)
	}

	return &testCA{cert: cert, key: key}
}

func (ca *testCA) certPEM() []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ca.cert.Raw})
}

func (ca *testCA) sign(csrDER []byte) ([]byte, error) {
	csr, err := x509.ParseCertificateRequest(csrDER)
	if err != nil {
		return nil, err
	}
	if err := csr.CheckSignature(); err != nil {
		return nil, err
	}

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(time.Now().UnixNano()),
		Subject:               csr.Subject,
		DNSNames:              csr.DNSNames,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	return x509.CreateCertificate(rand.Reader, template, ca.cert, csr.PublicKey, ca.key)
}

// fakeSigner implements LeafSigner by signing with a testCA, standing in for
// the SignLeafCertificate RPC that would otherwise call out to Cloud.
type fakeSigner struct {
	ca       *testCA
	notAfter time.Time
	err      error
}

func (f *fakeSigner) SignLeaf(ctx context.Context, csrDER []byte, hostname string) ([]byte, time.Time, error) {
	if f.err != nil {
		return nil, time.Time{}, f.err
	}
	certDER, err := f.ca.sign(csrDER)
	if err != nil {
		return nil, time.Time{}, err
	}
	notAfter := f.notAfter
	if notAfter.IsZero() {
		notAfter = time.Now().Add(24 * time.Hour)
	}
	return certDER, notAfter, nil
}

func TestGenerateLeafCert_Properties(t *testing.T) {
	ca := newTestCA(t)
	anchor, err := NewTrustAnchor(ca.certPEM())
	if err != nil {
		t.Fatalf("NewTrustAnchor: %v", err)
	}
	signer := &fakeSigner{ca: ca}

	cert, notAfter, err := GenerateLeafCert(context.Background(), "api.stripe.com", anchor, signer)
	if err != nil {
		t.Fatalf("GenerateLeafCert: %v", err)
	}
	if notAfter.IsZero() {
		t.Error("notAfter should not be zero")
	}

	if len(cert.Certificate) != 2 {
		t.Fatalf("chain length = %d, want 2 (leaf + CA)", len(cert.Certificate))
	}

	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Fatalf("parse leaf: %v", err)
	}

	if leaf.Subject.CommonName != "api.stripe.com" {
		t.Errorf("CN = %q, want %q", leaf.Subject.CommonName, "api.stripe.com")
	}
	if len(leaf.DNSNames) != 1 || leaf.DNSNames[0] != "api.stripe.com" {
		t.Errorf("DNSNames = %v, want [api.stripe.com]", leaf.DNSNames)
	}
	if leaf.IsCA {
		t.Error("IsCA = true, want false")
	}
	if leaf.KeyUsage&x509.KeyUsageDigitalSignature == 0 {
		t.Error("missing KeyUsageDigitalSignature")
	}
	if len(leaf.ExtKeyUsage) != 1 || leaf.ExtKeyUsage[0] != x509.ExtKeyUsageServerAuth {
		t.Errorf("ExtKeyUsage = %v, want [ServerAuth]", leaf.ExtKeyUsage)
	}

	pubKey, ok := leaf.PublicKey.(*ecdsa.PublicKey)
	if !ok || pubKey.Curve != elliptic.P256() {
		t.Error("leaf key not ECDSA P-256")
	}

	if _, ok := cert.PrivateKey.(*ecdsa.PrivateKey); !ok {
		t.Error("ephemeral leaf private key not ECDSA")
	}

	if err := leaf.CheckSignatureFrom(ca.cert); err != nil {
		t.Errorf("leaf not signed by CA: %v", err)
	}
}

func TestGenerateLeafCert_RejectsWildcard(t *testing.T) {
	ca := newTestCA(t)
	anchor, err := NewTrustAnchor(ca.certPEM())
	if err != nil {
		t.Fatalf("NewTrustAnchor: %v", err)
	}
	signer := &fakeSigner{ca: ca}

	_, _, err = GenerateLeafCert(context.Background(), "*.stripe.com", anchor, signer)
	if err == nil {
		t.Error("expected error for wildcard hostname, got nil")
	}
}

func TestGenerateLeafCert_SignerError(t *testing.T) {
	ca := newTestCA(t)
	anchor, err := NewTrustAnchor(ca.certPEM())
	if err != nil {
		t.Fatalf("NewTrustAnchor: %v", err)
	}
	signer := &fakeSigner{ca: ca, err: errors.New("cloud unavailable")}

	_, _, err = GenerateLeafCert(context.Background(), "api.stripe.com", anchor, signer)
	if err == nil {
		t.Error("expected error when signer fails, got nil")
	}
}

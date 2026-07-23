package tls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"
)

func TestNewCertificateAuthority_Properties(t *testing.T) {
	ca, err := NewCertificateAuthority()
	if err != nil {
		t.Fatalf("NewCertificateAuthority: %v", err)
	}

	block, _ := pem.Decode(ca.CACertPEM())
	if block == nil {
		t.Fatal("CACertPEM returned invalid PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("parse cert: %v", err)
	}

	if cert.Subject.CommonName != "Dipole Proxy CA" {
		t.Errorf("CN = %q, want %q", cert.Subject.CommonName, "Dipole Proxy CA")
	}
	if !cert.IsCA {
		t.Error("IsCA = false, want true")
	}
	if cert.Issuer.CommonName != cert.Subject.CommonName {
		t.Error("not self-signed: issuer != subject")
	}
	if cert.KeyUsage&x509.KeyUsageCertSign == 0 {
		t.Error("missing KeyUsageCertSign")
	}
	if cert.KeyUsage&x509.KeyUsageCRLSign == 0 {
		t.Error("missing KeyUsageCRLSign")
	}
	if cert.MaxPathLen != 1 {
		t.Errorf("MaxPathLen = %d, want 1", cert.MaxPathLen)
	}

	pubKey, ok := cert.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		t.Fatal("public key is not ECDSA")
	}
	if pubKey.Curve != elliptic.P256() {
		t.Errorf("curve = %v, want P-256", pubKey.Curve.Params().Name)
	}

	validity := cert.NotAfter.Sub(cert.NotBefore)
	if validity < 9*365*24*time.Hour {
		t.Errorf("validity = %v, want ~10 years", validity)
	}

	if err := cert.CheckSignatureFrom(cert); err != nil {
		t.Errorf("self-sign verification failed: %v", err)
	}
}

func TestLoadCertificateAuthority_ValidPEM(t *testing.T) {
	ca, err := NewCertificateAuthority()
	if err != nil {
		t.Fatalf("generate CA: %v", err)
	}

	keyDER, err := x509.MarshalECPrivateKey(ca.caKey)
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	loaded, err := LoadCertificateAuthority(ca.CACertPEM(), keyPEM)
	if err != nil {
		t.Fatalf("LoadCertificateAuthority: %v", err)
	}

	if string(loaded.CACertPEM()) != string(ca.CACertPEM()) {
		t.Error("loaded CA cert does not match original")
	}

	if loaded.caCert.SerialNumber.Cmp(ca.caCert.SerialNumber) != 0 {
		t.Error("loaded CA serial does not match original")
	}

	if loaded.caKey.D.Cmp(ca.caKey.D) != 0 {
		t.Error("loaded CA key does not match original")
	}
}

func TestLoadCertificateAuthority_SignsLeafCert(t *testing.T) {
	ca, err := NewCertificateAuthority()
	if err != nil {
		t.Fatalf("generate CA: %v", err)
	}

	keyDER, err := x509.MarshalECPrivateKey(ca.caKey)
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	loaded, err := LoadCertificateAuthority(ca.CACertPEM(), keyPEM)
	if err != nil {
		t.Fatalf("LoadCertificateAuthority: %v", err)
	}

	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate leaf key: %v", err)
	}

	serial, _ := rand.Int(rand.Reader, big.NewInt(1000000))
	leafTemplate := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "test.example.com"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(24 * time.Hour),
	}

	leafDER, err := loaded.signLeaf(leafTemplate, &leafKey.PublicKey)
	if err != nil {
		t.Fatalf("signLeaf: %v", err)
	}

	leafCert, err := x509.ParseCertificate(leafDER)
	if err != nil {
		t.Fatalf("parse leaf: %v", err)
	}

	roots := x509.NewCertPool()
	roots.AddCert(loaded.caCert)
	if _, err := leafCert.Verify(x509.VerifyOptions{Roots: roots}); err != nil {
		t.Fatalf("leaf cert does not chain to loaded CA: %v", err)
	}
}

func TestLoadCertificateAuthority_RejectsInvalidCert(t *testing.T) {
	_, err := LoadCertificateAuthority([]byte("not-pem"), []byte("not-pem"))
	if err == nil {
		t.Fatal("expected error for invalid cert PEM")
	}
}

func TestLoadCertificateAuthority_RejectsNonCA(t *testing.T) {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	serial, _ := rand.Int(rand.Reader, big.NewInt(1000000))

	template := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "not-a-ca"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
		IsCA:         false,
	}

	der, _ := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})

	keyDER, _ := x509.MarshalECPrivateKey(key)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	_, err := LoadCertificateAuthority(certPEM, keyPEM)
	if err == nil {
		t.Fatal("expected error for non-CA certificate")
	}
}

func TestLoadCertificateAuthority_RejectsInvalidKey(t *testing.T) {
	ca, _ := NewCertificateAuthority()
	_, err := LoadCertificateAuthority(ca.CACertPEM(), []byte("not-a-key"))
	if err == nil {
		t.Fatal("expected error for invalid key PEM")
	}
}

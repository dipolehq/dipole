package tls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/x509"
	"encoding/pem"
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

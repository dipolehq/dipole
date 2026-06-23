package tls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/x509"
	"testing"
	"time"
)

func TestGenerateLeafCert_Properties(t *testing.T) {
	ca, _ := NewCertificateAuthority()

	cert, err := GenerateLeafCert("api.stripe.com", ca)
	if err != nil {
		t.Fatalf("GenerateLeafCert: %v", err)
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

	validity := leaf.NotAfter.Sub(leaf.NotBefore)
	if validity < 23*time.Hour || validity > 25*time.Hour {
		t.Errorf("validity = %v, want ~24h", validity)
	}

	if err := leaf.CheckSignatureFrom(ca.caCert); err != nil {
		t.Errorf("leaf not signed by CA: %v", err)
	}
}

func TestGenerateLeafCert_RejectsWildcard(t *testing.T) {
	ca, _ := NewCertificateAuthority()

	_, err := GenerateLeafCert("*.stripe.com", ca)
	if err == nil {
		t.Error("expected error for wildcard hostname, got nil")
	}
}

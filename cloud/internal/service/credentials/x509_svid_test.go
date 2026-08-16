// x509_svid_test.go tests SPIFFE X.509-SVID certificate issuance: SPIFFE URI SAN, extended key usage, lifetime, and CA chain verification.
package credentials

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

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

func TestX509SVID_CorrectSPIFFEID(t *testing.T) {
	caKey, caCert := generateTestCA(t)
	handler := NewX509SVIDHandler(caKey, caCert)

	cfg, _ := json.Marshal(x509SVIDConfig{
		SpiffeID:        "spiffe://example.org/ns/production/sa/myapp",
		LifetimeMinutes: 15,
	})

	cred, err := handler.Retrieve(context.Background(), cfg, &domain.ClientEvidence{}, nil)
	if err != nil {
		t.Fatal(err)
	}

	if cred.Type != "x509" {
		t.Errorf("type = %q, want x509", cred.Type)
	}

	block, _ := pem.Decode([]byte(cred.Value))
	if block == nil {
		t.Fatal("no PEM block in cert value")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}

	if len(cert.URIs) == 0 {
		t.Fatal("no URI SANs in certificate")
	}
	if cert.URIs[0].String() != "spiffe://example.org/ns/production/sa/myapp" {
		t.Errorf("URI SAN = %v, want spiffe://example.org/ns/production/sa/myapp", cert.URIs[0])
	}

	lifetime := cert.NotAfter.Sub(cert.NotBefore)
	if lifetime < 14*time.Minute || lifetime > 16*time.Minute {
		t.Errorf("lifetime = %v, want ~15min", lifetime)
	}

	hasClientAuth := false
	for _, eku := range cert.ExtKeyUsage {
		if eku == x509.ExtKeyUsageClientAuth {
			hasClientAuth = true
		}
	}
	if !hasClientAuth {
		t.Error("missing ExtKeyUsageClientAuth")
	}

	roots := x509.NewCertPool()
	roots.AddCert(caCert)
	_, err = cert.Verify(x509.VerifyOptions{
		Roots:     roots,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	})
	if err != nil {
		t.Errorf("cert not signed by CA: %v", err)
	}

	if cred.Password == "" {
		t.Error("private key should be set")
	}
	keyBlock, _ := pem.Decode([]byte(cred.Password))
	if keyBlock == nil || keyBlock.Type != "EC PRIVATE KEY" {
		t.Error("private key should be EC PRIVATE KEY PEM")
	}
}

func TestX509SVID_ServerAuth_EKU(t *testing.T) {
	caKey, caCert := generateTestCA(t)
	handler := NewX509SVIDHandler(caKey, caCert)

	cfg, _ := json.Marshal(x509SVIDConfig{
		SpiffeID:         "spiffe://example.org/web-server",
		LifetimeMinutes:  5,
		ExtendedKeyUsage: []string{"server_auth", "client_auth"},
	})

	cred, err := handler.Retrieve(context.Background(), cfg, &domain.ClientEvidence{}, nil)
	if err != nil {
		t.Fatal(err)
	}

	block, _ := pem.Decode([]byte(cred.Value))
	cert, _ := x509.ParseCertificate(block.Bytes)

	hasServer, hasClient := false, false
	for _, eku := range cert.ExtKeyUsage {
		if eku == x509.ExtKeyUsageServerAuth {
			hasServer = true
		}
		if eku == x509.ExtKeyUsageClientAuth {
			hasClient = true
		}
	}
	if !hasServer {
		t.Error("missing ExtKeyUsageServerAuth")
	}
	if !hasClient {
		t.Error("missing ExtKeyUsageClientAuth")
	}
}

func generateTestCA(t *testing.T) (*ecdsa.PrivateKey, *x509.Certificate) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Test Standalone CA"},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		t.Fatal(err)
	}

	return key, cert
}

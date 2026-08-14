package pki

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/crypto"
)

type mockTenantCARepo struct {
	stored       bool
	certPEM      string
	keyEncrypted []byte
}

func (m *mockTenantCARepo) GetOrCreateCA(_ context.Context, _ uuid.UUID, certPEM string, keyEncrypted []byte) (string, []byte, error) {
	if m.stored {
		return m.certPEM, m.keyEncrypted, nil
	}
	m.stored = true
	m.certPEM = certPEM
	m.keyEncrypted = keyEncrypted
	return certPEM, keyEncrypted, nil
}

func TestGetOrCreateTenantCA_GeneratesValidCA(t *testing.T) {
	encKey := make([]byte, 32)
	if _, err := rand.Read(encKey); err != nil {
		t.Fatalf("generate key: %v", err)
	}

	repo := &mockTenantCARepo{}
	svc := NewTenantCAService(repo, encKey)

	tenantID := uuid.New()
	certPEM, keyPEM, err := svc.GetOrCreateTenantCA(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("GetOrCreateTenantCA: %v", err)
	}

	if certPEM == "" {
		t.Fatal("certPEM is empty")
	}
	if keyPEM == "" {
		t.Fatal("keyPEM is empty")
	}

	block, _ := pem.Decode([]byte(certPEM))
	if block == nil || block.Type != "CERTIFICATE" {
		t.Fatalf("invalid cert PEM block type: %v", block)
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("parse certificate: %v", err)
	}

	if !cert.IsCA {
		t.Fatal("certificate is not a CA")
	}
	if cert.KeyUsage&x509.KeyUsageCertSign == 0 {
		t.Fatal("certificate missing CertSign key usage")
	}

	expectedCN := "Dipole Tenant CA (" + tenantID.String() + ")"
	if cert.Subject.CommonName != expectedCN {
		t.Fatalf("unexpected CN: got %q, want %q", cert.Subject.CommonName, expectedCN)
	}

	keyBlock, _ := pem.Decode([]byte(keyPEM))
	if keyBlock == nil || keyBlock.Type != "EC PRIVATE KEY" {
		t.Fatalf("invalid key PEM block type: %v", keyBlock)
	}

	key, err := x509.ParseECPrivateKey(keyBlock.Bytes)
	if err != nil {
		t.Fatalf("parse key: %v", err)
	}

	if key.Curve.Params().BitSize != 256 {
		t.Fatalf("unexpected curve: got %d bits, want 256", key.Curve.Params().BitSize)
	}
}

func TestGetOrCreateTenantCA_ReturnsExistingCA(t *testing.T) {
	encKey := make([]byte, 32)
	if _, err := rand.Read(encKey); err != nil {
		t.Fatalf("generate key: %v", err)
	}

	repo := &mockTenantCARepo{}
	svc := NewTenantCAService(repo, encKey)

	tenantID := uuid.New()
	cert1, key1, err := svc.GetOrCreateTenantCA(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}

	cert2, key2, err := svc.GetOrCreateTenantCA(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}

	if cert1 != cert2 {
		t.Fatal("second call returned different cert")
	}
	if key1 != key2 {
		t.Fatal("second call returned different key")
	}
}

func TestGetOrCreateTenantCA_EncryptsKeyAtRest(t *testing.T) {
	encKey := make([]byte, 32)
	if _, err := rand.Read(encKey); err != nil {
		t.Fatalf("generate key: %v", err)
	}

	repo := &mockTenantCARepo{}
	svc := NewTenantCAService(repo, encKey)

	tenantID := uuid.New()
	_, keyPEM, err := svc.GetOrCreateTenantCA(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("GetOrCreateTenantCA: %v", err)
	}

	decrypted, err := crypto.Decrypt(repo.keyEncrypted, encKey)
	if err != nil {
		t.Fatalf("decrypt stored key: %v", err)
	}

	if string(decrypted) != keyPEM {
		t.Fatal("stored encrypted key does not decrypt to the returned key")
	}
}

func TestGenerateCA_ProducesUniqueSerials(t *testing.T) {
	encKey := make([]byte, 32)
	rand.Read(encKey)

	svc := &TenantCAService{encryptionKey: encKey}

	cert1, _, err := svc.generateCA(uuid.New())
	if err != nil {
		t.Fatalf("generate first: %v", err)
	}

	cert2, _, err := svc.generateCA(uuid.New())
	if err != nil {
		t.Fatalf("generate second: %v", err)
	}

	block1, _ := pem.Decode([]byte(cert1))
	block2, _ := pem.Decode([]byte(cert2))

	c1, _ := x509.ParseCertificate(block1.Bytes)
	c2, _ := x509.ParseCertificate(block2.Bytes)

	if c1.SerialNumber.Cmp(c2.SerialNumber) == 0 {
		t.Fatal("two generated CAs have the same serial number")
	}
}

func TestGenerateCA_KeyMatchesCert(t *testing.T) {
	encKey := make([]byte, 32)
	rand.Read(encKey)

	svc := &TenantCAService{encryptionKey: encKey}

	certPEM, keyPEM, err := svc.generateCA(uuid.New())
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	certBlock, _ := pem.Decode([]byte(certPEM))
	cert, _ := x509.ParseCertificate(certBlock.Bytes)

	keyBlock, _ := pem.Decode([]byte(keyPEM))
	key, _ := x509.ParseECPrivateKey(keyBlock.Bytes)

	pubKey := cert.PublicKey.(*ecdsa.PublicKey)
	if pubKey.X.Cmp(key.PublicKey.X) != 0 || pubKey.Y.Cmp(key.PublicKey.Y) != 0 {
		t.Fatal("certificate public key does not match private key")
	}
}

func (m *mockTenantCARepo) HasCA(_ context.Context, _ uuid.UUID) (bool, error) {
	return m.stored, nil
}

func (m *mockTenantCARepo) GetLeafLifetime(_ context.Context, _ uuid.UUID) (int, error) {
	return 86400, nil
}

func (m *mockTenantCARepo) SetLeafLifetime(_ context.Context, _ uuid.UUID, _ int) error {
	return nil
}

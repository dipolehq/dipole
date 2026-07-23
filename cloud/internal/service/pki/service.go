/*
Package pki implements per-tenant Certificate Authority generation and retrieval.

TenantCAService generates an ECDSA P-256 Root CA on first call for a given
tenant and returns the same CA on subsequent calls. The private key is encrypted
with AES-256-GCM before storage. Race conditions are handled at the repository
level (SELECT ... FOR UPDATE).
*/
package pki

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/repository"
	"github.com/dipolehq/dipole/cloud/internal/repository/postgres"
)

type TenantCAService struct {
	repo          repository.TenantCARepository
	encryptionKey []byte
}

func NewTenantCAService(repo repository.TenantCARepository, encryptionKey []byte) *TenantCAService {
	return &TenantCAService{repo: repo, encryptionKey: encryptionKey}
}

func (s *TenantCAService) GetOrCreateTenantCA(ctx context.Context, tenantID uuid.UUID) (string, string, error) {
	certPEM, keyPEM, err := s.generateCA(tenantID)
	if err != nil {
		return "", "", fmt.Errorf("generate CA: %w", err)
	}

	keyEncrypted, err := postgres.Encrypt([]byte(keyPEM), s.encryptionKey)
	if err != nil {
		return "", "", fmt.Errorf("encrypt CA key: %w", err)
	}

	storedCert, storedKeyEnc, err := s.repo.GetOrCreateCA(ctx, tenantID, certPEM, keyEncrypted)
	if err != nil {
		return "", "", fmt.Errorf("store CA: %w", err)
	}

	storedKeyPEM, err := postgres.Decrypt(storedKeyEnc, s.encryptionKey)
	if err != nil {
		return "", "", fmt.Errorf("decrypt CA key: %w", err)
	}

	return storedCert, string(storedKeyPEM), nil
}

func (s *TenantCAService) generateCA(tenantID uuid.UUID) (string, string, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("generate key: %w", err)
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return "", "", fmt.Errorf("generate serial: %w", err)
	}

	template := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: fmt.Sprintf("Dipole Tenant CA (%s)", tenantID)},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		MaxPathLen:            1,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return "", "", fmt.Errorf("create certificate: %w", err)
	}

	certPEM := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))

	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return "", "", fmt.Errorf("marshal key: %w", err)
	}
	keyPEM := string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}))

	return certPEM, keyPEM, nil
}

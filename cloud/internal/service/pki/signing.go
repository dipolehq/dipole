/*
Package pki — signing.go implements CSR validation and leaf certificate signing.

The proxy generates an ephemeral ECDSA key pair + CSR locally and sends the
CSR to Cloud. Cloud validates the CSR, then signs it with the tenant's Root CA.
The CA private key is decrypted in memory only during signing.

Security model:
  - The caller (gRPC handler) authenticates the proxy and authorizes the hostname.
  - SignCSR requires the tenant CA to already exist (no lazy generation).
  - CSR must contain the requested hostname as a DNS SAN (wildcards rejected).
  - The issued leaf cert only includes the single authorized hostname regardless of CSR SANs.
  - CA extensions in the CSR are rejected.
  - Leaf certs use KeyUsage=DigitalSignature, ExtKeyUsage=ServerAuth, IsCA=false.
  - NotBefore is backdated 5 minutes for clock skew tolerance.
*/
package pki

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
)

type SignCSRRequest struct {
	TenantID uuid.UUID
	CSRDER   []byte
	Hostname string
}

type SignCSRResponse struct {
	CertDER  []byte
	NotAfter time.Time
}

func (s *TenantCAService) SignCSR(ctx context.Context, req SignCSRRequest) (*SignCSRResponse, error) {
	csr, err := x509.ParseCertificateRequest(req.CSRDER)
	if err != nil {
		return nil, fmt.Errorf("invalid CSR: %w", err)
	}

	if err := csr.CheckSignature(); err != nil {
		return nil, fmt.Errorf("CSR signature verification failed: %w", err)
	}

	if err := validateCSRHostname(csr, req.Hostname); err != nil {
		return nil, err
	}

	if err := validateNoCAExtensions(csr); err != nil {
		return nil, err
	}

	hasCA, err := s.repo.HasCA(ctx, req.TenantID)
	if err != nil || !hasCA {
		return nil, fmt.Errorf("tenant CA not available: TLS Decrypt must be enabled first")
	}

	certPEM, keyPEM, err := s.GetOrCreateTenantCA(ctx, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("load tenant CA: %w", err)
	}

	caCertBlock, _ := pem.Decode([]byte(certPEM))
	if caCertBlock == nil {
		return nil, fmt.Errorf("invalid CA certificate PEM")
	}
	caCert, err := x509.ParseCertificate(caCertBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse CA certificate: %w", err)
	}

	caKeyBlock, _ := pem.Decode([]byte(keyPEM))
	if caKeyBlock == nil {
		return nil, fmt.Errorf("invalid CA key PEM")
	}
	caKey, err := x509.ParseECPrivateKey(caKeyBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse CA key: %w", err)
	}

	lifetimeSeconds, err := s.repo.GetLeafLifetime(ctx, req.TenantID)
	if err != nil {
		lifetimeSeconds = 86400
	}

	leafCertDER, notAfter, err := issueLeafCertificate(csr, caCert, caKey, req.Hostname, lifetimeSeconds)
	if err != nil {
		return nil, fmt.Errorf("issue leaf certificate: %w", err)
	}

	caKey = nil

	return &SignCSRResponse{
		CertDER:  leafCertDER,
		NotAfter: notAfter,
	}, nil
}

func validateCSRHostname(csr *x509.CertificateRequest, hostname string) error {
	if hostname == "" {
		return fmt.Errorf("hostname is required")
	}
	if strings.Contains(hostname, "*") {
		return fmt.Errorf("wildcard hostnames are not allowed")
	}
	if len(csr.DNSNames) == 0 {
		return fmt.Errorf("CSR must contain at least one DNS SAN")
	}

	found := false
	for _, san := range csr.DNSNames {
		if strings.Contains(san, "*") {
			return fmt.Errorf("wildcard SANs are not allowed in CSR")
		}
		if san == hostname {
			found = true
		}
	}
	if !found {
		return fmt.Errorf("CSR SAN does not contain the requested hostname %q", hostname)
	}
	return nil
}

func validateNoCAExtensions(csr *x509.CertificateRequest) error {
	for _, ext := range csr.Extensions {
		if ext.Id.String() == "2.5.29.19" {
			return fmt.Errorf("CSR must not contain BasicConstraints extension")
		}
	}
	return nil
}

func issueLeafCertificate(
	csr *x509.CertificateRequest,
	caCert *x509.Certificate,
	caKey *ecdsa.PrivateKey,
	hostname string,
	lifetimeSeconds int,
) ([]byte, time.Time, error) {
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("generate serial: %w", err)
	}

	now := time.Now()
	notBefore := now.Add(-5 * time.Minute)
	notAfter := now.Add(time.Duration(lifetimeSeconds) * time.Second)

	template := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: hostname},
		DNSNames:              []string{hostname},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, caCert, csr.PublicKey, caKey)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("sign certificate: %w", err)
	}

	return certDER, notAfter, nil
}

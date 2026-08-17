// signing.go validates a proxy-submitted CSR and signs it into a short-lived leaf certificate with the tenant's Root CA.
package pki

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
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

	certPEM, keyPEM, err := s.getTenantCA(ctx, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("tenant CA not available: TLS Decrypt must be enabled first: %w", err)
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
	serial, err := generateSerial()
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

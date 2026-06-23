/*
Package tls provides the TLS MITM infrastructure for the Agent Proxy.

CertificateAuthority generates an in-memory ECDSA P-256 Root CA on startup.
The CA private key never leaves this struct — it is used only internally to
sign leaf certificates. CACertPEM exposes the public certificate for trust
injection into client workloads.
*/
package tls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"
)

type CertificateAuthority struct {
	caKey     *ecdsa.PrivateKey
	caCert    *x509.Certificate
	caCertDER []byte
	caCertPEM []byte
}

func NewCertificateAuthority() (*CertificateAuthority, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate CA key: %w", err)
	}

	serial, err := randomSerial()
	if err != nil {
		return nil, fmt.Errorf("generate serial: %w", err)
	}

	template := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "Dipole Proxy CA"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		MaxPathLen:            1,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, fmt.Errorf("create CA certificate: %w", err)
	}

	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, fmt.Errorf("parse CA certificate: %w", err)
	}

	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})

	return &CertificateAuthority{
		caKey:     key,
		caCert:    cert,
		caCertDER: der,
		caCertPEM: pemBytes,
	}, nil
}

func (ca *CertificateAuthority) CACertPEM() []byte {
	return ca.caCertPEM
}

func (ca *CertificateAuthority) signLeaf(template *x509.Certificate, leafPubKey *ecdsa.PublicKey) ([]byte, error) {
	return x509.CreateCertificate(rand.Reader, template, ca.caCert, leafPubKey, ca.caKey)
}

func randomSerial() (*big.Int, error) {
	return rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
}

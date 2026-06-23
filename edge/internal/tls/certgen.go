/*
certgen.go generates per-hostname leaf TLS certificates signed by the Root CA.

GenerateLeafCert creates an ECDSA P-256 leaf cert for an exact hostname (never
wildcards), with 24-hour validity, SAN DNS=[hostname], and the full chain
(leaf + CA) in the returned tls.Certificate.
*/
package tls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"strings"
	"time"
)

func GenerateLeafCert(hostname string, ca *CertificateAuthority) (*tls.Certificate, error) {
	if strings.HasPrefix(hostname, "*.") {
		return nil, fmt.Errorf("wildcard hostnames not allowed: %s", hostname)
	}

	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate leaf key: %w", err)
	}

	serial, err := randomSerial()
	if err != nil {
		return nil, fmt.Errorf("generate serial: %w", err)
	}

	template := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: hostname},
		DNSNames:     []string{hostname},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	leafDER, err := ca.signLeaf(template, &leafKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("sign leaf cert: %w", err)
	}

	return &tls.Certificate{
		Certificate: [][]byte{leafDER, ca.caCertDER},
		PrivateKey:  leafKey,
	}, nil
}

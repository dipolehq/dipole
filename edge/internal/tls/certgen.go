// certgen.go generates a per-hostname key pair and CSR locally, then requests Cloud to sign it into a leaf certificate.
package tls

import (
	"context"
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

// LeafSigner signs a proxy-submitted CSR into a leaf certificate. The
// tenant's Root CA private key is held by the implementation's backing
// service (Cloud) and never exposed to the caller — only the signed
// certificate is returned.
type LeafSigner interface {
	SignLeaf(ctx context.Context, csrDER []byte, hostname string) (certDER []byte, notAfter time.Time, err error)
}

// GenerateLeafCert generates an ephemeral ECDSA P-256 key pair and CSR for
// hostname (never wildcards), has signer sign it into a leaf certificate,
// and returns the assembled chain (leaf + Root CA) with the ephemeral
// private key. The ephemeral private key never leaves the proxy.
func GenerateLeafCert(ctx context.Context, hostname string, anchor *TrustAnchor, signer LeafSigner) (*tls.Certificate, time.Time, error) {
	if strings.HasPrefix(hostname, "*.") {
		return nil, time.Time{}, fmt.Errorf("wildcard hostnames not allowed: %s", hostname)
	}

	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("generate leaf key: %w", err)
	}

	csrTemplate := &x509.CertificateRequest{
		Subject:  pkix.Name{CommonName: hostname},
		DNSNames: []string{hostname},
	}

	csrDER, err := x509.CreateCertificateRequest(rand.Reader, csrTemplate, leafKey)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("create CSR: %w", err)
	}

	certDER, notAfter, err := signer.SignLeaf(ctx, csrDER, hostname)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("sign leaf cert: %w", err)
	}

	return &tls.Certificate{
		Certificate: [][]byte{certDER, anchor.certDER},
		PrivateKey:  leafKey,
	}, notAfter, nil
}

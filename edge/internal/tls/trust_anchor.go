// trust_anchor.go holds the tenant's Root CA public certificate, used to build the trust chain for leaf certificates signed by Cloud.
package tls

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
)

// TrustAnchor holds only the tenant's Root CA public certificate. The
// private key never leaves Cloud: leaf certificates are signed remotely via
// LeafSigner, and the proxy never has signing capability of its own. This
// is deliberately not a "CertificateAuthority" — it cannot sign anything.
type TrustAnchor struct {
	cert    *x509.Certificate
	certDER []byte
	certPEM []byte
}

// NewTrustAnchor parses the tenant's Root CA certificate (PEM, as returned
// by Cloud's Register RPC) into a TrustAnchor.
func NewTrustAnchor(certPEM []byte) (*TrustAnchor, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("invalid CA certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse CA certificate: %w", err)
	}

	if !cert.IsCA {
		return nil, fmt.Errorf("certificate is not a CA")
	}

	return &TrustAnchor{
		cert:    cert,
		certDER: block.Bytes,
		certPEM: certPEM,
	}, nil
}

// CACertPEM returns the Root CA's public certificate, for local trust-store
// injection on the host running the proxy.
func (t *TrustAnchor) CACertPEM() []byte {
	return t.certPEM
}

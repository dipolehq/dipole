// x509_svid.go issues SPIFFE X.509-SVID certificates for mTLS authentication, signed by a standalone CA.
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
	"fmt"
	"math/big"
	"net/url"
	"time"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type X509SVIDHandler struct {
	caKey  *ecdsa.PrivateKey
	caCert *x509.Certificate
}

func NewX509SVIDHandler(caKey *ecdsa.PrivateKey, caCert *x509.Certificate) *X509SVIDHandler {
	return &X509SVIDHandler{caKey: caKey, caCert: caCert}
}

type x509SVIDConfig struct {
	SpiffeID         string   `json:"spiffe_id"`
	LifetimeMinutes  int      `json:"lifetime_minutes"`
	ExtendedKeyUsage []string `json:"extended_key_usage"`
}

func (h *X509SVIDHandler) Retrieve(_ context.Context, configJSON []byte, _ *domain.ClientEvidence, _ map[string]interface{}) (*domain.Credential, error) {
	var cfg x509SVIDConfig
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		return nil, fmt.Errorf("parse x509-svid config: %w", err)
	}

	lifetime := time.Duration(cfg.LifetimeMinutes) * time.Minute
	if lifetime == 0 {
		lifetime = 15 * time.Minute
	}

	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate leaf key: %w", err)
	}

	spiffeURI, err := url.Parse(cfg.SpiffeID)
	if err != nil {
		return nil, fmt.Errorf("parse spiffe id: %w", err)
	}

	ekus := buildEKUs(cfg.ExtendedKeyUsage)

	now := time.Now()
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(now.UnixNano()),
		Subject:               pkix.Name{},
		NotBefore:             now,
		NotAfter:              now.Add(lifetime),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           ekus,
		URIs:                  []*url.URL{spiffeURI},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, h.caCert, &leafKey.PublicKey, h.caKey)
	if err != nil {
		return nil, fmt.Errorf("sign x509-svid: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: h.caCert.Raw})
	chainPEM := string(certPEM) + string(caPEM)

	keyDER, err := x509.MarshalECPrivateKey(leafKey)
	if err != nil {
		return nil, fmt.Errorf("marshal leaf key: %w", err)
	}
	keyPEM := string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}))

	return &domain.Credential{
		Type:     "x509",
		Value:    chainPEM,
		Password: keyPEM,
	}, nil
}

func buildEKUs(names []string) []x509.ExtKeyUsage {
	if len(names) == 0 {
		return []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	}
	var ekus []x509.ExtKeyUsage
	for _, name := range names {
		switch name {
		case "client_auth":
			ekus = append(ekus, x509.ExtKeyUsageClientAuth)
		case "server_auth":
			ekus = append(ekus, x509.ExtKeyUsageServerAuth)
		}
	}
	if len(ekus) == 0 {
		return []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	}
	return ekus
}

/*
certificate.go builds a certificate_signed_attestation proof-of-possession
attestation for the CLI's proxyless credential flow.

The CLI signs a canonical payload binding the workload's certificate and a
timestamp with the certificate's own private key, proving to Cloud that the
CLI actually holds that private key — not just a copy of the (public)
certificate. This is stateless proof-of-possession with a bounded freshness
window, not a Cloud-issued nonce/challenge-response protocol: the timestamp
is client-supplied, so a captured {certificate, timestamp, signature} triple
remains valid (and replayable) until Cloud's freshness window expires. See
cloud/internal/service/trust/certificate_validator.go's package doc comment
for the full rationale, and its buildCertificateChallenge for the verifier
side; both sides must construct the exact same canonical bytes independently.
*/
package main

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"os"
	"strconv"
	"time"
)

// certificateChallengePrefix must exactly match
// cloud/internal/service/trust/certificate_validator.go's
// certificateChallengePrefix — this is a cross-repo protocol contract, not
// an implementation detail either side can change independently.
const certificateChallengePrefix = "dipole-certificate-challenge-v1\x00"

type certificateAttestation struct {
	CertificatePEM  string
	Timestamp       int64
	SignatureBase64 string
}

// buildCertificateAttestation loads a PEM certificate and its matching PEM
// private key from disk, signs a freshly-timestamped proof-of-possession
// challenge, and returns the attestation ready to send to Cloud.
func buildCertificateAttestation(certPath, keyPath string) (*certificateAttestation, error) {
	certPEMBytes, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("read --client-tls-certificate: %w", err)
	}
	keyPEMBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("read --client-tls-private-key: %w", err)
	}

	certBlock, _ := pem.Decode(certPEMBytes)
	if certBlock == nil {
		return nil, fmt.Errorf("--client-tls-certificate: no PEM block found")
	}
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("--client-tls-certificate: %w", err)
	}

	signer, err := parsePrivateKeyPEM(keyPEMBytes)
	if err != nil {
		return nil, fmt.Errorf("--client-tls-private-key: %w", err)
	}

	if !publicKeysEqual(cert.PublicKey, signer.Public()) {
		return nil, fmt.Errorf("--client-tls-certificate and --client-tls-private-key do not match: the certificate's public key does not correspond to this private key")
	}

	timestamp := time.Now().UTC().Unix()
	signature, err := signCertificateChallenge(signer, cert.Raw, timestamp)
	if err != nil {
		return nil, fmt.Errorf("sign certificate challenge: %w", err)
	}

	return &certificateAttestation{
		CertificatePEM:  string(certPEMBytes),
		Timestamp:       timestamp,
		SignatureBase64: base64.StdEncoding.EncodeToString(signature),
	}, nil
}

// publicKeysEqual reports whether certPub (from the parsed certificate)
// and signerPub (from the loaded private key) are the same key, failing
// fast and locally with a clear message rather than sending a request
// Cloud is guaranteed to reject with a less specific InvalidSignature.
// Both *rsa.PublicKey and *ecdsa.PublicKey implement the standard
// library's Equal(crypto.PublicKey) bool method.
func publicKeysEqual(certPub, signerPub crypto.PublicKey) bool {
	type equaler interface {
		Equal(crypto.PublicKey) bool
	}
	e, ok := certPub.(equaler)
	if !ok {
		return false
	}
	return e.Equal(signerPub)
}

// parsePrivateKeyPEM decodes a PEM-encoded private key in any of the
// formats Go's standard library can parse without guessing the type ahead
// of time from the PEM header alone — PKCS#1 ("RSA PRIVATE KEY"), EC
// ("EC PRIVATE KEY"), and PKCS#8 ("PRIVATE KEY", which can wrap either RSA
// or ECDSA) are all in real-world use across certificate tooling.
func parsePrivateKeyPEM(pemBytes []byte) (crypto.Signer, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found")
	}

	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	if key, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("unrecognized private key format (expected PKCS#1, EC, or PKCS#8): %w", err)
	}
	signer, ok := key.(crypto.Signer)
	if !ok {
		return nil, fmt.Errorf("private key type %T does not support signing", key)
	}
	return signer, nil
}

// buildCertificateChallenge must produce byte-for-byte identical output to
// cloud/internal/service/trust/certificate_validator.go's function of the
// same name — see that function's doc comment for the format definition.
func buildCertificateChallenge(certDER []byte, timestamp int64) []byte {
	certHash := sha256.Sum256(certDER)
	challenge := certificateChallengePrefix +
		hex.EncodeToString(certHash[:]) + "\x00" +
		strconv.FormatInt(timestamp, 10)
	return []byte(challenge)
}

// signCertificateChallenge signs buildCertificateChallenge's canonical
// bytes with signer, selecting the signature scheme from the key's own
// type (RSA PKCS#1 v1.5, or ECDSA) to match exactly what
// certificate_validator.go's verifyCertificateChallenge expects for each
// key type.
func signCertificateChallenge(signer crypto.Signer, certDER []byte, timestamp int64) ([]byte, error) {
	challenge := buildCertificateChallenge(certDER, timestamp)
	digest := sha256.Sum256(challenge)

	switch signer.Public().(type) {
	case *rsa.PublicKey:
		return signer.Sign(rand.Reader, digest[:], crypto.SHA256)
	case *ecdsa.PublicKey:
		return signer.Sign(rand.Reader, digest[:], crypto.SHA256)
	default:
		return nil, fmt.Errorf("unsupported private key type %T", signer.Public())
	}
}

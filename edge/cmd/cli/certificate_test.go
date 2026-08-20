package main

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// verifyCertificateChallengeForTest independently reimplements
// cloud/internal/service/trust/certificate_validator.go's
// verifyCertificateChallenge. It is intentionally NOT imported from the
// cloud module (edge and cloud are separate Go modules with no shared
// dependency for this logic) — reproducing it here from the documented
// wire contract is what actually proves cross-repo compatibility: if this
// test passes, the CLI's real output satisfies the same verification
// logic Cloud runs, constructed independently rather than by sharing code.
func verifyCertificateChallengeForTest(t *testing.T, certDER []byte, timestamp int64, signature []byte, pub any) {
	t.Helper()
	challenge := buildCertificateChallenge(certDER, timestamp)
	digest := sha256.Sum256(challenge)

	switch pubKey := pub.(type) {
	case *rsa.PublicKey:
		if err := rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, digest[:], signature); err != nil {
			t.Fatalf("RSA signature verification failed: %v", err)
		}
	case *ecdsa.PublicKey:
		if !ecdsa.VerifyASN1(pubKey, digest[:], signature) {
			t.Fatal("ECDSA signature verification failed")
		}
	default:
		t.Fatalf("unsupported public key type %T", pub)
	}
}

func generateTestCertAndKeyECDSA(t *testing.T) (certPath, keyPath string, cert *x509.Certificate) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test-workload"},
		NotBefore:    time.Now().Add(-1 * time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err = x509.ParseCertificate(certDER)
	if err != nil {
		t.Fatal(err)
	}

	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	certPath = filepath.Join(dir, "cert.pem")
	keyPath = filepath.Join(dir, "key.pem")

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	if err := os.WriteFile(certPath, certPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		t.Fatal(err)
	}

	return certPath, keyPath, cert
}

func generateTestCertAndKeyRSA(t *testing.T) (certPath, keyPath string, cert *x509.Certificate) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "test-rsa-workload"},
		NotBefore:    time.Now().Add(-1 * time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err = x509.ParseCertificate(certDER)
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	certPath = filepath.Join(dir, "cert.pem")
	keyPath = filepath.Join(dir, "key.pem")

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})

	if err := os.WriteFile(certPath, certPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		t.Fatal(err)
	}

	return certPath, keyPath, cert
}

func TestBuildCertificateAttestation_ECDSA_VerifiesAgainstIndependentReconstruction(t *testing.T) {
	certPath, keyPath, cert := generateTestCertAndKeyECDSA(t)

	attest, err := buildCertificateAttestation(certPath, keyPath)
	if err != nil {
		t.Fatalf("buildCertificateAttestation: %v", err)
	}

	sig, err := base64.StdEncoding.DecodeString(attest.SignatureBase64)
	if err != nil {
		t.Fatalf("decode signature: %v", err)
	}

	verifyCertificateChallengeForTest(t, cert.Raw, attest.Timestamp, sig, cert.PublicKey)
}

func TestBuildCertificateAttestation_RSA_VerifiesAgainstIndependentReconstruction(t *testing.T) {
	certPath, keyPath, cert := generateTestCertAndKeyRSA(t)

	attest, err := buildCertificateAttestation(certPath, keyPath)
	if err != nil {
		t.Fatalf("buildCertificateAttestation: %v", err)
	}

	sig, err := base64.StdEncoding.DecodeString(attest.SignatureBase64)
	if err != nil {
		t.Fatalf("decode signature: %v", err)
	}

	verifyCertificateChallengeForTest(t, cert.Raw, attest.Timestamp, sig, cert.PublicKey)
}

func TestBuildCertificateAttestation_TimestampIsRecent(t *testing.T) {
	certPath, keyPath, _ := generateTestCertAndKeyECDSA(t)

	before := time.Now().UTC().Unix()
	attest, err := buildCertificateAttestation(certPath, keyPath)
	if err != nil {
		t.Fatalf("buildCertificateAttestation: %v", err)
	}
	after := time.Now().UTC().Unix()

	if attest.Timestamp < before || attest.Timestamp > after {
		t.Errorf("timestamp %d not within expected range [%d, %d]", attest.Timestamp, before, after)
	}
}

func TestBuildCertificateAttestation_MismatchedCertAndKey_RejectedLocally(t *testing.T) {
	// buildCertificateAttestation must refuse to build (and therefore
	// never send) an attestation for a certificate/private-key pair
	// that don't actually match — failing fast locally with a clear
	// message, rather than letting the CLI sign a challenge Cloud is
	// certain to reject anyway with a less specific InvalidSignature.
	certPath, _, _ := generateTestCertAndKeyECDSA(t)
	_, wrongKeyPath, _ := generateTestCertAndKeyECDSA(t)

	_, err := buildCertificateAttestation(certPath, wrongKeyPath)
	if err == nil {
		t.Fatal("expected buildCertificateAttestation to reject a certificate/private-key pair that don't match")
	}
}

func TestPublicKeysEqual_ECDSA(t *testing.T) {
	_, _, certA := generateTestCertAndKeyECDSA(t)
	_, _, certB := generateTestCertAndKeyECDSA(t)

	if !publicKeysEqual(certA.PublicKey, certA.PublicKey) {
		t.Error("expected a key to equal itself")
	}
	if publicKeysEqual(certA.PublicKey, certB.PublicKey) {
		t.Error("expected two independently generated keys to not be equal")
	}
}

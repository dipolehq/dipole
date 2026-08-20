package trust

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

func generateCACert(t *testing.T) (*ecdsa.PrivateKey, string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Test CA"},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	return key, string(certPEM)
}

// generateLeafCertECDSA returns a leaf certificate (PEM) plus the ECDSA
// private key matching its public key — needed so tests can produce a
// genuine proof-of-possession signature, not just a bare certificate.
func generateLeafCertECDSA(t *testing.T, caKey *ecdsa.PrivateKey, caPEM string, cn string) (string, *ecdsa.PrivateKey) {
	t.Helper()

	block, _ := pem.Decode([]byte(caPEM))
	caCert, _ := x509.ParseCertificate(block.Bytes)

	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: cn, Organization: []string{"TestOrg"}},
		NotBefore:    time.Now().Add(-1 * time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, caCert, &leafKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}

	leafPEM := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER}))
	return leafPEM, leafKey
}

// generateLeafCertECDSAWithEKU is generateLeafCertECDSA but with an
// explicit ExtKeyUsage list, for tests that need to prove EKU-matching
// behavior specifically.
func generateLeafCertECDSAWithEKU(t *testing.T, caKey *ecdsa.PrivateKey, caPEM string, cn string, eku []x509.ExtKeyUsage) (string, *ecdsa.PrivateKey) {
	t.Helper()

	block, _ := pem.Decode([]byte(caPEM))
	caCert, _ := x509.ParseCertificate(block.Bytes)

	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(4),
		Subject:      pkix.Name{CommonName: cn, Organization: []string{"TestOrg"}},
		NotBefore:    time.Now().Add(-1 * time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		ExtKeyUsage:  eku,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, caCert, &leafKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}

	leafPEM := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER}))
	return leafPEM, leafKey
}

// generateLeafCertRSA is the RSA counterpart of generateLeafCertECDSA, used
// to prove verifyCertificateChallenge's RSA branch works end to end, not
// just ECDSA.
func generateLeafCertRSA(t *testing.T, caKey *ecdsa.PrivateKey, caPEM string, cn string) (string, *rsa.PrivateKey) {
	t.Helper()

	block, _ := pem.Decode([]byte(caPEM))
	caCert, _ := x509.ParseCertificate(block.Bytes)

	leafKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(3),
		Subject:      pkix.Name{CommonName: cn, Organization: []string{"TestOrg"}},
		NotBefore:    time.Now().Add(-1 * time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, caCert, &leafKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}

	leafPEM := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER}))
	return leafPEM, leafKey
}

// leafCertDER extracts the raw DER bytes from a PEM-encoded certificate,
// needed to build the same canonical challenge buildCertificateChallenge
// derives from evidence.CertificateAttestation.CertificatePEM.
func leafCertDER(t *testing.T, certPEM string) []byte {
	t.Helper()
	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		t.Fatal("no PEM block in test certificate")
	}
	return block.Bytes
}

func signChallengeECDSA(t *testing.T, key *ecdsa.PrivateKey, certDER []byte, timestamp int64) []byte {
	t.Helper()
	digest := sha256.Sum256(buildCertificateChallenge(certDER, timestamp))
	sig, err := ecdsa.SignASN1(rand.Reader, key, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return sig
}

func signChallengeRSA(t *testing.T, key *rsa.PrivateKey, certDER []byte, timestamp int64) []byte {
	t.Helper()
	digest := sha256.Sum256(buildCertificateChallenge(certDER, timestamp))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return sig
}

func TestCertificateValidator_ValidChainAndSignature_ECDSA_Attested(t *testing.T) {
	caKey, caPEM := generateCACert(t)
	leafPEM, leafKey := generateLeafCertECDSA(t, caKey, caPEM, "my-workload")
	timestamp := time.Now().UTC().Unix()
	sig := signChallengeECDSA(t, leafKey, leafCertDER(t, leafPEM), timestamp)

	cfg, _ := json.Marshal(certificateConfig{TrustAnchorPEM: caPEM})
	provider := &domain.TrustProvider{
		ID:         uuid.New(),
		Config:     cfg,
		MatchRules: []domain.MatchRule{{Attribute: "subject.cn", Value: "my-workload"}},
	}
	evidence := &domain.ClientEvidence{
		CertificateAttestation: &domain.CertificateAttestation{
			CertificatePEM: leafPEM,
			Timestamp:      timestamp,
			Signature:      sig,
		},
	}

	v := &CertificateValidator{}
	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Attested {
		t.Fatalf("expected attested, got reason: %s", result.Reason)
	}
}

func TestCertificateValidator_ValidChainAndSignature_RSA_Attested(t *testing.T) {
	caKey, caPEM := generateCACert(t)
	leafPEM, leafKey := generateLeafCertRSA(t, caKey, caPEM, "my-rsa-workload")
	timestamp := time.Now().UTC().Unix()
	sig := signChallengeRSA(t, leafKey, leafCertDER(t, leafPEM), timestamp)

	cfg, _ := json.Marshal(certificateConfig{TrustAnchorPEM: caPEM})
	provider := &domain.TrustProvider{ID: uuid.New(), Config: cfg}
	evidence := &domain.ClientEvidence{
		CertificateAttestation: &domain.CertificateAttestation{
			CertificatePEM: leafPEM,
			Timestamp:      timestamp,
			Signature:      sig,
		},
	}

	v := &CertificateValidator{}
	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Attested {
		t.Fatalf("expected attested (RSA proof-of-possession), got reason: %s", result.Reason)
	}
}

func TestCertificateValidator_WrongCA_InvalidSignature(t *testing.T) {
	caKey, caPEM := generateCACert(t)
	_, otherCAPEM := generateCACert(t)
	leafPEM, leafKey := generateLeafCertECDSA(t, caKey, caPEM, "my-workload")
	timestamp := time.Now().UTC().Unix()
	sig := signChallengeECDSA(t, leafKey, leafCertDER(t, leafPEM), timestamp)

	cfg, _ := json.Marshal(certificateConfig{TrustAnchorPEM: otherCAPEM})
	provider := &domain.TrustProvider{ID: uuid.New(), Config: cfg}
	evidence := &domain.ClientEvidence{
		CertificateAttestation: &domain.CertificateAttestation{
			CertificatePEM: leafPEM,
			Timestamp:      timestamp,
			Signature:      sig,
		},
	}

	v := &CertificateValidator{}
	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Attested {
		t.Fatal("expected denied")
	}
	if result.Reason != "InvalidSignature" {
		t.Fatalf("expected InvalidSignature, got %s", result.Reason)
	}
}

func TestCertificateValidator_NoProofOfPossession_ValidCertOnly_Rejected(t *testing.T) {
	// The core regression test for the CRITICAL finding this fix closes:
	// presenting a validly chain-verifiable certificate WITHOUT a valid
	// signature over it (e.g. an attacker who only obtained the public
	// certificate — logs, a TLS handshake capture, anywhere it's been
	// presented — but never had the private key) must not attest.
	caKey, caPEM := generateCACert(t)
	leafPEM, _ := generateLeafCertECDSA(t, caKey, caPEM, "my-workload")
	timestamp := time.Now().UTC().Unix()

	cfg, _ := json.Marshal(certificateConfig{TrustAnchorPEM: caPEM})
	provider := &domain.TrustProvider{ID: uuid.New(), Config: cfg}
	evidence := &domain.ClientEvidence{
		CertificateAttestation: &domain.CertificateAttestation{
			CertificatePEM: leafPEM,
			Timestamp:      timestamp,
			Signature:      []byte("not-a-real-signature-just-the-cert-was-obtained"),
		},
	}

	v := &CertificateValidator{}
	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Attested {
		t.Fatal("expected denied: a valid certificate with a bogus/missing signature must not prove possession")
	}
	if result.Reason != "InvalidSignature" {
		t.Fatalf("expected InvalidSignature, got %s", result.Reason)
	}
}

func TestCertificateValidator_SignedWithWrongKey_Rejected(t *testing.T) {
	// A different, unrelated private key signs the exact correct
	// challenge bytes for the real certificate — proves the check
	// actually verifies against THIS certificate's public key, not just
	// "some valid-looking signature."
	caKey, caPEM := generateCACert(t)
	leafPEM, _ := generateLeafCertECDSA(t, caKey, caPEM, "my-workload")
	timestamp := time.Now().UTC().Unix()

	_, wrongKey := generateLeafCertECDSA(t, caKey, caPEM, "attacker-owned-cert")
	sig := signChallengeECDSA(t, wrongKey, leafCertDER(t, leafPEM), timestamp)

	cfg, _ := json.Marshal(certificateConfig{TrustAnchorPEM: caPEM})
	provider := &domain.TrustProvider{ID: uuid.New(), Config: cfg}
	evidence := &domain.ClientEvidence{
		CertificateAttestation: &domain.CertificateAttestation{
			CertificatePEM: leafPEM,
			Timestamp:      timestamp,
			Signature:      sig,
		},
	}

	v := &CertificateValidator{}
	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Attested {
		t.Fatal("expected denied: signature from a different private key must not verify against this certificate's public key")
	}
	if result.Reason != "InvalidSignature" {
		t.Fatalf("expected InvalidSignature, got %s", result.Reason)
	}
}

func TestCertificateValidator_TamperedTimestamp_SignatureRejected(t *testing.T) {
	// The signature covers (certificate, timestamp) together — changing
	// the timestamp after signing (e.g. to extend an expired challenge's
	// apparent freshness) must invalidate the signature, since the
	// verifier reconstructs the challenge from the (now-tampered)
	// timestamp actually presented.
	caKey, caPEM := generateCACert(t)
	leafPEM, leafKey := generateLeafCertECDSA(t, caKey, caPEM, "my-workload")
	timestamp := time.Now().UTC().Unix()
	sig := signChallengeECDSA(t, leafKey, leafCertDER(t, leafPEM), timestamp)

	cfg, _ := json.Marshal(certificateConfig{TrustAnchorPEM: caPEM})
	provider := &domain.TrustProvider{ID: uuid.New(), Config: cfg}
	evidence := &domain.ClientEvidence{
		CertificateAttestation: &domain.CertificateAttestation{
			CertificatePEM: leafPEM,
			Timestamp:      timestamp + 100, // tampered after signing
			Signature:      sig,
		},
	}

	v := &CertificateValidator{}
	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Attested {
		t.Fatal("expected denied: signature must not verify against a tampered timestamp")
	}
	if result.Reason != "InvalidSignature" {
		t.Fatalf("expected InvalidSignature, got %s", result.Reason)
	}
}

func TestCertificateValidator_StaleTimestamp_AttestationExpired(t *testing.T) {
	caKey, caPEM := generateCACert(t)
	leafPEM, leafKey := generateLeafCertECDSA(t, caKey, caPEM, "my-workload")
	// Signed 10 minutes ago - beyond certificateMaxAge (5 min).
	timestamp := time.Now().UTC().Add(-10 * time.Minute).Unix()
	sig := signChallengeECDSA(t, leafKey, leafCertDER(t, leafPEM), timestamp)

	cfg, _ := json.Marshal(certificateConfig{TrustAnchorPEM: caPEM})
	provider := &domain.TrustProvider{ID: uuid.New(), Config: cfg}
	evidence := &domain.ClientEvidence{
		CertificateAttestation: &domain.CertificateAttestation{
			CertificatePEM: leafPEM,
			Timestamp:      timestamp,
			Signature:      sig,
		},
	}

	v := &CertificateValidator{}
	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Attested {
		t.Fatal("expected denied for a stale (replayed) signed challenge")
	}
	if result.Reason != "AttestationExpired" {
		t.Fatalf("expected AttestationExpired, got %s", result.Reason)
	}
}

func TestCertificateValidator_FutureTimestamp_BeyondSkew_NotYetValid(t *testing.T) {
	caKey, caPEM := generateCACert(t)
	leafPEM, leafKey := generateLeafCertECDSA(t, caKey, caPEM, "my-workload")
	// 10 minutes in the future - beyond certificateClockSkew (180s).
	timestamp := time.Now().UTC().Add(10 * time.Minute).Unix()
	sig := signChallengeECDSA(t, leafKey, leafCertDER(t, leafPEM), timestamp)

	cfg, _ := json.Marshal(certificateConfig{TrustAnchorPEM: caPEM})
	provider := &domain.TrustProvider{ID: uuid.New(), Config: cfg}
	evidence := &domain.ClientEvidence{
		CertificateAttestation: &domain.CertificateAttestation{
			CertificatePEM: leafPEM,
			Timestamp:      timestamp,
			Signature:      sig,
		},
	}

	v := &CertificateValidator{}
	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Attested {
		t.Fatal("expected denied for a timestamp too far in the future")
	}
	if result.Reason != "AttestationNotYetValid" {
		t.Fatalf("expected AttestationNotYetValid, got %s", result.Reason)
	}
}

func TestCertificateValidator_WithinClockSkew_Attested(t *testing.T) {
	caKey, caPEM := generateCACert(t)
	leafPEM, leafKey := generateLeafCertECDSA(t, caKey, caPEM, "my-workload")
	// 30s in the future - within certificateClockSkew (180s) tolerance.
	timestamp := time.Now().UTC().Add(30 * time.Second).Unix()
	sig := signChallengeECDSA(t, leafKey, leafCertDER(t, leafPEM), timestamp)

	cfg, _ := json.Marshal(certificateConfig{TrustAnchorPEM: caPEM})
	provider := &domain.TrustProvider{ID: uuid.New(), Config: cfg}
	evidence := &domain.ClientEvidence{
		CertificateAttestation: &domain.CertificateAttestation{
			CertificatePEM: leafPEM,
			Timestamp:      timestamp,
			Signature:      sig,
		},
	}

	v := &CertificateValidator{}
	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Attested {
		t.Fatalf("expected attested within clock skew tolerance, got reason: %s", result.Reason)
	}
}

func TestCertificateValidator_ClientAuthOnlyEKU_Attested(t *testing.T) {
	// Regression test for a real bug found during review: cert.Verify's
	// zero-value KeyUsages defaults to requiring ExtKeyUsageServerAuth,
	// which would reject exactly the certificate shape this validator
	// exists to accept — a workload identity certificate scoped to
	// ExtKeyUsageClientAuth only (this codebase's own X.509-SVID issuer,
	// credentials/x509_svid.go's buildEKUs, defaults to exactly this EKU
	// for workload certs). Before the fix, this exact certificate shape
	// would have been rejected with InvalidSignature despite a
	// completely valid chain and signature.
	caKey, caPEM := generateCACert(t)
	leafPEM, leafKey := generateLeafCertECDSAWithEKU(t, caKey, caPEM, "my-workload", []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth})
	timestamp := time.Now().UTC().Unix()
	sig := signChallengeECDSA(t, leafKey, leafCertDER(t, leafPEM), timestamp)

	cfg, _ := json.Marshal(certificateConfig{TrustAnchorPEM: caPEM})
	provider := &domain.TrustProvider{ID: uuid.New(), Config: cfg}
	evidence := &domain.ClientEvidence{
		CertificateAttestation: &domain.CertificateAttestation{
			CertificatePEM: leafPEM,
			Timestamp:      timestamp,
			Signature:      sig,
		},
	}

	v := &CertificateValidator{}
	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Attested {
		t.Fatalf("expected attested=true for a ClientAuth-only EKU certificate, got reason: %s", result.Reason)
	}
}

func TestCertificateValidator_ServerAuthOnlyEKU_Rejected(t *testing.T) {
	// The other direction: a certificate scoped ONLY to ServerAuth (a
	// TLS server certificate, not a client/workload identity
	// certificate) is a genuine purpose mismatch and must still be
	// rejected — proving the EKU fix isn't simply "accept everything."
	caKey, caPEM := generateCACert(t)
	leafPEM, leafKey := generateLeafCertECDSAWithEKU(t, caKey, caPEM, "a-tls-server-not-a-workload", []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth})
	timestamp := time.Now().UTC().Unix()
	sig := signChallengeECDSA(t, leafKey, leafCertDER(t, leafPEM), timestamp)

	cfg, _ := json.Marshal(certificateConfig{TrustAnchorPEM: caPEM})
	provider := &domain.TrustProvider{ID: uuid.New(), Config: cfg}
	evidence := &domain.ClientEvidence{
		CertificateAttestation: &domain.CertificateAttestation{
			CertificatePEM: leafPEM,
			Timestamp:      timestamp,
			Signature:      sig,
		},
	}

	v := &CertificateValidator{}
	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Attested {
		t.Fatal("expected attested=false for a ServerAuth-only certificate: not a valid client/workload identity credential")
	}
	if result.Reason != "InvalidSignature" {
		t.Fatalf("expected InvalidSignature, got %s", result.Reason)
	}
}

func TestCertificateValidator_MatchRuleFailed_WrongCN(t *testing.T) {
	caKey, caPEM := generateCACert(t)
	leafPEM, leafKey := generateLeafCertECDSA(t, caKey, caPEM, "other-workload")
	timestamp := time.Now().UTC().Unix()
	sig := signChallengeECDSA(t, leafKey, leafCertDER(t, leafPEM), timestamp)

	cfg, _ := json.Marshal(certificateConfig{TrustAnchorPEM: caPEM})
	provider := &domain.TrustProvider{
		ID:         uuid.New(),
		Config:     cfg,
		MatchRules: []domain.MatchRule{{Attribute: "subject.cn", Value: "my-workload"}},
	}
	evidence := &domain.ClientEvidence{
		CertificateAttestation: &domain.CertificateAttestation{
			CertificatePEM: leafPEM,
			Timestamp:      timestamp,
			Signature:      sig,
		},
	}

	v := &CertificateValidator{}
	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Attested {
		t.Fatal("expected denied")
	}
	if result.Reason != "MatchRuleFailed" {
		t.Fatalf("expected MatchRuleFailed, got %s", result.Reason)
	}
	if result.Attribute != "subject.cn" {
		t.Fatalf("expected attribute subject.cn, got %s", result.Attribute)
	}
}

func TestCertificateValidator_NoEvidence_NoDataFound(t *testing.T) {
	v := &CertificateValidator{}
	result, err := v.Validate(context.Background(), &domain.TrustProvider{}, &domain.ClientEvidence{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Reason != "NoDataFound" {
		t.Fatalf("expected NoDataFound, got %s", result.Reason)
	}
}

func TestCertificateValidator_NoMatchRules_ChainOnly(t *testing.T) {
	caKey, caPEM := generateCACert(t)
	leafPEM, leafKey := generateLeafCertECDSA(t, caKey, caPEM, "anything")
	timestamp := time.Now().UTC().Unix()
	sig := signChallengeECDSA(t, leafKey, leafCertDER(t, leafPEM), timestamp)

	cfg, _ := json.Marshal(certificateConfig{TrustAnchorPEM: caPEM})
	provider := &domain.TrustProvider{ID: uuid.New(), Config: cfg}
	evidence := &domain.ClientEvidence{
		CertificateAttestation: &domain.CertificateAttestation{
			CertificatePEM: leafPEM,
			Timestamp:      timestamp,
			Signature:      sig,
		},
	}

	v := &CertificateValidator{}
	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Attested {
		t.Fatalf("expected attested with no match rules, got reason: %s", result.Reason)
	}
	if result.Claims == nil {
		t.Fatal("expected Claims to be populated even when no match rules are configured")
	}
	if result.Claims["subject.cn"] != "anything" {
		t.Errorf("Claims[subject.cn] = %v, want anything", result.Claims["subject.cn"])
	}
}

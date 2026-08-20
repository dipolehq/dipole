package trust

import (
	"crypto/x509"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/beevik/etree"
	dsig "github.com/russellhaering/goxmldsig"
)

func TestAssertionExpired_WithinWindow_NotExpired(t *testing.T) {
	now := time.Now().UTC()
	notBefore := now.Add(-5 * time.Minute)
	notOnOrAfter := now.Add(5 * time.Minute)

	if assertionExpired(notBefore, notOnOrAfter) {
		t.Error("expected assertion within its validity window to not be expired")
	}
}

func TestAssertionExpired_AfterNotOnOrAfter_Expired(t *testing.T) {
	now := time.Now().UTC()
	notBefore := now.Add(-2 * time.Hour)
	notOnOrAfter := now.Add(-1 * time.Hour)

	if !assertionExpired(notBefore, notOnOrAfter) {
		t.Error("expected assertion past NotOnOrAfter (beyond clock skew) to be expired")
	}
}

func TestAssertionExpired_BeforeNotBefore_Expired(t *testing.T) {
	now := time.Now().UTC()
	notBefore := now.Add(1 * time.Hour)
	notOnOrAfter := now.Add(2 * time.Hour)

	if !assertionExpired(notBefore, notOnOrAfter) {
		t.Error("expected assertion not yet valid (beyond clock skew) to be expired")
	}
}

func TestAssertionExpired_WithinClockSkewOfNotOnOrAfter_NotExpired(t *testing.T) {
	now := time.Now().UTC()
	notBefore := now.Add(-1 * time.Hour)
	notOnOrAfter := now.Add(-30 * time.Second)

	if assertionExpired(notBefore, notOnOrAfter) {
		t.Error("expected assertion just past NotOnOrAfter but within clock skew tolerance to not be expired")
	}
}

func TestAssertionExpired_WithinClockSkewOfNotBefore_NotExpired(t *testing.T) {
	now := time.Now().UTC()
	notBefore := now.Add(30 * time.Second)
	notOnOrAfter := now.Add(1 * time.Hour)

	if assertionExpired(notBefore, notOnOrAfter) {
		t.Error("expected assertion just before NotBefore but within clock skew tolerance to not be expired")
	}
}

func TestAssertionExpired_ZeroValueTimes_NotEnforced(t *testing.T) {
	if assertionExpired(time.Time{}, time.Time{}) {
		t.Error("expected zero-value NotBefore/NotOnOrAfter (absent from assertion) to not be treated as expired")
	}
}

// --- Signed-XML test fixtures ---
//
// These build real SAML Response XML, signed with goxmldsig's own
// SigningContext (the same library saml_validator.go verifies with),
// rather than hand-written XML strings — so what's under test is
// exactly the signature scoping/binding logic, not incidental XML
// formatting.

// samlKeyStore adapts dsig.RandomKeyStoreForTest()'s generated RSA
// keypair + self-signed cert into a usable (*rsa.PrivateKey,
// *x509.Certificate) pair for building a signing context and a matching
// IdP certificate for the validator's cert store.
func samlKeyStore(t *testing.T) (dsig.X509KeyStore, *x509.Certificate) {
	t.Helper()
	ks := dsig.RandomKeyStoreForTest()
	_, certDER, err := ks.GetKeyPair()
	if err != nil {
		t.Fatalf("get keypair: %v", err)
	}
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		t.Fatalf("parse self-signed cert: %v", err)
	}
	return ks, cert
}

// buildAssertionXML returns an unsigned <Assertion> element with the
// given ID, subject NameID, and validity window.
func buildAssertionXML(id, nameID string, notBefore, notOnOrAfter time.Time) *etree.Element {
	assertion := etree.NewElement("Assertion")
	assertion.CreateAttr("xmlns", "urn:oasis:names:tc:SAML:2.0:assertion")
	assertion.CreateAttr("ID", id)
	assertion.CreateAttr("Version", "2.0")
	assertion.CreateAttr("IssueInstant", time.Now().UTC().Format(time.RFC3339))

	issuer := assertion.CreateElement("Issuer")
	issuer.SetText("https://idp.example.com")

	subject := assertion.CreateElement("Subject")
	nameIDEl := subject.CreateElement("NameID")
	nameIDEl.SetText(nameID)

	conditions := assertion.CreateElement("Conditions")
	conditions.CreateAttr("NotBefore", notBefore.UTC().Format(time.RFC3339))
	conditions.CreateAttr("NotOnOrAfter", notOnOrAfter.UTC().Format(time.RFC3339))

	return assertion
}

// signElement signs el as an enveloped signature using ks, returning the
// signed copy (el plus an appended <Signature> child).
func signElement(t *testing.T, el *etree.Element, ks dsig.X509KeyStore) *etree.Element {
	t.Helper()
	signingCtx := dsig.NewDefaultSigningContext(ks)
	if err := signingCtx.SetSignatureMethod(dsig.RSASHA256SignatureMethod); err != nil {
		t.Fatalf("set signature method: %v", err)
	}
	signed, err := signingCtx.SignEnveloped(el)
	if err != nil {
		t.Fatalf("sign element: %v", err)
	}
	return signed
}

// wrapInResponse builds a <samlp:Response> containing the given
// assertion element(s), base64-encoding the whole thing the same way a
// real IdP's POST-bound SAMLResponse would arrive.
func wrapInResponse(t *testing.T, responseSig *etree.Element, assertions ...*etree.Element) string {
	t.Helper()
	doc := etree.NewDocument()
	response := doc.CreateElement("Response")
	response.CreateAttr("xmlns", "urn:oasis:names:tc:SAML:2.0:protocol")
	response.CreateAttr("ID", "_response1")
	response.CreateAttr("Version", "2.0")
	response.CreateAttr("IssueInstant", time.Now().UTC().Format(time.RFC3339))
	response.CreateAttr("Destination", "https://sp.example.com/acs")

	if responseSig != nil {
		response.AddChild(responseSig)
	}
	for _, a := range assertions {
		response.AddChild(a)
	}

	xmlBytes, err := doc.WriteToBytes()
	if err != nil {
		t.Fatalf("serialize response: %v", err)
	}
	return base64.StdEncoding.EncodeToString(xmlBytes)
}

func TestSAMLValidator_ValidAssertionSignature_Attested(t *testing.T) {
	ks, idpCert := samlKeyStore(t)
	now := time.Now()
	assertion := buildAssertionXML("_assertion1", "user@example.com", now.Add(-5*time.Minute), now.Add(5*time.Minute))
	signedAssertion := signElement(t, assertion, ks)

	responseB64 := wrapInResponse(t, nil, signedAssertion)

	v := &SAMLValidator{}
	claims, expired, err := v.validateSAMLResponse(responseB64, idpCert)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if expired {
		t.Fatal("expected assertion to not be expired")
	}
	if claims["sub"] != "user@example.com" {
		t.Errorf("expected sub claim 'user@example.com', got %v", claims["sub"])
	}
}

func TestSAMLValidator_ValidResponseLevelSignature_Attested(t *testing.T) {
	// Some IdPs sign the whole Response instead of (or in addition to)
	// the Assertion. The Response's own Signature must be accepted too.
	ks, idpCert := samlKeyStore(t)
	now := time.Now()
	assertion := buildAssertionXML("_assertion1", "user@example.com", now.Add(-5*time.Minute), now.Add(5*time.Minute))

	// Build the response first (unsigned), then sign the whole thing.
	doc := etree.NewDocument()
	response := doc.CreateElement("Response")
	response.CreateAttr("xmlns", "urn:oasis:names:tc:SAML:2.0:protocol")
	response.CreateAttr("ID", "_response1")
	response.CreateAttr("Version", "2.0")
	response.CreateAttr("IssueInstant", time.Now().UTC().Format(time.RFC3339))
	response.CreateAttr("Destination", "https://sp.example.com/acs")
	response.AddChild(assertion)

	signedResponse := signElement(t, response, ks)
	xmlBytes, err := etree.NewDocumentWithRoot(signedResponse).WriteToBytes()
	if err != nil {
		t.Fatalf("serialize: %v", err)
	}
	responseB64 := base64.StdEncoding.EncodeToString(xmlBytes)

	v := &SAMLValidator{}
	claims, expired, err := v.validateSAMLResponse(responseB64, idpCert)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if expired {
		t.Fatal("expected assertion to not be expired")
	}
	if claims["sub"] != "user@example.com" {
		t.Errorf("expected sub claim 'user@example.com', got %v", claims["sub"])
	}
}

func TestSAMLValidator_TamperedAssertion_SignatureRejected(t *testing.T) {
	ks, idpCert := samlKeyStore(t)
	now := time.Now()
	assertion := buildAssertionXML("_assertion1", "user@example.com", now.Add(-5*time.Minute), now.Add(5*time.Minute))
	signedAssertion := signElement(t, assertion, ks)

	// Tamper with the subject after signing, exactly the attack a
	// signature check exists to catch: claims can't be modified
	// post-signing without detection.
	nameIDEl := signedAssertion.FindElement("./Subject/NameID")
	nameIDEl.SetText("attacker@evil.com")

	responseB64 := wrapInResponse(t, nil, signedAssertion)

	v := &SAMLValidator{}
	_, _, err := v.validateSAMLResponse(responseB64, idpCert)
	if err == nil {
		t.Fatal("expected signature verification to fail for a tampered assertion")
	}
}

func TestSAMLValidator_WrongIdPCertificate_SignatureRejected(t *testing.T) {
	ks, _ := samlKeyStore(t)
	_, wrongCert := samlKeyStore(t) // a different keypair/cert than the one that signed
	now := time.Now()
	assertion := buildAssertionXML("_assertion1", "user@example.com", now.Add(-5*time.Minute), now.Add(5*time.Minute))
	signedAssertion := signElement(t, assertion, ks)

	responseB64 := wrapInResponse(t, nil, signedAssertion)

	v := &SAMLValidator{}
	_, _, err := v.validateSAMLResponse(responseB64, wrongCert)
	if err == nil {
		t.Fatal("expected signature verification to fail when the configured IdP certificate doesn't match the signer")
	}
}

func TestSAMLValidator_NoSignature_Rejected(t *testing.T) {
	_, idpCert := samlKeyStore(t)
	now := time.Now()
	assertion := buildAssertionXML("_assertion1", "user@example.com", now.Add(-5*time.Minute), now.Add(5*time.Minute))

	responseB64 := wrapInResponse(t, nil, assertion) // never signed

	v := &SAMLValidator{}
	_, _, err := v.validateSAMLResponse(responseB64, idpCert)
	if err == nil {
		t.Fatal("expected an unsigned assertion to be rejected")
	}
}

func TestSAMLValidator_MultipleAssertions_RejectedOutright(t *testing.T) {
	// Proves the "exactly one Assertion" enforcement added as part of
	// the signature-binding hardening: a response carrying a
	// validly-signed legitimate assertion plus a second, unsigned
	// assertion with different claims must be rejected outright, not
	// silently resolved by picking one of the two.
	//
	// Framing note: this is defense-in-depth/correctness hardening, not
	// a reproduction of a proven exploit. An attempt to actually defeat
	// the pre-fix code with this same two-assertion shape (plus several
	// signature-reference variants) did not succeed — goxmldsig's
	// digest comparison (re-hashing whatever element it's asked to
	// validate and comparing against the signed digest) independently
	// caught every tampering shape tried, even though its *signature
	// search* alone is not element-scoped. This test still has value:
	// it removes reliance on that same implicit whole-document
	// canonicalization/digest behavior to provide the safety property,
	// replacing it with an explicit, auditable rule enforced in our own
	// code, and it prevents the specific double-parse inconsistency
	// (signature search finding one element, encoding/xml's "last one
	// wins" claims-extraction landing on a different one) that existed
	// before this fix regardless of whether it was independently
	// exploitable.
	ks, idpCert := samlKeyStore(t)
	now := time.Now()

	legitimate := buildAssertionXML("_assertion1", "real-user@example.com", now.Add(-5*time.Minute), now.Add(5*time.Minute))
	signedLegitimate := signElement(t, legitimate, ks)

	forged := buildAssertionXML("_assertion2", "attacker@evil.com", now.Add(-5*time.Minute), now.Add(5*time.Minute))
	// forged is deliberately never signed.

	responseB64 := wrapInResponse(t, nil, signedLegitimate, forged)

	v := &SAMLValidator{}
	_, _, err := v.validateSAMLResponse(responseB64, idpCert)
	if err == nil {
		t.Fatal("expected a response containing more than one Assertion to be rejected outright (XSW mitigation)")
	}
}

func TestSAMLValidator_ExpiredAssertion_ReportedExpired(t *testing.T) {
	ks, idpCert := samlKeyStore(t)
	now := time.Now()
	// Validity window closed well over an hour ago - outside clock skew.
	assertion := buildAssertionXML("_assertion1", "user@example.com", now.Add(-2*time.Hour), now.Add(-1*time.Hour))
	signedAssertion := signElement(t, assertion, ks)

	responseB64 := wrapInResponse(t, nil, signedAssertion)

	v := &SAMLValidator{}
	_, expired, err := v.validateSAMLResponse(responseB64, idpCert)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !expired {
		t.Fatal("expected assertion outside its validity window to be reported as expired")
	}
}

func TestNewSAMLValidator_SSRFProtected(t *testing.T) {
	v := NewSAMLValidator()
	transport, ok := v.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected httpClient.Transport to be *http.Transport, got %T", v.httpClient.Transport)
	}
	if transport.DialContext == nil {
		t.Fatal("expected DialContext to be set to the SSRF-safe dialer")
	}

	// End-to-end proof: a malicious samlMetadataUrl pointing at a
	// loopback address (the same class of target as internal
	// infrastructure or cloud metadata services) must be refused, not
	// just theoretically blocked by the dialer's presence.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, err := v.httpClient.Get(server.URL)
	if err == nil {
		t.Fatal("expected request to loopback address to be refused by SSRF-safe transport")
	}
}

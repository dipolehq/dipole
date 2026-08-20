package trust

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
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

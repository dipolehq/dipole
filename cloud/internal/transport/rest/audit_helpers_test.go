// audit_helpers_test.go tests clientIP's header precedence, specifically that CF-Connecting-IP (set authoritatively by Cloudflare, not spoofable by the client) is trusted over X-Forwarded-For (which the client can prepend an arbitrary value to).
package rest

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientIP_PrefersCFConnectingIPOverXForwardedFor(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("CF-Connecting-IP", "203.0.113.7")
	r.Header.Set("X-Forwarded-For", "6.6.6.6, 203.0.113.7")

	if got := clientIP(r); got != "203.0.113.7" {
		t.Errorf("clientIP() = %q, want %q (CF-Connecting-IP must win over a client-spoofable X-Forwarded-For)", got, "203.0.113.7")
	}
}

func TestClientIP_FallsBackToXForwardedForWhenNoCFHeader(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-Forwarded-For", "203.0.113.7, 10.0.0.1")

	if got := clientIP(r); got != "203.0.113.7" {
		t.Errorf("clientIP() = %q, want %q", got, "203.0.113.7")
	}
}

func TestClientIP_FallsBackToXRealIPWhenNoOtherHeaders(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-Real-IP", "203.0.113.7")

	if got := clientIP(r); got != "203.0.113.7" {
		t.Errorf("clientIP() = %q, want %q", got, "203.0.113.7")
	}
}

func TestClientIP_FallsBackToRemoteAddrWhenNoHeaders(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "203.0.113.7:54321"

	if got := clientIP(r); got != "203.0.113.7" {
		t.Errorf("clientIP() = %q, want %q", got, "203.0.113.7")
	}
}

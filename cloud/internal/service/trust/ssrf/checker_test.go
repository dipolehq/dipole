package ssrf

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestIsPubliclyRoutable(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{"public IPv4", "8.8.8.8", true},
		{"public IPv4 2", "1.1.1.1", true},
		{"loopback IPv4", "127.0.0.1", false},
		{"loopback IPv4 range", "127.255.255.255", false},
		{"unspecified IPv4", "0.0.0.0", false},
		{"link-local IPv4 (AWS/GCP metadata)", "169.254.169.254", false},
		{"private 10/8", "10.0.0.1", false},
		{"private 172.16/12", "172.16.0.1", false},
		{"private 192.168/16", "192.168.1.1", false},
		{"cgnat 100.64/10", "100.64.0.1", false},
		{"multicast", "224.0.0.1", false},
		{"loopback IPv6", "::1", false},
		{"unspecified IPv6", "::", false},
		{"link-local IPv6", "fe80::1", false},
		{"unique local IPv6 (ULA)", "fc00::1", false},
		{"public IPv6", "2606:4700:4700::1111", true},
		{"IPv4-mapped IPv6 loopback", "::ffff:127.0.0.1", false},
		{"IPv4-mapped IPv6 metadata", "::ffff:169.254.169.254", false},
		{"IPv4-mapped IPv6 public", "::ffff:8.8.8.8", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ip := net.ParseIP(tc.ip)
			if ip == nil {
				t.Fatalf("failed to parse test IP %q", tc.ip)
			}
			got := IsPubliclyRoutable(ip)
			if got != tc.want {
				t.Errorf("IsPubliclyRoutable(%s) = %v, want %v", tc.ip, got, tc.want)
			}
		})
	}
}

func TestIsPubliclyRoutable_Nil(t *testing.T) {
	if IsPubliclyRoutable(nil) {
		t.Error("expected nil IP to be rejected")
	}
}

// TestSafeDialContext_BlocksLoopback verifies the dialer actually refuses a
// real connection attempt to a loopback address, exercising the Control
// hook end-to-end rather than just the pure IsPubliclyRoutable predicate.
func TestSafeDialContext_BlocksLoopback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	dialer := &net.Dialer{Timeout: 2 * time.Second}
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: SafeDialContext(dialer),
		},
		Timeout: 5 * time.Second,
	}

	// httptest.NewServer listens on 127.0.0.1, which must be blocked.
	_, err := client.Get(server.URL)
	if err == nil {
		t.Fatal("expected connection to loopback test server to be refused")
	}
	if !strings.Contains(err.Error(), "ssrf") {
		t.Errorf("expected ssrf-related error, got: %v", err)
	}
}

// TestControlFunc_AllowsPublicAddress verifies the Control hook permits a
// public address. A true end-to-end version of this test (dialing a real
// public address) isn't practical in a unit test without live network
// access, so this exercises controlFunc directly instead — the same
// function SafeDialContext installs as Dialer.Control.
func TestControlFunc_AllowsPublicAddress(t *testing.T) {
	if err := controlFunc("tcp", "8.8.8.8:443", nil); err != nil {
		t.Errorf("expected public address to be allowed, got error: %v", err)
	}
}

func TestControlFunc_BlocksMetadataAddress(t *testing.T) {
	if err := controlFunc("tcp", "169.254.169.254:443", nil); err == nil {
		t.Error("expected metadata address to be blocked")
	}
}

func TestControlFunc_RejectsUnresolvedHostname(t *testing.T) {
	// controlFunc must only ever receive a resolved literal IP:port from
	// the Go runtime's dialer. If it somehow receives a hostname (e.g. a
	// caller bug bypassing resolution), fail closed rather than silently
	// allowing the connection.
	if err := controlFunc("tcp", "example.com:443", nil); err == nil {
		t.Error("expected non-literal-IP address to be rejected")
	}
}

func TestSafeDialContext_ContextCancellation(t *testing.T) {
	dialer := &net.Dialer{Timeout: 2 * time.Second}
	dial := SafeDialContext(dialer)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := dial(ctx, "tcp", "8.8.8.8:443")
	if err == nil {
		t.Error("expected dial with cancelled context to fail")
	}
}

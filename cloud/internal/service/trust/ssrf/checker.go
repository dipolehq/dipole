/*
Package ssrf protects outbound HTTP requests made by the trust service
against Server-Side Request Forgery: a malicious or misconfigured Trust
Provider config (e.g. a K8s/OIDC "oidcEndpoint") pointing at cloud metadata
services (169.254.169.254), loopback, or internal-only networks.

Validation happens at dial time via net.Dialer.Control, after DNS
resolution has already produced a literal IP address — not by
string-matching the request hostname before resolution. Hostname-only
checks are bypassable by DNS rebinding: a hostname that looks public at
request-build time can resolve to a private IP by the time the connection
is actually made. Dial-time validation closes that gap because it inspects
the IP the OS is about to connect to, not the hostname the caller supplied.
*/
package ssrf

import (
	"context"
	"fmt"
	"net"
	"syscall"
)

// cgnatRange is RFC 6598 shared address space (100.64.0.0/10), used by some
// cloud providers for internal NAT/load-balancer infrastructure. Not
// covered by net.IP.IsPrivate, so it's checked explicitly.
var cgnatRange = mustParseCIDR("100.64.0.0/10")

func mustParseCIDR(s string) *net.IPNet {
	_, n, err := net.ParseCIDR(s)
	if err != nil {
		panic(fmt.Sprintf("ssrf: invalid CIDR %q: %v", s, err))
	}
	return n
}

// IsPubliclyRoutable reports whether ip is safe to make outbound requests
// to. It rejects loopback, unspecified, link-local (unicast and
// multicast), private RFC1918/ULA, multicast, and RFC 6598 CGNAT
// addresses. IPv4-mapped IPv6 addresses (e.g. ::ffff:127.0.0.1) are
// correctly unwrapped by the underlying net.IP methods before checking.
func IsPubliclyRoutable(ip net.IP) bool {
	if ip == nil {
		return false
	}
	switch {
	case ip.IsLoopback(),
		ip.IsUnspecified(),
		ip.IsLinkLocalUnicast(),
		ip.IsLinkLocalMulticast(),
		ip.IsPrivate(),
		ip.IsMulticast():
		return false
	case cgnatRange.Contains(ip):
		return false
	}
	return true
}

// SafeDialContext wraps dialer with a Control hook that rejects connections
// to any address that fails IsPubliclyRoutable. Intended for use as
// http.Transport.DialContext so every outbound request made through that
// transport — including ones reached via HTTP redirects — is protected,
// not just the first request.
func SafeDialContext(dialer *net.Dialer) func(ctx context.Context, network, addr string) (net.Conn, error) {
	d := *dialer
	d.Control = controlFunc
	return d.DialContext
}

func controlFunc(_, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("ssrf: parse resolved address %q: %w", address, err)
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return fmt.Errorf("ssrf: resolved address %q is not a literal IP", host)
	}

	if !IsPubliclyRoutable(ip) {
		return fmt.Errorf("ssrf: refusing outbound connection to non-public address %s", ip)
	}

	return nil
}

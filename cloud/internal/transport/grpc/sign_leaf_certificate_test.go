/*
sign_leaf_certificate_test.go tests the SignLeafCertificate RPC handler,
specifically the security-critical checks: verified-identity enforcement,
hostname authorization, TLS Decrypt enablement, and the happy-path
CSR-to-leaf-cert flow. These checks are what stand between a proxy and the
ability to mint trusted certificates for arbitrary hostnames or other
tenants. Caller identity itself (tenant_id/proxy_id) is established by the
session auth interceptor before this handler ever runs — see
session_auth_test.go for those tests — so these tests inject a
VerifiedProxy into the context directly, exactly as the interceptor would.
*/
package grpc

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/service/pki"
	pb "github.com/dipolehq/dipole/gen/dipole/v1"
)

// mockLeafSigner lets tests control SignCSR's outcome and whether TLS
// Decrypt is considered enabled, without needing a real tenant CA/repo.
type mockLeafSigner struct {
	tlsDecryptEnabled bool
	signErr           error
	signResp          *pki.SignCSRResponse
	lastReq           pki.SignCSRRequest
}

func (m *mockLeafSigner) SignCSR(_ context.Context, req pki.SignCSRRequest) (*pki.SignCSRResponse, error) {
	m.lastReq = req
	if m.signErr != nil {
		return nil, m.signErr
	}
	if m.signResp != nil {
		return m.signResp, nil
	}
	return &pki.SignCSRResponse{CertDER: []byte("fake-cert-der"), NotAfter: time.Now().Add(24 * time.Hour)}, nil
}

func (m *mockLeafSigner) IsTlsDecryptEnabled(_ context.Context, _ uuid.UUID) bool {
	return m.tlsDecryptEnabled
}

// mockHostnameAuthorizer lets tests control which hostnames are authorized.
type mockHostnameAuthorizer struct {
	authorized map[string]bool
	err        error
}

func (m *mockHostnameAuthorizer) IsHostnameAuthorizedForTLS(_ context.Context, _ uuid.UUID, hostname string) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	return m.authorized[hostname], nil
}

// generateTestCSR builds a real, validly self-signed CSR for the given
// hostname, matching what the edge proxy would produce.
func generateTestCSR(t *testing.T, hostname string) []byte {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	template := &x509.CertificateRequest{DNSNames: []string{hostname}}
	der, err := x509.CreateCertificateRequest(rand.Reader, template, key)
	if err != nil {
		t.Fatalf("create CSR: %v", err)
	}
	return der
}

// withVerifiedProxy returns a context carrying a VerifiedProxy identity,
// exactly as SessionAuthInterceptor would inject it after resolving a
// session token. Handler-level tests use this instead of going through the
// interceptor, since the interceptor itself is tested separately.
func withVerifiedProxy(ctx context.Context, tenantID, proxyID uuid.UUID) context.Context {
	return context.WithValue(ctx, verifiedProxyContextKey{}, VerifiedProxy{TenantID: tenantID, ProxyID: proxyID})
}

// testServerFixture bundles the mocks needed for a fully-configured
// PolicyEvaluationServer.SignLeafCertificate call.
type testServerFixture struct {
	server   *PolicyEvaluationServer
	signer   *mockLeafSigner
	hostAuth *mockHostnameAuthorizer
}

func newTestServerFixture() *testServerFixture {
	signer := &mockLeafSigner{tlsDecryptEnabled: true}
	hostAuth := &mockHostnameAuthorizer{authorized: map[string]bool{}}

	server := &PolicyEvaluationServer{
		leafSigner:    signer,
		hostnameAuthz: hostAuth,
		logger:        newTestLogger(),
	}

	return &testServerFixture{server: server, signer: signer, hostAuth: hostAuth}
}

func TestSignLeafCertificate_HappyPath(t *testing.T) {
	f := newTestServerFixture()
	tenantID := uuid.New()
	proxyID := uuid.New()
	f.hostAuth.authorized["api.stripe.com"] = true

	notAfter := time.Now().Add(24 * time.Hour)
	f.signer.signResp = &pki.SignCSRResponse{CertDER: []byte("signed-cert"), NotAfter: notAfter}

	ctx := withVerifiedProxy(context.Background(), tenantID, proxyID)
	resp, err := f.server.SignLeafCertificate(ctx, &pb.SignLeafRequest{
		CsrDer:   generateTestCSR(t, "api.stripe.com"),
		Hostname: "api.stripe.com",
		ProxyId:  proxyID.String(),
		TenantId: tenantID.String(),
	})
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got error: %s", resp.ErrorMessage)
	}
	if string(resp.CertDer) != "signed-cert" {
		t.Errorf("cert_der = %q, want %q", resp.CertDer, "signed-cert")
	}
	if resp.NotAfter != notAfter.Unix() {
		t.Errorf("not_after = %d, want %d", resp.NotAfter, notAfter.Unix())
	}
	if f.signer.lastReq.TenantID != tenantID {
		t.Errorf("SignCSR called with tenant %s, want %s", f.signer.lastReq.TenantID, tenantID)
	}
	if f.signer.lastReq.Hostname != "api.stripe.com" {
		t.Errorf("SignCSR called with hostname %q, want %q", f.signer.lastReq.Hostname, "api.stripe.com")
	}
}

// TestSignLeafCertificate_NoVerifiedIdentity_Denied confirms the handler
// fails closed if it somehow runs without a verified identity in context
// (e.g. a bug in interceptor wiring) — it must never fall back to trusting
// the request body's tenant_id/proxy_id fields on its own.
func TestSignLeafCertificate_NoVerifiedIdentity_Denied(t *testing.T) {
	f := newTestServerFixture()
	tenantID := uuid.New()
	f.hostAuth.authorized["api.stripe.com"] = true

	resp, err := f.server.SignLeafCertificate(context.Background(), &pb.SignLeafRequest{
		CsrDer:   generateTestCSR(t, "api.stripe.com"),
		Hostname: "api.stripe.com",
		ProxyId:  uuid.New().String(),
		TenantId: tenantID.String(),
	})
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if resp.Success {
		t.Fatal("expected denial when no verified identity is present in context")
	}
}

// TestSignLeafCertificate_RequestIdentityMismatch_Denied is the core
// security regression test for this fix: even if a caller's session token
// verifies it as tenant A / proxy X, claiming a *different* tenant_id or
// proxy_id in the request body must be rejected outright rather than
// silently ignored — a mismatch is treated as a possible spoofing attempt.
func TestSignLeafCertificate_RequestIdentityMismatch_Denied(t *testing.T) {
	f := newTestServerFixture()
	tenantID := uuid.New()
	proxyID := uuid.New()
	otherTenantID := uuid.New()
	f.hostAuth.authorized["victim.example.com"] = true

	ctx := withVerifiedProxy(context.Background(), tenantID, proxyID)
	resp, err := f.server.SignLeafCertificate(ctx, &pb.SignLeafRequest{
		CsrDer:   generateTestCSR(t, "victim.example.com"),
		Hostname: "victim.example.com",
		ProxyId:  proxyID.String(),
		TenantId: otherTenantID.String(), // does not match the verified session identity
	})
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if resp.Success {
		t.Fatal("expected denial when request tenant_id does not match verified session identity")
	}
	if f.signer.lastReq.Hostname != "" {
		t.Error("SignCSR must not be called when request identity does not match the verified session")
	}
}

func TestSignLeafCertificate_HostnameNotAuthorized_Denied(t *testing.T) {
	f := newTestServerFixture()
	tenantID := uuid.New()
	proxyID := uuid.New()
	// Deliberately do not authorize "internal.corp" — no active TLS policy for it.

	ctx := withVerifiedProxy(context.Background(), tenantID, proxyID)
	resp, _ := f.server.SignLeafCertificate(ctx, &pb.SignLeafRequest{
		CsrDer:   generateTestCSR(t, "internal.corp"),
		Hostname: "internal.corp",
		ProxyId:  proxyID.String(),
		TenantId: tenantID.String(),
	})
	if resp.Success {
		t.Fatal("expected denial for unauthorized hostname")
	}
	if f.signer.lastReq.Hostname != "" {
		t.Error("SignCSR must not be called when hostname is not authorized")
	}
}

func TestSignLeafCertificate_TLSDecryptNotEnabled_Denied(t *testing.T) {
	f := newTestServerFixture()
	f.signer.tlsDecryptEnabled = false
	tenantID := uuid.New()
	proxyID := uuid.New()
	f.hostAuth.authorized["api.stripe.com"] = true

	ctx := withVerifiedProxy(context.Background(), tenantID, proxyID)
	resp, _ := f.server.SignLeafCertificate(ctx, &pb.SignLeafRequest{
		CsrDer:   generateTestCSR(t, "api.stripe.com"),
		Hostname: "api.stripe.com",
		ProxyId:  proxyID.String(),
		TenantId: tenantID.String(),
	})
	if resp.Success {
		t.Fatal("expected denial when TLS Decrypt / tenant CA is not enabled")
	}
}

func TestSignLeafCertificate_SigningInfraNotConfigured_Denied(t *testing.T) {
	server := &PolicyEvaluationServer{logger: newTestLogger()} // no leafSigner/hostnameAuthz set
	ctx := withVerifiedProxy(context.Background(), uuid.New(), uuid.New())
	resp, err := server.SignLeafCertificate(ctx, &pb.SignLeafRequest{
		CsrDer:   generateTestCSR(t, "api.stripe.com"),
		Hostname: "api.stripe.com",
		ProxyId:  uuid.New().String(),
		TenantId: uuid.New().String(),
	})
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if resp.Success {
		t.Fatal("expected denial when signing infrastructure is not wired up")
	}
}

func TestSignLeafCertificate_SignCSRError_Denied(t *testing.T) {
	f := newTestServerFixture()
	tenantID := uuid.New()
	proxyID := uuid.New()
	f.hostAuth.authorized["api.stripe.com"] = true
	f.signer.signErr = fmt.Errorf("CSR SAN does not contain the requested hostname")

	ctx := withVerifiedProxy(context.Background(), tenantID, proxyID)
	resp, err := f.server.SignLeafCertificate(ctx, &pb.SignLeafRequest{
		CsrDer:   generateTestCSR(t, "api.stripe.com"),
		Hostname: "api.stripe.com",
		ProxyId:  proxyID.String(),
		TenantId: tenantID.String(),
	})
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if resp.Success {
		t.Fatal("expected denial when SignCSR returns an error")
	}
}

// TestSignLeafCertificate_SignCSRError_DoesNotLeakInternalErrorDetails
// guards against reintroducing an information-disclosure bug: internal
// error text from SignCSR (crypto, parsing, or database failures) must
// never be forwarded to the caller verbatim, no matter what the underlying
// error says.
func TestSignLeafCertificate_SignCSRError_DoesNotLeakInternalErrorDetails(t *testing.T) {
	f := newTestServerFixture()
	tenantID := uuid.New()
	proxyID := uuid.New()
	f.hostAuth.authorized["api.stripe.com"] = true

	sensitiveErr := fmt.Errorf("decrypt CA key: cipher: message authentication failed (tenant_id=%s)", tenantID)
	f.signer.signErr = sensitiveErr

	ctx := withVerifiedProxy(context.Background(), tenantID, proxyID)
	resp, err := f.server.SignLeafCertificate(ctx, &pb.SignLeafRequest{
		CsrDer:   generateTestCSR(t, "api.stripe.com"),
		Hostname: "api.stripe.com",
		ProxyId:  proxyID.String(),
		TenantId: tenantID.String(),
	})
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if resp.Success {
		t.Fatal("expected denial when SignCSR returns an error")
	}
	if strings.Contains(resp.ErrorMessage, "cipher") || strings.Contains(resp.ErrorMessage, tenantID.String()) {
		t.Fatalf("ErrorMessage leaked internal error details: %q", resp.ErrorMessage)
	}
	if resp.ErrorMessage != "certificate signing failed" {
		t.Errorf("ErrorMessage = %q, want a fixed generic message", resp.ErrorMessage)
	}
}

func TestSignLeafCertificate_MissingFields_Denied(t *testing.T) {
	f := newTestServerFixture()
	tenantID := uuid.New()
	proxyID := uuid.New()
	ctx := withVerifiedProxy(context.Background(), tenantID, proxyID)

	cases := []struct {
		name string
		req  *pb.SignLeafRequest
	}{
		{"empty hostname", &pb.SignLeafRequest{TenantId: tenantID.String(), Hostname: "", ProxyId: proxyID.String(), CsrDer: []byte("x")}},
		{"empty csr", &pb.SignLeafRequest{TenantId: tenantID.String(), Hostname: "x", ProxyId: proxyID.String(), CsrDer: nil}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := f.server.SignLeafCertificate(ctx, tc.req)
			if err != nil {
				t.Fatalf("unexpected transport error: %v", err)
			}
			if resp.Success {
				t.Fatalf("expected denial for %s", tc.name)
			}
		})
	}
}

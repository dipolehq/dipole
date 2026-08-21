// session_auth_test.go tests SessionAuthInterceptor and SessionAuthStreamInterceptor directly, covering token validation, identity injection, and method exemptions.
package grpc

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type mockSessionProxyAuthenticator struct {
	proxies map[string]*domain.AgentProxy
	err     error
}

func newMockSessionProxyAuthenticator() *mockSessionProxyAuthenticator {
	return &mockSessionProxyAuthenticator{proxies: make(map[string]*domain.AgentProxy)}
}

func (m *mockSessionProxyAuthenticator) register(token string, tenantID, proxyID uuid.UUID) {
	m.proxies[hashSessionToken(token)] = &domain.AgentProxy{ID: proxyID, TenantID: tenantID}
}

func (m *mockSessionProxyAuthenticator) GetBySessionTokenHash(_ context.Context, tokenHash string) (*domain.AgentProxy, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.proxies[tokenHash], nil
}

func withBearerMetadata(ctx context.Context, value string) context.Context {
	if value == "" {
		return metadata.NewIncomingContext(ctx, metadata.MD{})
	}
	return metadata.NewIncomingContext(ctx, metadata.Pairs("authorization", value))
}

func passthroughHandler(ctx context.Context, _ any) (any, error) {
	verified, ok := VerifiedProxyFromContext(ctx)
	if !ok {
		return "no-identity", nil
	}
	return verified, nil
}

func TestSessionAuthInterceptor_MissingMetadata_Unauthenticated(t *testing.T) {
	auth := newMockSessionProxyAuthenticator()
	interceptor := SessionAuthInterceptor(auth)

	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/dipole.v1.PolicyEvaluationService/Evaluate"}, passthroughHandler)
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got %v", err)
	}
}

func TestSessionAuthInterceptor_EmptyBearerToken_Unauthenticated(t *testing.T) {
	auth := newMockSessionProxyAuthenticator()
	interceptor := SessionAuthInterceptor(auth)

	ctx := withBearerMetadata(context.Background(), "Bearer ")
	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/dipole.v1.PolicyEvaluationService/Evaluate"}, passthroughHandler)
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got %v", err)
	}
}

func TestSessionAuthInterceptor_UnknownToken_Unauthenticated(t *testing.T) {
	auth := newMockSessionProxyAuthenticator()
	interceptor := SessionAuthInterceptor(auth)

	ctx := withBearerMetadata(context.Background(), "Bearer not-a-real-token")
	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/dipole.v1.PolicyEvaluationService/Evaluate"}, passthroughHandler)
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got %v", err)
	}
}

func TestSessionAuthInterceptor_RepositoryError_Internal(t *testing.T) {
	auth := &mockSessionProxyAuthenticator{err: context.DeadlineExceeded}
	interceptor := SessionAuthInterceptor(auth)

	ctx := withBearerMetadata(context.Background(), "Bearer some-token")
	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/dipole.v1.PolicyEvaluationService/Evaluate"}, passthroughHandler)
	if status.Code(err) != codes.Internal {
		t.Fatalf("expected Internal, got %v", err)
	}
}

func TestSessionAuthInterceptor_ValidToken_InjectsVerifiedIdentity(t *testing.T) {
	auth := newMockSessionProxyAuthenticator()
	tenantID := uuid.New()
	proxyID := uuid.New()
	auth.register("valid-token-123", tenantID, proxyID)
	interceptor := SessionAuthInterceptor(auth)

	ctx := withBearerMetadata(context.Background(), "Bearer valid-token-123")
	result, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/dipole.v1.PolicyEvaluationService/Evaluate"}, passthroughHandler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	verified, ok := result.(VerifiedProxy)
	if !ok {
		t.Fatalf("handler did not receive a verified identity: %v", result)
	}
	if verified.TenantID != tenantID || verified.ProxyID != proxyID {
		t.Errorf("verified = %+v, want tenant=%s proxy=%s", verified, tenantID, proxyID)
	}
}

func TestSessionAuthInterceptor_TokenCaseInsensitiveBearerPrefix(t *testing.T) {
	auth := newMockSessionProxyAuthenticator()
	tenantID := uuid.New()
	proxyID := uuid.New()
	auth.register("mixed-case-token", tenantID, proxyID)
	interceptor := SessionAuthInterceptor(auth)

	ctx := withBearerMetadata(context.Background(), "BEARER mixed-case-token")
	result, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/dipole.v1.PolicyEvaluationService/Evaluate"}, passthroughHandler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := result.(VerifiedProxy); !ok {
		t.Fatalf("handler did not receive a verified identity: %v", result)
	}
}

func TestSessionAuthInterceptor_RegisterIsExempt(t *testing.T) {
	auth := newMockSessionProxyAuthenticator()
	interceptor := SessionAuthInterceptor(auth)

	result, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/dipole.v1.PolicyEvaluationService/Register"}, passthroughHandler)
	if err != nil {
		t.Fatalf("unexpected error for exempt method: %v", err)
	}
	if result != "no-identity" {
		t.Errorf("expected handler to run without a verified identity for Register, got %v", result)
	}
}

type fakeServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (f *fakeServerStream) Context() context.Context { return f.ctx }

func TestSessionAuthStreamInterceptor_MissingToken_Unauthenticated(t *testing.T) {
	auth := newMockSessionProxyAuthenticator()
	interceptor := SessionAuthStreamInterceptor(auth)

	stream := &fakeServerStream{ctx: context.Background()}
	err := interceptor(nil, stream, &grpc.StreamServerInfo{FullMethod: "/dipole.v1.PolicyEvaluationService/IngestEvents"}, func(any, grpc.ServerStream) error {
		t.Fatal("handler must not run without a valid token")
		return nil
	})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got %v", err)
	}
}

func TestSessionAuthStreamInterceptor_ValidToken_InjectsVerifiedIdentityIntoStreamContext(t *testing.T) {
	auth := newMockSessionProxyAuthenticator()
	tenantID := uuid.New()
	proxyID := uuid.New()
	auth.register("stream-token", tenantID, proxyID)
	interceptor := SessionAuthStreamInterceptor(auth)

	stream := &fakeServerStream{ctx: withBearerMetadata(context.Background(), "Bearer stream-token")}

	var gotVerified VerifiedProxy
	var gotOK bool
	err := interceptor(nil, stream, &grpc.StreamServerInfo{FullMethod: "/dipole.v1.PolicyEvaluationService/IngestEvents"}, func(_ any, ss grpc.ServerStream) error {
		gotVerified, gotOK = VerifiedProxyFromContext(ss.Context())
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !gotOK {
		t.Fatal("handler's stream context did not carry a verified identity")
	}
	if gotVerified.TenantID != tenantID || gotVerified.ProxyID != proxyID {
		t.Errorf("verified = %+v, want tenant=%s proxy=%s", gotVerified, tenantID, proxyID)
	}
}

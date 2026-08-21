// session_auth.go authenticates every Agent Proxy gRPC call except Register and the health check, verifying the bearer session token and injecting the resulting verified identity into the request context.
package grpc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

const sessionAuthMetadataKey = "authorization"

var sessionAuthMethodExemptions = map[string]bool{
	"/dipole.v1.PolicyEvaluationService/Register": true,
	"/grpc.health.v1.Health/Check":                true,
}

type SessionProxyAuthenticator interface {
	GetBySessionTokenHash(ctx context.Context, tokenHash string) (*domain.AgentProxy, error)
}

type verifiedProxyContextKey struct{}

type VerifiedProxy struct {
	ProxyID  uuid.UUID
	TenantID uuid.UUID
}

func VerifiedProxyFromContext(ctx context.Context) (VerifiedProxy, bool) {
	v, ok := ctx.Value(verifiedProxyContextKey{}).(VerifiedProxy)
	return v, ok
}

func hashSessionToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func authenticateSessionToken(ctx context.Context, auth SessionProxyAuthenticator) (VerifiedProxy, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return VerifiedProxy{}, status.Error(codes.Unauthenticated, "missing session token")
	}

	values := md.Get(sessionAuthMetadataKey)
	if len(values) == 0 || values[0] == "" {
		return VerifiedProxy{}, status.Error(codes.Unauthenticated, "missing session token")
	}

	token := values[0]
	const bearerPrefix = "bearer "
	if len(token) > len(bearerPrefix) && strings.EqualFold(token[:len(bearerPrefix)], bearerPrefix) {
		token = token[len(bearerPrefix):]
	}
	if token == "" {
		return VerifiedProxy{}, status.Error(codes.Unauthenticated, "missing session token")
	}

	proxy, err := auth.GetBySessionTokenHash(ctx, hashSessionToken(token))
	if err != nil {
		return VerifiedProxy{}, status.Error(codes.Internal, "session authentication failed")
	}
	if proxy == nil {
		return VerifiedProxy{}, status.Error(codes.Unauthenticated, "invalid session token")
	}

	return VerifiedProxy{ProxyID: proxy.ID, TenantID: proxy.TenantID}, nil
}

func SessionAuthInterceptor(auth SessionProxyAuthenticator) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if sessionAuthMethodExemptions[info.FullMethod] {
			return handler(ctx, req)
		}

		verified, err := authenticateSessionToken(ctx, auth)
		if err != nil {
			return nil, err
		}

		return handler(context.WithValue(ctx, verifiedProxyContextKey{}, verified), req)
	}
}

func SessionAuthStreamInterceptor(auth SessionProxyAuthenticator) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if sessionAuthMethodExemptions[info.FullMethod] {
			return handler(srv, ss)
		}

		verified, err := authenticateSessionToken(ss.Context(), auth)
		if err != nil {
			return err
		}

		return handler(srv, &verifiedServerStream{
			ServerStream: ss,
			ctx:          context.WithValue(ss.Context(), verifiedProxyContextKey{}, verified),
		})
	}
}

type verifiedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *verifiedServerStream) Context() context.Context {
	return s.ctx
}

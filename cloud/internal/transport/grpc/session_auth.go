/*
session_auth.go authenticates every Agent Proxy gRPC call except Register.

Register is the only RPC that establishes trust from scratch (via a
single-use registration token) and issues the caller a bearer session
token in return. Every other RPC on this service must present that session
token as gRPC metadata; SessionAuthInterceptor (unary) and
SessionAuthStreamInterceptor (streaming, for IngestEvents) verify it against
the proxy it was issued to and inject the verified identity into the
request context. Handlers must read tenant_id/proxy_id from that verified
identity (via VerifiedProxyFromContext) rather than trusting the
corresponding fields in the request body, which are unauthenticated
client-supplied data.
*/
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

// sessionAuthMetadataKey is the gRPC metadata key carrying the bearer
// session token. gRPC lowercases metadata keys, so lookups must use the
// lowercase form regardless of how the client sets it.
const sessionAuthMetadataKey = "authorization"

// sessionAuthMethodExemptions lists RPCs that authenticate themselves
// through their own mechanism instead of the session token, because they
// run before any session token exists. Register is the only one: it
// verifies a single-use registration token and is what issues the session
// token in the first place.
var sessionAuthMethodExemptions = map[string]bool{
	"/dipole.v1.PolicyEvaluationService/Register": true,
}

// SessionProxyAuthenticator resolves a proxy by its session token hash.
// Implemented by repository.AgentProxyRepository.
type SessionProxyAuthenticator interface {
	GetBySessionTokenHash(ctx context.Context, tokenHash string) (*domain.AgentProxy, error)
}

type verifiedProxyContextKey struct{}

// VerifiedProxy is the caller identity established by SessionAuthInterceptor
// / SessionAuthStreamInterceptor. Handlers must use this — never the
// tenant_id/proxy_id fields on the request message — as the source of
// truth for which proxy and tenant is making the call.
type VerifiedProxy struct {
	ProxyID  uuid.UUID
	TenantID uuid.UUID
}

// VerifiedProxyFromContext returns the caller identity the auth
// interceptor verified for this RPC. ok is false if no interceptor ran
// (e.g. this RPC is exempt, or authentication was somehow bypassed) —
// handlers on non-exempt RPCs must treat ok=false as unauthenticated and
// deny the request rather than falling back to request-body fields.
func VerifiedProxyFromContext(ctx context.Context) (VerifiedProxy, bool) {
	v, ok := ctx.Value(verifiedProxyContextKey{}).(VerifiedProxy)
	return v, ok
}

func hashSessionToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// authenticateSessionToken extracts the bearer token from ctx's incoming
// metadata, resolves it to a proxy, and returns the verified identity. It
// never falls back to any other signal: a missing or unresolvable token is
// always Unauthenticated.
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

// SessionAuthInterceptor is a unary server interceptor enforcing session
// token authentication on every RPC except sessionAuthMethodExemptions.
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

// SessionAuthStreamInterceptor is the streaming counterpart of
// SessionAuthInterceptor, needed for IngestEvents (a client-streaming RPC).
// It wraps the stream's context so the handler's stream.Context() call
// returns the context carrying the verified identity.
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

// verifiedServerStream wraps grpc.ServerStream to override Context() with
// one carrying the verified caller identity, since ServerStream has no
// setter for its context.
type verifiedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *verifiedServerStream) Context() context.Context {
	return s.ctx
}

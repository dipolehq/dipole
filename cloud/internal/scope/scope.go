// Package scope provides request-scoped tenant and resource set isolation.
package scope

import (
	"context"

	"github.com/google/uuid"
)

var DefaultResourceSetID = uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff")

type RequestScope struct {
	TenantID      uuid.UUID
	ResourceSetID uuid.UUID
}

type contextKey struct{}

func WithScope(ctx context.Context, s *RequestScope) context.Context {
	if s == nil {
		panic("scope: WithScope called with nil RequestScope")
	}
	if s.TenantID == uuid.Nil {
		panic("scope: RequestScope.TenantID cannot be nil")
	}
	return context.WithValue(ctx, contextKey{}, s)
}

func FromContext(ctx context.Context) *RequestScope {
	if s, ok := ctx.Value(contextKey{}).(*RequestScope); ok {
		return s
	}
	return nil
}

func TenantID(ctx context.Context) uuid.UUID {
	if s := FromContext(ctx); s != nil {
		return s.TenantID
	}
	return uuid.Nil
}

func ResourceSetID(ctx context.Context) uuid.UUID {
	if s := FromContext(ctx); s != nil {
		return s.ResourceSetID
	}
	return uuid.Nil
}

func ResolveResourceSetID(provided *uuid.UUID) uuid.UUID {
	if provided != nil {
		return *provided
	}
	return DefaultResourceSetID
}

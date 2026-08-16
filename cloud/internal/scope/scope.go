/*
Package scope provides transport-independent request scoping for multi-tenant,
multi-resource-set isolation.

This package is the shared contract between the transport layer (which sets the
scope from HTTP headers, gRPC metadata, or background job config) and the
repository layer (which reads the scope to enforce SQL-level filtering).

Neither transport nor repository imports the other — both import this package.

Architecture:

	Request → Transport middleware → scope.WithScope(ctx, ...) → Service → Repository → scope.FromContext(ctx) → SQL WHERE clause
*/
package scope

import (
	"context"

	"github.com/google/uuid"
)

// DefaultResourceSetID is the well-known UUID for the default resource set.
// Every tenant has this resource set, and entities without explicit assignment
// belong here.
var DefaultResourceSetID = uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff")

// RequestScope carries the tenant and resource set identity for the current
// operation. It is set by the transport layer and consumed by the repository
// layer for query scoping.
type RequestScope struct {
	TenantID      uuid.UUID
	ResourceSetID uuid.UUID // Always set for authenticated requests; equals DefaultResourceSetID when no explicit selection.
}

type contextKey struct{}

// WithScope attaches a RequestScope to the context.
func WithScope(ctx context.Context, s *RequestScope) context.Context {
	return context.WithValue(ctx, contextKey{}, s)
}

func FromContext(ctx context.Context) *RequestScope {
	if s, ok := ctx.Value(contextKey{}).(*RequestScope); ok {
		return s
	}
	return nil
}

// ResolveResourceSetID returns the effective resource set for a value that may
// be nil (unassigned). Every resource-scoped row must carry a non-nil resource
// set id — this is the single place that nil collapses to DefaultResourceSetID.
func ResolveResourceSetID(provided *uuid.UUID) uuid.UUID {
	if provided != nil {
		return *provided
	}
	return DefaultResourceSetID
}

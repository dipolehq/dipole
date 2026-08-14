/*
scope_filter.go provides the shared SQL filter helper for resource-set scoping.

Every resource-scoped repository List/Count method calls ResourceSetFilter to
append the appropriate WHERE clause. The filter reads the RequestScope from the
context (set by transport-layer middleware) and returns:

  - A SQL fragment like " AND resource_set_id = $3" plus the appended arg, OR
  - An empty string with no arg change (when scope is nil — internal/system calls).

This keeps the scoping logic in one place. New repositories or queries get
resource-set isolation by calling this single function — no per-handler wiring
needed.

Import path: github.com/dipolehq/dipole/cloud/internal/repository/postgres
Dependency:  github.com/dipolehq/dipole/cloud/internal/scope (transport-independent)
*/
package postgres

import (
	"context"
	"fmt"

	"github.com/dipolehq/dipole/cloud/internal/scope"
)

// ResourceSetFilter appends a resource_set_id filter clause to the query
// if a RequestScope is present in the context.
//
// Parameters:
//   - ctx: request context (carries scope set by middleware)
//   - args: pointer to the existing query args slice (will be appended to)
//   - n: pointer to the current parameter counter (will be incremented)
//
// Returns a SQL fragment to append to the WHERE clause. Empty string means
// no filtering (internal/system call with no scope in context).
//
// Usage (for queries that build WHERE with string concatenation):
//
//	where := "tenant_id = $1" + ResourceSetFilter(ctx, &args, &n)
func ResourceSetFilter(ctx context.Context, args *[]interface{}, n *int) string {
	s := scope.FromContext(ctx)
	if s == nil {
		return ""
	}
	*n++
	*args = append(*args, s.ResourceSetID)
	return fmt.Sprintf(" AND resource_set_id = $%d", *n)
}

// appendResourceSetFilter appends a "resource_set_id = $N" clause to a clauses
// slice if a RequestScope is present. Used by event filter builders that collect
// clauses in a []string and join them with " AND ".
func appendResourceSetFilter(ctx context.Context, clauses *[]string, args *[]interface{}, n *int) {
	s := scope.FromContext(ctx)
	if s == nil {
		return
	}
	*n++
	*args = append(*args, s.ResourceSetID)
	*clauses = append(*clauses, fmt.Sprintf("resource_set_id = $%d", *n))
}

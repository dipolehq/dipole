// scope_filter.go provides the shared SQL filter helper for resource-set scoping, used by every resource-scoped repository's List/Count methods.
package postgres

import (
	"context"
	"fmt"

	"github.com/dipolehq/dipole/cloud/internal/scope"
)

func ResourceSetFilter(ctx context.Context, args *[]interface{}, n *int) string {
	s := scope.FromContext(ctx)
	if s == nil {
		return ""
	}
	*n++
	*args = append(*args, s.ResourceSetID)
	return fmt.Sprintf(" AND resource_set_id = $%d", *n)
}

func appendResourceSetFilter(ctx context.Context, clauses *[]string, args *[]interface{}, n *int) {
	s := scope.FromContext(ctx)
	if s == nil {
		return
	}
	*n++
	*args = append(*args, s.ResourceSetID)
	*clauses = append(*clauses, fmt.Sprintf("resource_set_id = $%d", *n))
}

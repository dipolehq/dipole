// list_helpers.go provides shared WHERE-clause and pagination builders used by every tenant- and resource-set-scoped List/Count method in this package.
package postgres

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

func buildTenantSearchFilter(tenantID uuid.UUID, search string) (where string, args []interface{}, n int) {
	args = []interface{}{tenantID, escapeILIKE(search)}
	where = "tenant_id = $1 AND ($2 = '' OR name ILIKE '%' || $2 || '%' OR description ILIKE '%' || $2 || '%')"
	return where, args, 2
}

func buildTenantFilter(tenantID uuid.UUID) (where string, args []interface{}, n int) {
	return "tenant_id = $1", []interface{}{tenantID}, 1
}

func appendLimitOffset(args *[]interface{}, n *int, perPage, offset int) string {
	*n++
	limitParam := fmt.Sprintf("$%d", *n)
	*args = append(*args, perPage)
	*n++
	offsetParam := fmt.Sprintf("$%d", *n)
	*args = append(*args, offset)
	return fmt.Sprintf("LIMIT %s OFFSET %s", limitParam, offsetParam)
}

func escapeILIKE(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

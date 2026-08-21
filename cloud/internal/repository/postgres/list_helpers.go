// list_helpers.go provides shared WHERE-clause and pagination builders used by every tenant- and resource-set-scoped List/Count method in this package.
package postgres

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// buildTenantSearchFilter returns the WHERE-clause fragment and positional
// args for a query scoped by tenant_id ($1) and an optional case-insensitive
// name/description search term ($2). The search term is escaped via
// escapeILIKE so a literal %, _, or \ typed by a user is matched literally
// rather than treated as a SQL wildcard — consolidating this logic into one
// place also fixes an existing inconsistency where some repositories (e.g.
// access conditions, client workloads, credential providers) never escaped
// the search term while others (agent controllers, access policies) did.
//
// n is the count of positional parameters already consumed (2), ready to
// be passed to ResourceSetFilter or appendLimitOffset to continue
// numbering from $3 onward.
func buildTenantSearchFilter(tenantID uuid.UUID, search string) (where string, args []interface{}, n int) {
	args = []interface{}{tenantID, escapeILIKE(search)}
	where = "tenant_id = $1 AND ($2 = '' OR name ILIKE '%' || $2 || '%' OR description ILIKE '%' || $2 || '%')"
	return where, args, 2
}

// buildTenantFilter is the search-less counterpart of buildTenantSearchFilter,
// for repositories whose List has no free-text search.
func buildTenantFilter(tenantID uuid.UUID) (where string, args []interface{}, n int) {
	return "tenant_id = $1", []interface{}{tenantID}, 1
}

// appendLimitOffset appends perPage and offset to args (continuing the
// positional-parameter count from *n) and returns the "LIMIT $n OFFSET $m"
// SQL fragment to append after a query's ORDER BY clause.
func appendLimitOffset(args *[]interface{}, n *int, perPage, offset int) string {
	*n++
	limitParam := fmt.Sprintf("$%d", *n)
	*args = append(*args, perPage)
	*n++
	offsetParam := fmt.Sprintf("$%d", *n)
	*args = append(*args, offset)
	return fmt.Sprintf("LIMIT %s OFFSET %s", limitParam, offsetParam)
}

// escapeILIKE escapes special ILIKE pattern characters (\, %, _) in a
// search term so they are matched literally rather than as SQL wildcards.
func escapeILIKE(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

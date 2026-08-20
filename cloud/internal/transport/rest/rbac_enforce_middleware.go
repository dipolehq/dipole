/*
Package rest — rbac_enforce_middleware.go enforces the permissions
RBACPermissionsMiddleware (rbac_middleware.go) already resolved into
context, denying any request whose resource/method the caller's role
does not permit.

Split from RBACPermissionsMiddleware deliberately: that middleware's job
is "resolve what this request can do" (a pure read of role state), this
middleware's job is "decide whether THIS request is allowed" (a policy
decision over that state). Keeping them separate means the resolution
logic (and its DB calls) stays independently testable from the
enforcement decision, and either could change without touching the
other.
*/
package rest

import (
	"net/http"
	"strings"

	"github.com/dipolehq/dipole/cloud/internal/service"
)

// selfServicePathPrefixes lists request paths that are exempt from
// resource-type permission checks because they act on the CALLER'S OWN
// account or session (profile, password, MFA enrollment, personal API
// tokens, tenant switching, session refresh, product feedback) — scoped
// by the caller's own identity from the validated auth token (see
// mfa_handler.go, profile.go, auth_handler.go: every handler here
// derives the acting identity from UserEmailFromContext/
// TenantIDFromContext, never from a path parameter or role check), not
// by any role-gated resource. A role's Users/Roles/etc. permissions
// govern managing OTHER users' accounts and access — they have no
// bearing on whether someone may view/edit their own profile, switch
// which of their own tenants is active, refresh their own session, or
// send feedback, all of which every authenticated user must always be
// able to do regardless of role.
//
// This is a deliberate, explicit allowlist, not an omission: any new
// self-service endpoint must be added here consciously. Everything not
// listed here is expected to resolve to a real ResourceRegistry entry
// and be checked normally; RBACEnforcementMiddleware fails closed (see
// its own doc comment) for any path that resolves to no resource and
// isn't in this allowlist, rather than silently allowing it.
var selfServicePathPrefixes = []string{
	"/api/v1/profile",            // GET/PUT /profile, POST /profile/token, GET /profile/tokens, DELETE /profile/tokens/{id}, POST /profile/password, and all /profile/mfa/* routes
	"/api/v1/auth/switch-tenant", // switches the caller's own active tenant context; scoped by the caller's own validated identity, not a role-gated resource
	"/api/v1/auth/refresh",       // refreshes the caller's own session token
	"/api/v1/user/tenants",       // lists tenants the CALLER (by validated email) belongs to
	"/api/v1/feedback",           // product feedback widget; every authenticated user regardless of role must be able to send feedback
}

// noResourceExemptPaths lists exact request paths that are exempt
// because they don't correspond to any single manageable resource type
// at all — a cross-resource summary view, not an entity with its own
// CRUD or read/write permission to hold. Exact-match, not prefix, since
// unlike selfServicePathPrefixes these aren't a whole sub-tree of
// self-scoped actions, just a couple of standalone views.
var noResourceExemptPaths = map[string]bool{
	"/api/v1/dashboard":          true, // cross-resource summary counts, not any one manageable entity
	"/api/v1/events/mcp-tracing": true, // MCP session tracing view, not currently modeled as its own RBAC resource
}

func isRBACExempt(path string) bool {
	if noResourceExemptPaths[path] {
		return true
	}
	for _, prefix := range selfServicePathPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// RBACEnforcementMiddleware denies any request whose resolved resource
// (via service.ResourceFromPath) is not permitted, at the caller's
// current permission level (via PermissionsFromContext + CheckPermission),
// for the request's HTTP method.
//
// Must run after RBACPermissionsMiddleware (needs permissions already in
// context) and, transitively, after AuthMiddleware.
//
// Fail-closed policy for paths matching no known resource: rather than
// silently allowing an unrecognized path through (which would make
// "forget to register a new resource in ResourceRegistry" a silent
// security hole instead of a loud, visible one), any path that is not in
// the explicit exemption allowlist above AND does not resolve to a
// ResourceRegistry entry is denied with a 403. Legitimate new endpoints
// must be deliberately exempted or registered — there is no default-allow
// path through this middleware.
func RBACEnforcementMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isRBACExempt(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		resource := service.ResourceFromPath(r.URL.Path)
		if resource == "" {
			requiredPermission := "read"
			if service.IsWriteMethod(r.Method) {
				requiredPermission = "write"
			}
			respondDomainError(w, &service.ForbiddenError{Resource: "unknown", RequiredPermission: requiredPermission})
			return
		}

		permissions := PermissionsFromContext(r.Context())
		if err := service.NewRBACService().CheckPermission(permissions, resource, r.Method); err != nil {
			respondDomainError(w, err)
			return
		}

		next.ServeHTTP(w, r)
	})
}

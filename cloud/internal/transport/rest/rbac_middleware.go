/*
Package rest — rbac_middleware.go loads the authenticated request's
effective RBAC permissions and stores them in context for downstream
authorization checks (see service.CheckPermission, PermissionsFromContext).

This middleware only loads permissions; it does not enforce them. Denying
a request based on the loaded permissions (checking them against the
request's resource/method) is a separate, later concern — this stays a
single-responsibility step: resolve "what can this request do", not "is
this request allowed to do the thing it's doing".
*/
package rest

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/dipolehq/dipole/cloud/internal/repository"
	"github.com/dipolehq/dipole/cloud/internal/service"
	"github.com/google/uuid"
)

// RBACPermissionsMiddleware resolves the authenticated user's Role and its
// Permissions and stores them in context via PermissionsFromContext.
//
// Must run after AuthMiddleware (needs tenantID/email already resolved).
// Does not depend on ResourceSetMiddleware/scope — Permissions gate which
// *resource types* a role may act on (e.g. "trust_providers": read/write),
// which is orthogonal to *which resource set* a request is scoped to.
// Role-to-ResourceSet assignment (RoleRepository.GetResourceSetIDs) is a
// separate, existing mechanism for data-scoping and is untouched here.
//
// Tenant isolation: both lookups below (UserRepository.GetByEmail,
// RoleRepository.GetByID) are tenant-scoped by their own signatures using
// the tenantID AuthMiddleware already resolved from the validated token —
// never a client-supplied value — so a role can only ever be loaded from
// the same tenant the request authenticated into.
//
// Fail-closed semantics: every branch below that cannot establish a
// concrete, active, well-formed set of permissions for this request sets
// an empty (nil) Permissions map, not a missing one and not "allow all".
// service.CheckPermission already treats any resource absent from the
// map as forbidden, so an empty map denies every check by construction —
// there is no separate "denied" state to get wrong.
func RBACPermissionsMiddleware(users repository.UserRepository, roles repository.RoleRepository, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantID := TenantIDFromContext(r.Context())
			email := UserEmailFromContext(r.Context())

			permissions := resolvePermissions(r.Context(), users, roles, tenantID, email, logger)

			ctx := context.WithValue(r.Context(), ctxPermissions, permissions)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// tenantWideAccessEmailPrefix identifies the synthetic email values
// APITokenAuthenticator.ValidateToken assigns to API-token-authenticated
// and legacy-tenant-token-authenticated requests (see
// cloud/internal/service/auth/auth.go: "apitoken@scoped",
// "apitoken@"+subdomain). Neither corresponds to a row in the users
// table — these bearer tokens authenticate as the tenant itself, not as
// a specific person, matching domain.APIToken's own documented model
// ("Tokens with nil ResourceSetID have full tenant-wide access").
//
// Deliberate design decision, not a fallback-of-convenience: these paths
// get full tenant-wide (Super Admin-equivalent) permissions, consistent
// with the access level these tokens already had before RBAC enforcement
// existed, and consistent with how domain.APIToken already documents its
// own access model. This does not weaken anything that was previously
// enforced (nothing was enforced on these paths before), and it does not
// grant cross-tenant access (the permissions are still evaluated only
// against the resources of the token's own tenant, via every handler's
// existing tenantID-scoped repository calls).
const tenantWideAccessEmailPrefix = "apitoken@"

// resolvePermissions is the pure resolution logic behind
// RBACPermissionsMiddleware, kept separate from the http.Handler wrapper
// so it can be unit tested directly with fake repositories rather than
// through httptest request/response plumbing.
func resolvePermissions(ctx context.Context, users repository.UserRepository, roles repository.RoleRepository, tenantID uuid.UUID, email string, logger *slog.Logger) service.Permissions {
	if tenantID == uuid.Nil || email == "" {
		return nil
	}

	if isTenantWideAccessIdentity(email) {
		return service.SuperAdminPermissions()
	}

	user, err := users.GetByEmail(ctx, tenantID, email)
	if err != nil {
		logger.Warn("rbac: failed to load user for permission resolution", "error", err)
		return nil
	}
	if user == nil || !user.IsActive {
		return nil
	}
	if user.RoleID == nil {
		return nil
	}

	role, err := roles.GetByID(ctx, tenantID, *user.RoleID)
	if err != nil {
		logger.Warn("rbac: failed to load role for permission resolution", "error", err)
		return nil
	}
	if role == nil || !role.IsActive {
		return nil
	}

	permissions, err := service.ParsePermissions(role.Permissions)
	if err != nil || permissions == nil {
		logger.Warn("rbac: role has empty or unparseable permissions; denying by default",
			"role_id", role.ID, "role_name", role.Name)
		return nil
	}

	return permissions
}

func isTenantWideAccessIdentity(email string) bool {
	return len(email) >= len(tenantWideAccessEmailPrefix) && email[:len(tenantWideAccessEmailPrefix)] == tenantWideAccessEmailPrefix
}

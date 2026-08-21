// rbac_middleware.go loads the authenticated request's effective RBAC permissions into context.
package rest

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/dipolehq/dipole/cloud/internal/repository"
	"github.com/dipolehq/dipole/cloud/internal/service"
	"github.com/google/uuid"
)

func RBACPermissionsMiddleware(users repository.UserRepository, roles repository.RoleRepository, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantID := TenantIDFromContext(r.Context())
			email := UserEmailFromContext(r.Context())

			permissions, userID := resolvePermissions(r.Context(), users, roles, tenantID, email, logger)

			ctx := context.WithValue(r.Context(), ctxPermissions, permissions)
			if userID != nil {
				ctx = context.WithValue(ctx, ctxUserID, *userID)
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

const tenantWideAccessEmailPrefix = "apitoken@"

// resolvePermissions resolves the caller's effective RBAC permissions and,
// as a side effect of the same user lookup, the acting user's ID — so
// callers needing the user's identity for audit attribution (see
// UserIDFromContext) don't need a second GetByEmail query per request.
func resolvePermissions(ctx context.Context, users repository.UserRepository, roles repository.RoleRepository, tenantID uuid.UUID, email string, logger *slog.Logger) (service.Permissions, *uuid.UUID) {
	if tenantID == uuid.Nil || email == "" {
		return nil, nil
	}

	if isTenantWideAccessIdentity(email) {
		return service.SuperAdminPermissions(), nil
	}

	user, err := users.GetByEmail(ctx, tenantID, email)
	if err != nil {
		logger.Warn("rbac: failed to load user for permission resolution", "error", err)
		return nil, nil
	}
	if user == nil || !user.IsActive {
		return nil, nil
	}
	if user.RoleID == nil {
		return nil, &user.ID
	}

	role, err := roles.GetByID(ctx, tenantID, *user.RoleID)
	if err != nil {
		logger.Warn("rbac: failed to load role for permission resolution", "error", err)
		return nil, &user.ID
	}
	if role == nil || !role.IsActive {
		return nil, &user.ID
	}

	permissions, err := service.ParsePermissions(role.Permissions)
	if err != nil || permissions == nil {
		logger.Warn("rbac: role has empty or unparseable permissions; denying by default",
			"role_id", role.ID, "role_name", role.Name)
		return nil, &user.ID
	}

	return permissions, &user.ID
}

func isTenantWideAccessIdentity(email string) bool {
	return len(email) >= len(tenantWideAccessEmailPrefix) && email[:len(tenantWideAccessEmailPrefix)] == tenantWideAccessEmailPrefix
}

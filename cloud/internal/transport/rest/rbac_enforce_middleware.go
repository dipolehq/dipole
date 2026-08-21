// rbac_enforce_middleware.go denies requests whose resolved resource is not permitted by the caller's RBAC role.
package rest

import (
	"net/http"
	"strings"

	"github.com/dipolehq/dipole/cloud/internal/service"
)

var selfServicePathPrefixes = []string{
	"/api/v1/profile",
	"/api/v1/auth/switch-tenant",
	"/api/v1/auth/refresh",
	"/api/v1/user/tenants",
	"/api/v1/feedback",
}

var noResourceExemptPaths = map[string]bool{
	"/api/v1/dashboard": true,
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

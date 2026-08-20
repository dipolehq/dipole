package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/service"
)

// --- Fake APIAuthenticator, for exercising the real ApplyMiddleware chain ---

type fakeAuthenticatorForRBAC struct {
	tenantID      uuid.UUID
	email         string
	resourceSetID *uuid.UUID
}

func (f *fakeAuthenticatorForRBAC) ValidateToken(_ context.Context, token string) (uuid.UUID, string, *uuid.UUID, error) {
	if token != "valid-token" {
		return uuid.Nil, "", nil, ErrForbiddenTestSentinel
	}
	return f.tenantID, f.email, f.resourceSetID, nil
}

// ErrForbiddenTestSentinel is just a distinguishable error for the fake
// authenticator's rejection path; AuthMiddleware only checks err != nil.
var ErrForbiddenTestSentinel = &service.ForbiddenError{Resource: "auth", RequiredPermission: "read"}

// buildTestMuxAndMiddleware wires one real endpoint per HTTP method
// against a fixed resource path, through the ACTUAL ApplyMiddleware
// chain (RBACPermissionsMiddleware + RBACEnforcementMiddleware + auth +
// resource-set + rate limiting), rather than calling a handler directly
// — proving the full request pipeline enforces permissions end to end,
// not just the pure CheckPermission/resolvePermissions functions in
// isolation.
func buildTestMuxAndMiddleware(t *testing.T, tenantID, roleID uuid.UUID, email string, permissions service.Permissions) (http.Handler, *fakeUserRepoForRBAC, *fakeRoleRepoForRBAC) {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/trust-providers", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("POST /api/v1/trust-providers", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	mux.HandleFunc("GET /api/v1/profile", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /api/v1/events/audit", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	users := newFakeUserRepoForRBAC()
	if permissions != nil {
		users.put(tenantID, &domain.User{Email: email, IsActive: true, RoleID: &roleID})
	}
	roles := newFakeRoleRepoForRBAC()
	if permissions != nil {
		permsJSON, err := json.Marshal(permissions)
		if err != nil {
			t.Fatalf("marshal permissions: %v", err)
		}
		roles.put(tenantID, &domain.Role{ID: roleID, IsActive: true, Permissions: permsJSON})
	}

	authenticator := &fakeAuthenticatorForRBAC{tenantID: tenantID, email: email}

	handler := ApplyMiddleware(mux, authenticator, nil, "", users, roles, testLoggerForRBAC())
	return handler, users, roles
}

func TestRBACEnforcement_ReadOnlyRole_GetAllowed(t *testing.T) {
	tenantID, roleID := uuid.New(), uuid.New()
	handler, _, _ := buildTestMuxAndMiddleware(t, tenantID, roleID, "alice@example.com", service.Permissions{"trust_providers": service.ReadOnly})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/trust-providers", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET with ReadOnly: status = %d, want 200", w.Code)
	}
}

func TestRBACEnforcement_ReadOnlyRole_PostForbidden(t *testing.T) {
	tenantID, roleID := uuid.New(), uuid.New()
	handler, _, _ := buildTestMuxAndMiddleware(t, tenantID, roleID, "alice@example.com", service.Permissions{"trust_providers": service.ReadOnly})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/trust-providers", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("POST with ReadOnly: status = %d, want 403", w.Code)
	}
	var resp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Error.Code != "forbidden" {
		t.Errorf("Error.Code = %q, want %q", resp.Error.Code, "forbidden")
	}
}

func TestRBACEnforcement_NoAccessRole_GetForbidden(t *testing.T) {
	tenantID, roleID := uuid.New(), uuid.New()
	handler, _, _ := buildTestMuxAndMiddleware(t, tenantID, roleID, "alice@example.com", service.Permissions{"trust_providers": service.NoAccess})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/trust-providers", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("GET with NoAccess: status = %d, want 403", w.Code)
	}
}

func TestRBACEnforcement_ReadWriteRole_PostAllowed(t *testing.T) {
	tenantID, roleID := uuid.New(), uuid.New()
	handler, _, _ := buildTestMuxAndMiddleware(t, tenantID, roleID, "alice@example.com", service.Permissions{"trust_providers": service.ReadWrite})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/trust-providers", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("POST with ReadWrite: status = %d, want 201", w.Code)
	}
}

func TestRBACEnforcement_NoRoleAssigned_EveryRequestForbidden(t *testing.T) {
	// permissions == nil simulates a user with no role, or a role whose
	// permissions failed to parse (see resolvePermissions) - the most
	// important fail-closed case to prove through the REAL middleware
	// chain, not just the pure function.
	tenantID, roleID := uuid.New(), uuid.New()
	handler, _, _ := buildTestMuxAndMiddleware(t, tenantID, roleID, "alice@example.com", nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/trust-providers", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("GET with no role/permissions: status = %d, want 403", w.Code)
	}
}

func TestRBACEnforcement_SelfServiceProfile_AlwaysAllowed_EvenWithNoAccessEverywhere(t *testing.T) {
	// /api/v1/profile must remain reachable regardless of the caller's
	// resource permissions - it's exempted, not resource-checked.
	tenantID, roleID := uuid.New(), uuid.New()
	handler, _, _ := buildTestMuxAndMiddleware(t, tenantID, roleID, "alice@example.com", service.Permissions{}) // empty: NoAccess on everything

	req := httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/profile with no resource permissions: status = %d, want 200 (self-service must always be reachable)", w.Code)
	}
}

func TestRBACEnforcement_SelfServiceSessionAndFeedbackPaths_AlwaysAllowed(t *testing.T) {
	// Regression coverage for a real gap found in the initial Phase 3
	// implementation: /auth/switch-tenant, /auth/refresh, /user/tenants,
	// and /feedback are all reachable by every authenticated user on
	// ordinary pages (tenant switcher, session refresh, feedback widget)
	// today, and none of them correspond to a manageable resource type —
	// they act on the caller's own session/identity, same class as
	// /profile. Before this exemption existed, fail-closed enforcement
	// would have 403'd them for every user, including Super Admin.
	tenantID, roleID := uuid.New(), uuid.New()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/auth/switch-tenant", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("POST /api/v1/auth/refresh", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("GET /api/v1/user/tenants", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("POST /api/v1/feedback", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	users := newFakeUserRepoForRBAC()
	users.put(tenantID, &domain.User{Email: "alice@example.com", IsActive: true, RoleID: &roleID})
	roles := newFakeRoleRepoForRBAC()
	roles.put(tenantID, &domain.Role{ID: roleID, IsActive: true, Permissions: mustMarshal(service.Permissions{})}) // NoAccess on everything

	authenticator := &fakeAuthenticatorForRBAC{tenantID: tenantID, email: "alice@example.com"}
	handler := ApplyMiddleware(mux, authenticator, nil, "", users, roles, testLoggerForRBAC())

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/auth/switch-tenant"},
		{http.MethodPost, "/api/v1/auth/refresh"},
		{http.MethodGet, "/api/v1/user/tenants"},
		{http.MethodPost, "/api/v1/feedback"},
	}
	for _, c := range cases {
		req := httptest.NewRequest(c.method, c.path, nil)
		req.Header.Set("Authorization", "Bearer valid-token")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("%s %s with NoAccess everywhere: status = %d, want 200 (self-service session/feedback paths must always be reachable)", c.method, c.path, w.Code)
		}
	}
}

func TestRBACEnforcement_EventResource_ReadOnlyPermissionAllowsGet(t *testing.T) {
	tenantID, roleID := uuid.New(), uuid.New()
	handler, _, _ := buildTestMuxAndMiddleware(t, tenantID, roleID, "alice@example.com", service.Permissions{"audit_logs": service.ReadOnly})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events/audit", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/events/audit with audit_logs=ReadOnly: status = %d, want 200", w.Code)
	}
}

func TestRBACEnforcement_TenantIsolation_EndToEnd(t *testing.T) {
	// The same email exists in two different tenants with different
	// permission levels on the same resource. Requests authenticated
	// into tenant A must be governed by tenant A's role, never tenant
	// B's, proven through the REAL middleware chain (not just the
	// resolvePermissions unit test in rbac_middleware_test.go) — this is
	// the standing tenant-isolation requirement applied to the actual
	// enforcement path Phase 3 adds.
	tenantA, tenantB := uuid.New(), uuid.New()
	roleA, roleB := uuid.New(), uuid.New()
	const sharedEmail = "shared@example.com"

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/trust-providers", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	users := newFakeUserRepoForRBAC()
	users.put(tenantA, &domain.User{Email: sharedEmail, IsActive: true, RoleID: &roleA})
	users.put(tenantB, &domain.User{Email: sharedEmail, IsActive: true, RoleID: &roleB})

	roles := newFakeRoleRepoForRBAC()
	roles.put(tenantA, &domain.Role{ID: roleA, IsActive: true, Permissions: mustMarshal(service.Permissions{"trust_providers": service.ReadWrite})})
	roles.put(tenantB, &domain.Role{ID: roleB, IsActive: true, Permissions: mustMarshal(service.Permissions{"trust_providers": service.ReadOnly})})

	// A request that authenticates into tenant A (ReadWrite) must succeed.
	authA := &fakeAuthenticatorForRBAC{tenantID: tenantA, email: sharedEmail}
	handlerA := ApplyMiddleware(mux, authA, nil, "", users, roles, testLoggerForRBAC())
	reqA := httptest.NewRequest(http.MethodPost, "/api/v1/trust-providers", nil)
	reqA.Header.Set("Authorization", "Bearer valid-token")
	wA := httptest.NewRecorder()
	handlerA.ServeHTTP(wA, reqA)
	if wA.Code != http.StatusCreated {
		t.Errorf("tenant A (ReadWrite) POST: status = %d, want 201", wA.Code)
	}

	// The identical request, authenticated into tenant B (ReadOnly for
	// the SAME resource, via the SAME shared email) must be forbidden —
	// proving tenant B's permissions were used, not tenant A's leaking
	// across.
	authB := &fakeAuthenticatorForRBAC{tenantID: tenantB, email: sharedEmail}
	handlerB := ApplyMiddleware(mux, authB, nil, "", users, roles, testLoggerForRBAC())
	reqB := httptest.NewRequest(http.MethodPost, "/api/v1/trust-providers", nil)
	reqB.Header.Set("Authorization", "Bearer valid-token")
	wB := httptest.NewRecorder()
	handlerB.ServeHTTP(wB, reqB)
	if wB.Code != http.StatusForbidden {
		t.Errorf("tenant B (ReadOnly) POST: status = %d, want 403 — must not inherit tenant A's ReadWrite permission", wB.Code)
	}
}

func TestRBACEnforcement_EveryRegistryResource_DeniedWithoutPermission(t *testing.T) {
	// A table-driven sweep proving CheckPermission is actually reached
	// (and denies) for every resource in service.ResourceRegistry, using
	// their real registered route paths — not just the one resource
	// (trust_providers) exercised by the tests above. Uses a permissive
	// mux (any method on any path succeeds) so a 200 here would only
	// happen if enforcement failed to run, distinguishing "denied by
	// RBAC" from "handler doesn't exist".
	for _, res := range service.ResourceRegistry {
		for _, segment := range res.PathSegments {
			t.Run(res.ID+"/"+segment, func(t *testing.T) {
				tenantID, roleID := uuid.New(), uuid.New()

				mux := http.NewServeMux()
				path := "/api/v1/" + segment
				mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				})

				users := newFakeUserRepoForRBAC()
				users.put(tenantID, &domain.User{Email: "alice@example.com", IsActive: true, RoleID: &roleID})
				roles := newFakeRoleRepoForRBAC()
				// Deliberately grant every OTHER resource ReadWrite but
				// leave THIS one absent from the map entirely, so a
				// pass here can only mean this specific resource's
				// absence was correctly treated as NoAccess.
				perms := service.Permissions{}
				for _, other := range service.ResourceRegistry {
					if other.ID != res.ID {
						perms[other.ID] = service.ReadWrite
					}
				}
				roles.put(tenantID, &domain.Role{ID: roleID, IsActive: true, Permissions: mustMarshal(perms)})

				authenticator := &fakeAuthenticatorForRBAC{tenantID: tenantID, email: "alice@example.com"}
				handler := ApplyMiddleware(mux, authenticator, nil, "", users, roles, testLoggerForRBAC())

				req := httptest.NewRequest(http.MethodGet, path, nil)
				req.Header.Set("Authorization", "Bearer valid-token")
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, req)

				if w.Code != http.StatusForbidden {
					t.Errorf("GET %s with %s absent from permissions: status = %d, want 403", path, res.ID, w.Code)
				}
			})
		}
	}
}

func TestRBACEnforcement_EveryRegistryResource_AllowedWithReadOnlyPermission(t *testing.T) {
	// The positive counterpart: granting exactly ReadOnly on each
	// resource (and nothing else) allows a GET to its real route.
	for _, res := range service.ResourceRegistry {
		for _, segment := range res.PathSegments {
			t.Run(res.ID+"/"+segment, func(t *testing.T) {
				tenantID, roleID := uuid.New(), uuid.New()

				mux := http.NewServeMux()
				path := "/api/v1/" + segment
				mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				})

				users := newFakeUserRepoForRBAC()
				users.put(tenantID, &domain.User{Email: "alice@example.com", IsActive: true, RoleID: &roleID})
				roles := newFakeRoleRepoForRBAC()
				roles.put(tenantID, &domain.Role{ID: roleID, IsActive: true, Permissions: mustMarshal(service.Permissions{res.ID: service.ReadOnly})})

				authenticator := &fakeAuthenticatorForRBAC{tenantID: tenantID, email: "alice@example.com"}
				handler := ApplyMiddleware(mux, authenticator, nil, "", users, roles, testLoggerForRBAC())

				req := httptest.NewRequest(http.MethodGet, path, nil)
				req.Header.Set("Authorization", "Bearer valid-token")
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, req)

				if w.Code != http.StatusOK {
					t.Errorf("GET %s with %s=ReadOnly: status = %d, want 200", path, res.ID, w.Code)
				}
			})
		}
	}
}

func TestRBACEnforcement_UnauthenticatedRequest_401NotLeakedAsForbidden(t *testing.T) {
	// A request with no/invalid token must fail at AuthMiddleware (401),
	// never reach RBAC enforcement at all - proving the middleware order
	// is correct (auth before permissions).
	tenantID, roleID := uuid.New(), uuid.New()
	handler, _, _ := buildTestMuxAndMiddleware(t, tenantID, roleID, "alice@example.com", service.Permissions{"trust_providers": service.ReadWrite})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/trust-providers", nil)
	// No Authorization header at all.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated request: status = %d, want 401 (must fail at auth, not RBAC)", w.Code)
	}
}

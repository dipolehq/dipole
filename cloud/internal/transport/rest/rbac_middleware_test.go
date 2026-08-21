package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/service"
)

// --- Fakes ---

type fakeUserRepoForRBAC struct {
	usersByTenantEmail map[string]*domain.User // key: tenantID.String()+"|"+email
	err                error
}

func newFakeUserRepoForRBAC() *fakeUserRepoForRBAC {
	return &fakeUserRepoForRBAC{usersByTenantEmail: map[string]*domain.User{}}
}

func (f *fakeUserRepoForRBAC) put(tenantID uuid.UUID, u *domain.User) {
	f.usersByTenantEmail[tenantID.String()+"|"+u.Email] = u
}

func (f *fakeUserRepoForRBAC) Create(context.Context, uuid.UUID, *domain.User) error { return nil }
func (f *fakeUserRepoForRBAC) GetByID(context.Context, uuid.UUID, uuid.UUID) (*domain.User, error) {
	return nil, nil
}
func (f *fakeUserRepoForRBAC) GetByEmail(_ context.Context, tenantID uuid.UUID, email string) (*domain.User, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.usersByTenantEmail[tenantID.String()+"|"+email], nil
}
func (f *fakeUserRepoForRBAC) GetTenantsByEmail(context.Context, string) ([]domain.TenantSummary, error) {
	return nil, nil
}
func (f *fakeUserRepoForRBAC) List(context.Context, uuid.UUID, domain.ListOptions) ([]domain.User, int, error) {
	return nil, 0, nil
}
func (f *fakeUserRepoForRBAC) Update(context.Context, uuid.UUID, *domain.User) error { return nil }
func (f *fakeUserRepoForRBAC) Delete(context.Context, uuid.UUID, uuid.UUID) error    { return nil }

type fakeRoleRepoForRBAC struct {
	rolesByTenantID map[string]*domain.Role // key: tenantID.String()+"|"+roleID.String()
	err             error
}

func newFakeRoleRepoForRBAC() *fakeRoleRepoForRBAC {
	return &fakeRoleRepoForRBAC{rolesByTenantID: map[string]*domain.Role{}}
}

func (f *fakeRoleRepoForRBAC) put(tenantID uuid.UUID, r *domain.Role) {
	f.rolesByTenantID[tenantID.String()+"|"+r.ID.String()] = r
}

func (f *fakeRoleRepoForRBAC) Create(context.Context, uuid.UUID, *domain.Role) error { return nil }
func (f *fakeRoleRepoForRBAC) GetByID(_ context.Context, tenantID uuid.UUID, id uuid.UUID) (*domain.Role, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.rolesByTenantID[tenantID.String()+"|"+id.String()], nil
}
func (f *fakeRoleRepoForRBAC) List(context.Context, uuid.UUID, domain.ListOptions) ([]domain.Role, int, error) {
	return nil, 0, nil
}
func (f *fakeRoleRepoForRBAC) Update(context.Context, uuid.UUID, *domain.Role) error { return nil }
func (f *fakeRoleRepoForRBAC) Delete(context.Context, uuid.UUID, uuid.UUID) error    { return nil }
func (f *fakeRoleRepoForRBAC) AssignResourceSets(context.Context, uuid.UUID, uuid.UUID, []uuid.UUID) error {
	return nil
}
func (f *fakeRoleRepoForRBAC) GetResourceSetIDs(context.Context, uuid.UUID, uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}

func testLoggerForRBAC() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// --- resolvePermissions tests ---

func TestResolvePermissions_ActiveUserWithActiveRole_ReturnsParsedPermissions(t *testing.T) {
	tenantID := uuid.New()
	roleID := uuid.New()
	permsJSON, _ := json.Marshal(service.AdminPermissions())

	users := newFakeUserRepoForRBAC()
	users.put(tenantID, &domain.User{Email: "alice@example.com", IsActive: true, RoleID: &roleID})

	roles := newFakeRoleRepoForRBAC()
	roles.put(tenantID, &domain.Role{ID: roleID, IsActive: true, Permissions: permsJSON})

	got, _ := resolvePermissions(context.Background(), users, roles, tenantID, "alice@example.com", testLoggerForRBAC())

	if err := service.NewRBACService().CheckPermission(got, "access_policies", http.MethodPost); err != nil {
		t.Errorf("expected AdminPermissions to allow POST access_policies, got %v", err)
	}
	if err := service.NewRBACService().CheckPermission(got, "roles", http.MethodPut); err == nil {
		t.Error("expected AdminPermissions to forbid PUT roles")
	}
}

func TestResolvePermissions_NoTenantID_ReturnsNil(t *testing.T) {
	users := newFakeUserRepoForRBAC()
	roles := newFakeRoleRepoForRBAC()

	got, _ := resolvePermissions(context.Background(), users, roles, uuid.Nil, "alice@example.com", testLoggerForRBAC())
	assertDeniesEverything(t, got)
}

func TestResolvePermissions_NoEmail_ReturnsNil(t *testing.T) {
	users := newFakeUserRepoForRBAC()
	roles := newFakeRoleRepoForRBAC()

	got, _ := resolvePermissions(context.Background(), users, roles, uuid.New(), "", testLoggerForRBAC())
	assertDeniesEverything(t, got)
}

func TestResolvePermissions_UserNotFound_ReturnsNil(t *testing.T) {
	tenantID := uuid.New()
	users := newFakeUserRepoForRBAC() // empty - no matching user
	roles := newFakeRoleRepoForRBAC()

	got, _ := resolvePermissions(context.Background(), users, roles, tenantID, "ghost@example.com", testLoggerForRBAC())
	assertDeniesEverything(t, got)
}

func TestResolvePermissions_UserRepoError_ReturnsNil(t *testing.T) {
	tenantID := uuid.New()
	users := newFakeUserRepoForRBAC()
	users.err = fmt.Errorf("db unavailable")
	roles := newFakeRoleRepoForRBAC()

	got, _ := resolvePermissions(context.Background(), users, roles, tenantID, "alice@example.com", testLoggerForRBAC())
	assertDeniesEverything(t, got)
}

func TestResolvePermissions_InactiveUser_ReturnsNil(t *testing.T) {
	tenantID := uuid.New()
	roleID := uuid.New()
	users := newFakeUserRepoForRBAC()
	users.put(tenantID, &domain.User{Email: "alice@example.com", IsActive: false, RoleID: &roleID})
	roles := newFakeRoleRepoForRBAC()
	roles.put(tenantID, &domain.Role{ID: roleID, IsActive: true, Permissions: mustMarshal(service.SuperAdminPermissions())})

	got, _ := resolvePermissions(context.Background(), users, roles, tenantID, "alice@example.com", testLoggerForRBAC())
	assertDeniesEverything(t, got)
}

func TestResolvePermissions_UserHasNoRoleAssigned_ReturnsNil(t *testing.T) {
	tenantID := uuid.New()
	users := newFakeUserRepoForRBAC()
	users.put(tenantID, &domain.User{Email: "alice@example.com", IsActive: true, RoleID: nil})
	roles := newFakeRoleRepoForRBAC()

	got, _ := resolvePermissions(context.Background(), users, roles, tenantID, "alice@example.com", testLoggerForRBAC())
	assertDeniesEverything(t, got)
}

func TestResolvePermissions_RoleNotFound_ReturnsNil(t *testing.T) {
	tenantID := uuid.New()
	roleID := uuid.New() // never inserted into the fake role repo
	users := newFakeUserRepoForRBAC()
	users.put(tenantID, &domain.User{Email: "alice@example.com", IsActive: true, RoleID: &roleID})
	roles := newFakeRoleRepoForRBAC()

	got, _ := resolvePermissions(context.Background(), users, roles, tenantID, "alice@example.com", testLoggerForRBAC())
	assertDeniesEverything(t, got)
}

func TestResolvePermissions_RoleRepoError_ReturnsNil(t *testing.T) {
	tenantID := uuid.New()
	roleID := uuid.New()
	users := newFakeUserRepoForRBAC()
	users.put(tenantID, &domain.User{Email: "alice@example.com", IsActive: true, RoleID: &roleID})
	roles := newFakeRoleRepoForRBAC()
	roles.err = fmt.Errorf("db unavailable")

	got, _ := resolvePermissions(context.Background(), users, roles, tenantID, "alice@example.com", testLoggerForRBAC())
	assertDeniesEverything(t, got)
}

func TestResolvePermissions_InactiveRole_ReturnsNil(t *testing.T) {
	tenantID := uuid.New()
	roleID := uuid.New()
	users := newFakeUserRepoForRBAC()
	users.put(tenantID, &domain.User{Email: "alice@example.com", IsActive: true, RoleID: &roleID})
	roles := newFakeRoleRepoForRBAC()
	roles.put(tenantID, &domain.Role{ID: roleID, IsActive: false, Permissions: mustMarshal(service.SuperAdminPermissions())})

	got, _ := resolvePermissions(context.Background(), users, roles, tenantID, "alice@example.com", testLoggerForRBAC())
	assertDeniesEverything(t, got)
}

func TestResolvePermissions_EmptyRolePermissions_ReturnsNil(t *testing.T) {
	// This was the exact state the tenant-bootstrap "SuperAdmin" role used
	// to be left in before Phase 4 fixed auth/signup.go to set real
	// SuperAdminPermissions() - kept as a general defensive case (any role
	// with a nil/absent Permissions field, however it got that way, must
	// fail closed, not silently grant access).
	tenantID := uuid.New()
	roleID := uuid.New()
	users := newFakeUserRepoForRBAC()
	users.put(tenantID, &domain.User{Email: "alice@example.com", IsActive: true, RoleID: &roleID})
	roles := newFakeRoleRepoForRBAC()
	roles.put(tenantID, &domain.Role{ID: roleID, IsActive: true, Permissions: nil})

	got, _ := resolvePermissions(context.Background(), users, roles, tenantID, "alice@example.com", testLoggerForRBAC())
	assertDeniesEverything(t, got)
}

func TestResolvePermissions_MalformedRolePermissions_ReturnsNil(t *testing.T) {
	tenantID := uuid.New()
	roleID := uuid.New()
	users := newFakeUserRepoForRBAC()
	users.put(tenantID, &domain.User{Email: "alice@example.com", IsActive: true, RoleID: &roleID})
	roles := newFakeRoleRepoForRBAC()
	roles.put(tenantID, &domain.Role{ID: roleID, IsActive: true, Permissions: json.RawMessage(`not valid json`)})

	got, _ := resolvePermissions(context.Background(), users, roles, tenantID, "alice@example.com", testLoggerForRBAC())
	assertDeniesEverything(t, got)
}

func TestResolvePermissions_ScopedAPIToken_GetsTenantWideAccess(t *testing.T) {
	// Matches auth.APITokenAuthenticator.tryScopedAPIToken's synthetic
	// email "apitoken@scoped" - these tokens have no associated user row
	// at all by design (domain.APIToken has no UserID field).
	tenantID := uuid.New()
	users := newFakeUserRepoForRBAC() // deliberately empty
	roles := newFakeRoleRepoForRBAC() // deliberately empty

	got, _ := resolvePermissions(context.Background(), users, roles, tenantID, "apitoken@scoped", testLoggerForRBAC())

	if err := service.NewRBACService().CheckPermission(got, "roles", http.MethodDelete); err != nil {
		t.Errorf("expected tenant-wide (Super Admin) access for a scoped API token identity, got %v", err)
	}
}

func TestResolvePermissions_LegacyTenantToken_GetsTenantWideAccess(t *testing.T) {
	// Matches auth.APITokenAuthenticator.tryLegacyTenantToken's synthetic
	// email "apitoken@"+subdomain.
	tenantID := uuid.New()
	users := newFakeUserRepoForRBAC()
	roles := newFakeRoleRepoForRBAC()

	got, _ := resolvePermissions(context.Background(), users, roles, tenantID, "apitoken@acme-corp", testLoggerForRBAC())

	if err := service.NewRBACService().CheckPermission(got, "identity_providers", http.MethodPost); err != nil {
		t.Errorf("expected tenant-wide (Super Admin) access for a legacy tenant token identity, got %v", err)
	}
}

func TestResolvePermissions_TenantIsolation_UserFromOtherTenantNotFound(t *testing.T) {
	// The same email exists in two different tenants with different
	// roles. resolvePermissions must only ever resolve the user/role
	// belonging to the tenant actually passed in (which itself comes
	// only from the validated auth token, never client input) - never
	// leak or fall back to a same-email user in a different tenant.
	tenantA := uuid.New()
	tenantB := uuid.New()
	roleA := uuid.New()
	roleB := uuid.New()

	users := newFakeUserRepoForRBAC()
	users.put(tenantA, &domain.User{Email: "shared@example.com", IsActive: true, RoleID: &roleA})
	users.put(tenantB, &domain.User{Email: "shared@example.com", IsActive: true, RoleID: &roleB})

	roles := newFakeRoleRepoForRBAC()
	roles.put(tenantA, &domain.Role{ID: roleA, IsActive: true, Permissions: mustMarshal(service.SuperAdminPermissions())})
	roles.put(tenantB, &domain.Role{ID: roleB, IsActive: true, Permissions: mustMarshal(service.ReadOnlyPermissions())})

	gotA, _ := resolvePermissions(context.Background(), users, roles, tenantA, "shared@example.com", testLoggerForRBAC())
	gotB, _ := resolvePermissions(context.Background(), users, roles, tenantB, "shared@example.com", testLoggerForRBAC())

	if err := service.NewRBACService().CheckPermission(gotA, "roles", http.MethodDelete); err != nil {
		t.Errorf("tenant A's role should allow DELETE roles (Super Admin), got %v", err)
	}
	if err := service.NewRBACService().CheckPermission(gotB, "roles", http.MethodDelete); err == nil {
		t.Error("tenant B's role should forbid DELETE roles (Read Only) - must not have inherited tenant A's permissions")
	}
}

func mustMarshal(p service.Permissions) json.RawMessage {
	b, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}
	return b
}

// assertDeniesEverything confirms got is a fail-closed Permissions value:
// service.CheckPermission must deny both a read and a write on any
// resource, proving there's no path by which a nil/empty map is treated
// as "allow all" anywhere downstream.
func assertDeniesEverything(t *testing.T, got service.Permissions) {
	t.Helper()
	svc := service.NewRBACService()
	if err := svc.CheckPermission(got, "trust_providers", http.MethodGet); err == nil {
		t.Error("expected fail-closed permissions to deny GET trust_providers")
	}
	if err := svc.CheckPermission(got, "trust_providers", http.MethodPost); err == nil {
		t.Error("expected fail-closed permissions to deny POST trust_providers")
	}
}

// --- RBACPermissionsMiddleware (full HTTP middleware) test ---

func TestRBACPermissionsMiddleware_StoresResolvedPermissionsInContext(t *testing.T) {
	tenantID := uuid.New()
	roleID := uuid.New()

	users := newFakeUserRepoForRBAC()
	users.put(tenantID, &domain.User{Email: "alice@example.com", IsActive: true, RoleID: &roleID})
	roles := newFakeRoleRepoForRBAC()
	roles.put(tenantID, &domain.Role{ID: roleID, IsActive: true, Permissions: mustMarshal(service.AdminPermissions())})

	var capturedPermissions service.Permissions
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPermissions = PermissionsFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	mw := RBACPermissionsMiddleware(users, roles, testLoggerForRBAC())(next)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/trust-providers", nil)
	ctx := context.WithValue(req.Context(), ctxTenantID, tenantID)
	ctx = context.WithValue(ctx, ctxUserEmail, "alice@example.com")
	req = req.WithContext(ctx)

	mw.ServeHTTP(httptest.NewRecorder(), req)

	if capturedPermissions == nil {
		t.Fatal("expected PermissionsFromContext to return a non-nil map for a valid active user/role")
	}
	if err := service.NewRBACService().CheckPermission(capturedPermissions, "access_policies", http.MethodPost); err != nil {
		t.Errorf("expected downstream handler to see AdminPermissions allowing POST access_policies, got %v", err)
	}
}

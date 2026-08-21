// rbac_test.go tests permission checking, default role permission sets, and ResourceFromPath's path-to-resource matching.
package service

import (
	"errors"
	"testing"
)

func TestRBAC_ReadOnlyBlocksWrite(t *testing.T) {
	svc := NewRBACService()
	perms := ReadOnlyPermissions()

	err := svc.CheckPermission(perms, "client_workloads", "POST")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestRBAC_ReadOnlyAllowsRead(t *testing.T) {
	svc := NewRBACService()
	perms := ReadOnlyPermissions()

	err := svc.CheckPermission(perms, "client_workloads", "GET")
	if err != nil {
		t.Fatalf("expected nil (allowed), got %v", err)
	}
}

func TestRBAC_AdminAllowsWrite(t *testing.T) {
	svc := NewRBACService()
	perms := AdminPermissions()

	err := svc.CheckPermission(perms, "access_policies", "POST")
	if err != nil {
		t.Fatalf("expected nil (allowed), got %v", err)
	}
}

func TestRBAC_AdminBlocksRoleWrite(t *testing.T) {
	svc := NewRBACService()
	perms := AdminPermissions()

	err := svc.CheckPermission(perms, "roles", "PUT")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden for Admin writing roles, got %v", err)
	}
}

func TestRBAC_CheckPermission_ForbiddenErrorDetail(t *testing.T) {
	svc := NewRBACService()
	perms := ReadOnlyPermissions()

	err := svc.CheckPermission(perms, "trust_providers", "DELETE")

	var forbiddenErr *ForbiddenError
	if !errors.As(err, &forbiddenErr) {
		t.Fatalf("expected *ForbiddenError, got %T: %v", err, err)
	}
	if forbiddenErr.Resource != "trust_providers" {
		t.Errorf("Resource = %q, want %q", forbiddenErr.Resource, "trust_providers")
	}
	if forbiddenErr.RequiredPermission != "write" {
		t.Errorf("RequiredPermission = %q, want %q", forbiddenErr.RequiredPermission, "write")
	}
}

func TestRBAC_CheckPermission_NoAccessBlocksBothReadAndWrite(t *testing.T) {
	svc := NewRBACService()
	perms := Permissions{"trust_providers": NoAccess}

	if err := svc.CheckPermission(perms, "trust_providers", "GET"); !errors.Is(err, ErrForbidden) {
		t.Errorf("expected NoAccess to forbid GET, got %v", err)
	}
	if err := svc.CheckPermission(perms, "trust_providers", "POST"); !errors.Is(err, ErrForbidden) {
		t.Errorf("expected NoAccess to forbid POST, got %v", err)
	}
}

func TestRBAC_CheckPermission_ReadOnlyAllowsReadBlocksWrite(t *testing.T) {
	svc := NewRBACService()
	perms := Permissions{"trust_providers": ReadOnly}

	if err := svc.CheckPermission(perms, "trust_providers", "GET"); err != nil {
		t.Errorf("expected ReadOnly to allow GET, got %v", err)
	}
	for _, method := range []string{"POST", "PUT", "DELETE"} {
		if err := svc.CheckPermission(perms, "trust_providers", method); !errors.Is(err, ErrForbidden) {
			t.Errorf("expected ReadOnly to forbid %s, got %v", method, err)
		}
	}
}

func TestRBAC_CheckPermission_ReadWriteAllowsEverything(t *testing.T) {
	svc := NewRBACService()
	perms := Permissions{"trust_providers": ReadWrite}

	for _, method := range []string{"GET", "POST", "PUT", "DELETE"} {
		if err := svc.CheckPermission(perms, "trust_providers", method); err != nil {
			t.Errorf("expected ReadWrite to allow %s, got %v", method, err)
		}
	}
}

func TestSuperAdminPermissions_CoversEveryRegistryResourceWithReadWrite(t *testing.T) {
	perms := SuperAdminPermissions()
	if len(perms) != len(ResourceRegistry) {
		t.Fatalf("SuperAdminPermissions has %d entries, want %d (one per ResourceRegistry entry)", len(perms), len(ResourceRegistry))
	}
	for _, res := range ResourceRegistry {
		if perms[res.ID] != ReadWrite {
			t.Errorf("SuperAdminPermissions[%q] = %q, want ReadWrite", res.ID, perms[res.ID])
		}
	}
}

func TestReadOnlyPermissions_CoversEveryRegistryResourceWithReadOnly(t *testing.T) {
	perms := ReadOnlyPermissions()
	if len(perms) != len(ResourceRegistry) {
		t.Fatalf("ReadOnlyPermissions has %d entries, want %d", len(perms), len(ResourceRegistry))
	}
	for _, res := range ResourceRegistry {
		if perms[res.ID] != ReadOnly {
			t.Errorf("ReadOnlyPermissions[%q] = %q, want ReadOnly", res.ID, perms[res.ID])
		}
	}
}

func TestAdminPermissions_ReadOnlyResourcesNeverGetReadWrite(t *testing.T) {
	perms := AdminPermissions()
	for _, res := range ResourceRegistry {
		if res.ReadOnly && perms[res.ID] == ReadWrite {
			t.Errorf("AdminPermissions[%q] = ReadWrite, but this resource has no write endpoint (ResourceRegistry marks it ReadOnly)", res.ID)
		}
	}
}

func TestAdminPermissions_RolesAndIdentityProvidersAreReadOnly(t *testing.T) {
	perms := AdminPermissions()
	if perms["roles"] != ReadOnly {
		t.Errorf(`AdminPermissions["roles"] = %q, want ReadOnly`, perms["roles"])
	}
	if perms["identity_providers"] != ReadOnly {
		t.Errorf(`AdminPermissions["identity_providers"] = %q, want ReadOnly`, perms["identity_providers"])
	}
}

func TestResourceFromPath_NewEventAndComplianceResources(t *testing.T) {
	cases := []struct {
		path     string
		expected string
	}{
		{"/api/v1/events/authorization", "access_authorization_events"},
		{"/api/v1/events/audit", "audit_logs"},
		{"/api/v1/events/workload", "workload_events"},
		{"/api/v1/global-policy-compliance", "global_policy_compliance"},
		{"/api/v1/global-policy-compliance/report", "global_policy_compliance"},
	}
	for _, c := range cases {
		if got := ResourceFromPath(c.path); got != c.expected {
			t.Errorf("ResourceFromPath(%q) = %q, want %q", c.path, got, c.expected)
		}
	}
}

func TestResourceRegistry_NoDuplicateIDs(t *testing.T) {
	seen := map[string]bool{}
	for _, res := range ResourceRegistry {
		if seen[res.ID] {
			t.Errorf("duplicate resource ID %q in ResourceRegistry", res.ID)
		}
		seen[res.ID] = true
	}
}

func TestResourceRegistry_NoDuplicatePathSegments(t *testing.T) {
	seen := map[string]string{}
	for _, res := range ResourceRegistry {
		for _, seg := range res.PathSegments {
			if owner, ok := seen[seg]; ok {
				t.Errorf("path segment %q is claimed by both %q and %q", seg, owner, res.ID)
			}
			seen[seg] = res.ID
		}
	}
}

func TestRBAC_CheckPermission_UnknownResource_Forbidden(t *testing.T) {
	svc := NewRBACService()
	perms := SuperAdminPermissions()
	err := svc.CheckPermission(perms, "some_future_resource", "GET")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden for a resource absent from the permissions map, got %v", err)
	}
}

func TestResourceFromPath_AllRegisteredResources(t *testing.T) {
	cases := []struct {
		path     string
		expected string
	}{
		{"/api/v1/client-workloads", "client_workloads"},
		{"/api/v1/client-workloads/123", "client_workloads"},
		{"/api/v1/server-workloads", "server_workloads"},
		{"/api/v1/trust-providers", "trust_providers"},
		{"/api/v1/trust-providers/123", "trust_providers"},
		{"/api/v1/credential-providers", "credential_providers"},
		{"/api/v1/access-policies", "access_policies"},
		{"/api/v1/access-conditions", "access_conditions"},
		{"/api/v1/agent-controllers", "agent_controllers"},
		{"/api/v1/agent-controllers/123/device-code", "agent_controllers"},
		{"/api/v1/users", "users"},
		{"/api/v1/users/123", "users"},
		{"/api/v1/roles", "roles"},
		{"/api/v1/roles/123", "roles"},
		{"/api/v1/resource-sets", "resource_sets"},
		{"/api/v1/log-streams", "log_streams"},
		{"/api/v1/identity-providers", "identity_providers"},
	}
	for _, c := range cases {
		if got := ResourceFromPath(c.path); got != c.expected {
			t.Errorf("ResourceFromPath(%q) = %q, want %q", c.path, got, c.expected)
		}
	}
}

func TestResourceFromPath_NoResourceMatch_ReturnsEmpty(t *testing.T) {
	cases := []string{
		"/api/v1/health",
		"/api/v1/profile",
		"/api/v1/profile/mfa/status",
		"/api/v1/dashboard",
		"/api/v1/auth/login",
	}
	for _, path := range cases {
		if got := ResourceFromPath(path); got != "" {
			t.Errorf("ResourceFromPath(%q) = %q, want empty string", path, got)
		}
	}
}

func TestResourceFromPath_SubstringIsNotMistakenForResource(t *testing.T) {
	cases := []struct {
		path     string
		expected string
	}{
		{"/api/v1/user-roles", ""},
		{"/api/v1/super-users", ""},
		{"/api/v1/trust-providers-old", ""},
		{"/api/v1/non-agent-controllers", ""},
	}
	for _, c := range cases {
		if got := ResourceFromPath(c.path); got != c.expected {
			t.Errorf("ResourceFromPath(%q) = %q, want %q (segment-exact match should not fire on substrings)", c.path, got, c.expected)
		}
	}
}

func TestResourceFromPath_FirstMatchingSegmentWins_Deterministic(t *testing.T) {
	path := "/api/v1/roles/123/users"
	for i := 0; i < 20; i++ {
		if got := ResourceFromPath(path); got != "roles" {
			t.Fatalf("ResourceFromPath(%q) = %q, want %q (must be deterministic across repeated calls)", path, got, "roles")
		}
	}
}

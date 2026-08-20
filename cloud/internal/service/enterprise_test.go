package service

import (
	"testing"
)

func TestRBAC_ReadOnlyBlocksWrite(t *testing.T) {
	svc := NewRBACService()
	perms := ReadOnlyPermissions()

	err := svc.CheckPermission(perms, "client_workloads", "POST")
	if err != ErrForbidden {
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
	if err != ErrForbidden {
		t.Fatalf("expected ErrForbidden for Admin writing roles, got %v", err)
	}
}

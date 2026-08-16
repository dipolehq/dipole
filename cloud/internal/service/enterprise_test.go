package service

import (
	"testing"

	"github.com/google/uuid"
)

func TestResourceSet_DefaultCannotBeDeleted(t *testing.T) {
	svc := &ResourceSetService{}
	err := svc.Delete(nil, uuid.New(), DefaultResourceSetID)
	if err == nil {
		t.Fatal("expected error when deleting default Resource Set")
	}
}

func TestResourceSet_ResolveDefault(t *testing.T) {
	id := ResolveResourceSetID(nil)
	if id != DefaultResourceSetID {
		t.Errorf("expected default RS ID, got %s", id)
	}
}

func TestResourceSet_ResolveProvided(t *testing.T) {
	custom := uuid.New()
	id := ResolveResourceSetID(&custom)
	if id != custom {
		t.Errorf("expected provided RS ID %s, got %s", custom, id)
	}
}

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

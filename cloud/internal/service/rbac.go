/*
rbac.go implements Role-Based Access Control for the admin API.

RBACService checks if a user's role permits a specific action (read/write)
on a specific resource type. Permissions are per-resource-type with read/write
granularity. Roles can be scoped to Resource Sets.
*/
package service

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
)

var ErrForbidden = errors.New("forbidden: insufficient permissions")

type Permissions map[string]ResourcePermission

type ResourcePermission struct {
	Read  bool `json:"read"`
	Write bool `json:"write"`
}

type RBACService struct{}

func NewRBACService() *RBACService {
	return &RBACService{}
}

func (s *RBACService) CheckPermission(permissions Permissions, resource string, method string) error {
	perm, ok := permissions[resource]
	if !ok {
		return ErrForbidden
	}

	if isWriteMethod(method) {
		if !perm.Write {
			return ErrForbidden
		}
	} else {
		if !perm.Read {
			return ErrForbidden
		}
	}

	return nil
}

func isWriteMethod(method string) bool {
	return method == http.MethodPost || method == http.MethodPut || method == http.MethodDelete
}

func ParsePermissions(raw json.RawMessage) (Permissions, error) {
	var perms Permissions
	if err := json.Unmarshal(raw, &perms); err != nil {
		return nil, err
	}
	return perms, nil
}

func SuperAdminPermissions() Permissions {
	return Permissions{
		"client_workloads":     {Read: true, Write: true},
		"server_workloads":     {Read: true, Write: true},
		"trust_providers":      {Read: true, Write: true},
		"credential_providers": {Read: true, Write: true},
		"access_policies":      {Read: true, Write: true},
		"access_conditions":    {Read: true, Write: true},
		"agent_controllers":    {Read: true, Write: true},
		"users":                {Read: true, Write: true},
		"roles":                {Read: true, Write: true},
		"resource_sets":        {Read: true, Write: true},
		"log_streams":          {Read: true, Write: true},
		"identity_providers":   {Read: true, Write: true},
	}
}

func AdminPermissions() Permissions {
	return Permissions{
		"client_workloads":     {Read: true, Write: true},
		"server_workloads":     {Read: true, Write: true},
		"trust_providers":      {Read: true, Write: true},
		"credential_providers": {Read: true, Write: true},
		"access_policies":      {Read: true, Write: true},
		"access_conditions":    {Read: true, Write: true},
		"agent_controllers":    {Read: true, Write: true},
		"users":                {Read: true, Write: true},
		"roles":                {Read: true, Write: false},
		"resource_sets":        {Read: true, Write: true},
		"log_streams":          {Read: true, Write: true},
		"identity_providers":   {Read: true, Write: false},
	}
}

func ReadOnlyPermissions() Permissions {
	return Permissions{
		"client_workloads":     {Read: true, Write: false},
		"server_workloads":     {Read: true, Write: false},
		"trust_providers":      {Read: true, Write: false},
		"credential_providers": {Read: true, Write: false},
		"access_policies":      {Read: true, Write: false},
		"access_conditions":    {Read: true, Write: false},
		"agent_controllers":    {Read: true, Write: false},
		"users":                {Read: true, Write: false},
		"roles":                {Read: true, Write: false},
		"resource_sets":        {Read: true, Write: false},
		"log_streams":          {Read: true, Write: false},
		"identity_providers":   {Read: true, Write: false},
	}
}

type DefaultRole struct {
	Name        string
	Permissions Permissions
}

func DefaultRoles() []DefaultRole {
	return []DefaultRole{
		{Name: "Super Admin", Permissions: SuperAdminPermissions()},
		{Name: "Admin", Permissions: AdminPermissions()},
		{Name: "Auditor", Permissions: ReadOnlyPermissions()},
	}
}

func ResourceFromPath(path string) string {
	resources := map[string]string{
		"client-workloads":     "client_workloads",
		"server-workloads":     "server_workloads",
		"trust-providers":      "trust_providers",
		"credential-providers": "credential_providers",
		"access-policies":      "access_policies",
		"access-conditions":    "access_conditions",
		"agent-controllers":    "agent_controllers",
		"users":                "users",
		"roles":                "roles",
		"resource-sets":        "resource_sets",
		"log-streams":          "log_streams",
		"identity-providers":   "identity_providers",
	}

	for pathSegment, resource := range resources {
		if containsSegment(path, pathSegment) {
			return resource
		}
	}
	return ""
}

func containsSegment(path, segment string) bool {
	return len(path) >= len(segment) && findSubstring(path, segment)
}

func findSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// UserResourceSetRole represents a user's role scoped to a specific Resource Set.
type UserResourceSetRole struct {
	UserID        uuid.UUID
	ResourceSetID uuid.UUID
	RoleID        uuid.UUID
}

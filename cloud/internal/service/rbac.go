// rbac.go implements Role-Based Access Control for the admin API: role permission storage, default role definitions, and the resource registry.
package service

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

var ErrForbidden = errors.New("forbidden: insufficient permissions")

type ForbiddenError struct {
	Resource           string
	RequiredPermission string
}

func (e *ForbiddenError) Error() string {
	return "forbidden: " + e.RequiredPermission + " access to " + e.Resource + " is not permitted for this role"
}

func (e *ForbiddenError) Unwrap() error {
	return ErrForbidden
}

type PermissionLevel string

const (
	NoAccess  PermissionLevel = "no_access"
	ReadOnly  PermissionLevel = "read_only"
	ReadWrite PermissionLevel = "read_write"
)

type Permissions map[string]PermissionLevel

type RBACService struct{}

func NewRBACService() *RBACService {
	return &RBACService{}
}

func (s *RBACService) CheckPermission(permissions Permissions, resource string, method string) error {
	requiredPermission := "read"
	if IsWriteMethod(method) {
		requiredPermission = "write"
	}

	level := permissions[resource]

	switch requiredPermission {
	case "write":
		if level != ReadWrite {
			return &ForbiddenError{Resource: resource, RequiredPermission: requiredPermission}
		}
	default:
		if level != ReadOnly && level != ReadWrite {
			return &ForbiddenError{Resource: resource, RequiredPermission: requiredPermission}
		}
	}

	return nil
}

func IsWriteMethod(method string) bool {
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
	perms := make(Permissions, len(ResourceRegistry))
	for _, res := range ResourceRegistry {
		perms[res.ID] = ReadWrite
	}
	return perms
}

func AdminPermissions() Permissions {
	perms := make(Permissions, len(ResourceRegistry))
	for _, res := range ResourceRegistry {
		switch {
		case res.ReadOnly:
			perms[res.ID] = ReadOnly
		case res.ID == "roles" || res.ID == "identity_providers":
			perms[res.ID] = ReadOnly
		default:
			perms[res.ID] = ReadWrite
		}
	}
	return perms
}

func ReadOnlyPermissions() Permissions {
	perms := make(Permissions, len(ResourceRegistry))
	for _, res := range ResourceRegistry {
		perms[res.ID] = ReadOnly
	}
	return perms
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

type Resource struct {
	ID           string
	Label        string
	PathSegments []string
	ReadOnly     bool
}

var ResourceRegistry = []Resource{
	{ID: "client_workloads", Label: "Client Workloads", PathSegments: []string{"client-workloads"}},
	{ID: "server_workloads", Label: "Server Workloads", PathSegments: []string{"server-workloads"}},
	{ID: "trust_providers", Label: "Trust Providers", PathSegments: []string{"trust-providers"}},
	{ID: "credential_providers", Label: "Credential Providers", PathSegments: []string{"credential-providers"}},
	{ID: "access_policies", Label: "Access Policies", PathSegments: []string{"access-policies"}},
	{ID: "access_conditions", Label: "Access Conditions", PathSegments: []string{"access-conditions"}},
	{ID: "agent_controllers", Label: "Agent Controllers", PathSegments: []string{"agent-controllers", "agent-proxies"}},
	{ID: "users", Label: "Users", PathSegments: []string{"users"}},
	{ID: "roles", Label: "Roles", PathSegments: []string{"roles"}},
	{ID: "resource_sets", Label: "Resource Sets", PathSegments: []string{"resource-sets"}},
	{ID: "log_streams", Label: "Log Streams", PathSegments: []string{"log-streams"}},
	{ID: "identity_providers", Label: "Identity Providers", PathSegments: []string{"identity-providers"}},
	{ID: "access_authorization_events", Label: "Access Authorization Events", PathSegments: []string{"authorization"}, ReadOnly: true},
	{ID: "audit_logs", Label: "Audit Logs", PathSegments: []string{"audit"}, ReadOnly: true},
	{ID: "workload_events", Label: "Workload Events", PathSegments: []string{"workload"}, ReadOnly: true},
	{ID: "global_policy_compliance", Label: "Global Policy Compliance", PathSegments: []string{"global-policy-compliance"}},
	{ID: "sign_on_policy", Label: "Sign-On Policy", PathSegments: []string{"sign-on-policy"}},
	{ID: "tls_decrypt", Label: "TLS Decrypt", PathSegments: []string{"tls-decrypt"}},
}

var resourcePathIndex = buildResourcePathIndex(ResourceRegistry)

func buildResourcePathIndex(registry []Resource) map[string]string {
	index := make(map[string]string, len(registry))
	for _, res := range registry {
		for _, segment := range res.PathSegments {
			index[segment] = res.ID
		}
	}
	return index
}

func ResourceFromPath(path string) string {
	for _, segment := range strings.Split(path, "/") {
		if resource, ok := resourcePathIndex[segment]; ok {
			return resource
		}
	}
	return ""
}

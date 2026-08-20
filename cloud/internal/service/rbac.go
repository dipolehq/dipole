/*
rbac.go implements Role-Based Access Control for the admin API.

RBACService checks whether a role's Permissions (one PermissionLevel —
NoAccess/ReadOnly/ReadWrite — per resource, see ResourceRegistry) allow a
given HTTP method against a given resource. Enforcement against real
requests happens in cloud/internal/transport/rest/rbac_middleware.go and
rbac_enforce_middleware.go; this file is the pure policy/data layer they
depend on.
*/
package service

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

var ErrForbidden = errors.New("forbidden: insufficient permissions")

// ForbiddenError wraps ErrForbidden with the specific resource and
// permission level that was required but not granted, so callers (the
// REST transport's error responses in particular) can tell a caller
// exactly what was denied instead of an opaque "forbidden". Errors.Is
// against ErrForbidden still works via Unwrap, so existing/future
// callers that only check the sentinel continue to work unchanged.
type ForbiddenError struct {
	Resource           string
	RequiredPermission string // "read" or "write"
}

func (e *ForbiddenError) Error() string {
	return "forbidden: " + e.RequiredPermission + " access to " + e.Resource + " is not permitted for this role"
}

func (e *ForbiddenError) Unwrap() error {
	return ErrForbidden
}

// PermissionLevel is the single access level a role holds for one
// resource — one of NoAccess, ReadOnly, or ReadWrite. Deliberately a
// single enum value per resource rather than independent Read/Write
// booleans: a pair of booleans can represent a nonsensical state (write
// without read) that has no meaning in this product, where every
// resource's mutation endpoints (create/update/delete) always require
// the same permission level as viewing it. An enum makes that
// nonsensical state unrepresentable instead of merely unused.
//
// The three string values below are the wire contract shared with the
// web client (web/src/features/administration/api/rolesApi.ts's
// Record<string, string>) — do not rename them without updating that
// side, since every stored role's Permissions JSON uses these exact
// strings as values.
type PermissionLevel string

const (
	NoAccess  PermissionLevel = "no_access"
	ReadOnly  PermissionLevel = "read_only"
	ReadWrite PermissionLevel = "read_write"
)

// Permissions maps a resource ID (see ResourceRegistry) to the access
// level a role holds for that resource. A resource absent from the map
// is treated identically to NoAccess by CheckPermission — there is no
// third "unspecified" state.
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

	level := permissions[resource] // zero value "" for an absent key, correctly treated as NoAccess below

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

// IsWriteMethod reports whether method requires "write" permission
// (POST/PUT/DELETE) rather than "read" (GET and everything else).
// Exported so callers outside this package that need to describe a
// permission requirement before calling CheckPermission (e.g. building
// an error message for a request that never resolved to a checkable
// resource at all) can derive the same requiredPermission value
// CheckPermission itself uses, rather than re-deriving the method list
// independently and risking drift.
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

// SuperAdminPermissions grants ReadWrite on every resource in
// ResourceRegistry — built by iterating the registry rather than listing
// resource IDs a second time, so a resource added to the registry is
// automatically covered here without a second edit.
func SuperAdminPermissions() Permissions {
	perms := make(Permissions, len(ResourceRegistry))
	for _, res := range ResourceRegistry {
		perms[res.ID] = ReadWrite
	}
	return perms
}

// AdminPermissions grants ReadWrite on every resource except "roles" and
// "identity_providers" (kept ReadOnly — role/SSO configuration changes
// are reserved for Super Admin), and NoAccess on any resource with no
// write endpoints at all (the event/log/compliance-report resources,
// which are inherently read-only — see ResourceRegistry.ReadOnly).
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

// ReadOnlyPermissions grants ReadOnly on every resource in
// ResourceRegistry.
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

// Resource describes one permission-checkable entity: a stable ID (used
// as the Permissions map key and sent over the wire), a human-readable
// Label (for display — the web client fetches this instead of
// hardcoding its own copy of the entity list, so the two can never drift
// out of sync the way they previously did), the URL path segment(s) that
// identify requests against it, and whether it has any write endpoints
// at all.
type Resource struct {
	ID    string
	Label string
	// PathSegments are the exact URL path segments (as split on "/")
	// that identify a request against this resource. Almost always one
	// segment; a couple of resources are addressed by more than one
	// distinct path (see the events/* resources below).
	PathSegments []string
	// ReadOnly marks a resource with no write (POST/PUT/DELETE)
	// endpoint at all — e.g. audit logs are queried, never mutated
	// through this API. AdminPermissions/SuperAdminPermissions use this
	// to avoid granting a "write" level that has no corresponding
	// endpoint to exercise; CheckPermission itself does not special-case
	// it (a ReadWrite grant on a read-only resource is simply never
	// exercised by a write request, since none exists to make).
	ReadOnly bool
}

// ResourceRegistry is the single source of truth for every
// permission-checkable resource in the admin API. Adding support for a
// new resource is exactly one addition here (plus registering its
// route(s), if not already registered) — SuperAdminPermissions,
// AdminPermissions, ReadOnlyPermissions, resourcePathIndex, and
// ResourceFromPath all derive from this list, so nothing else needs a
// parallel edit.
//
// "Integrations" (shown in the web client's permission table) is
// deliberately absent: it has no backend endpoint of any kind today
// (confirmed by repo-wide search) — there is nothing to enforce a
// permission against yet. Add it here once the corresponding feature and
// routes exist, not before.
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

// resourcePathIndex is ResourceRegistry flattened into a path-segment ->
// resource-ID lookup, built once at package init rather than on every
// ResourceFromPath call.
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

// ResourceFromPath returns the canonical resource ID for the first path
// segment that matches a known admin API resource, or "" if none match.
// Segment matching (not substring search) means the result is
// deterministic and depends only on the path's own left-to-right segment
// order, never on Go's unordered map iteration — a segment like
// "user-roles" or "trust-providers-old" (which contains a resource's
// segment as a substring but isn't equal to it) cannot be mistaken for
// the resource it merely contains.
//
// Note on the events/* resources: their path segments ("authorization",
// "audit", "workload") are only unambiguous because they're always
// preceded by "events" in the actual registered routes
// (/api/v1/events/authorization, etc.) — "audit" and "workload" alone
// are generic enough that ResourceFromPath's simple segment-membership
// check works today only because no other registered resource happens to
// use those exact words as a path segment. If that ever changes, this
// function must be extended to match the full remaining sub-path (e.g.
// "events/audit") rather than a single segment in isolation.
func ResourceFromPath(path string) string {
	for _, segment := range strings.Split(path, "/") {
		if resource, ok := resourcePathIndex[segment]; ok {
			return resource
		}
	}
	return ""
}

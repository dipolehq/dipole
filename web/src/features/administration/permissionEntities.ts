/**
 * permissionEntities.ts is the single frontend source of truth for the
 * permission-checkable entities shown on the Role create/detail pages.
 *
 * Each entry's `id` MUST match a resource ID in the backend's
 * cloud/internal/service/rbac.go ResourceRegistry exactly — that ID is
 * both the Permissions map key sent over the wire and the value
 * RBACEnforcementMiddleware checks incoming requests against. Only
 * `label` (the display text) is a purely frontend concern.
 *
 * Adding a new permission-checkable resource is additive on both sides,
 * not a rebuild: add one entry to ResourceRegistry on the backend (with
 * its route's path segment), add one matching entry here (with its
 * display label) — nothing else on either side needs to change. Keeping
 * this list and the backend's list in sync is a manual step for now;
 * the longer-term fix (the backend exposing its registry via an endpoint
 * this file fetches from instead of hardcoding) is tracked as a
 * follow-up, not implemented here.
 *
 * "Integrations" is intentionally absent — there is no backend feature
 * or endpoint behind it yet, so there is nothing to enforce a permission
 * against. Add it here once that feature and its ResourceRegistry entry
 * exist.
 */

export type PermissionLevel = 'no_access' | 'read_only' | 'read_write'

export interface PermissionEntity {
  id: string
  label: string
}

export const PERMISSION_ENTITIES: readonly PermissionEntity[] = [
  { id: 'access_policies', label: 'Access Policies' },
  { id: 'client_workloads', label: 'Client Workloads' },
  { id: 'trust_providers', label: 'Trust Providers' },
  { id: 'access_conditions', label: 'Access Conditions' },
  { id: 'credential_providers', label: 'Credential Providers' },
  { id: 'server_workloads', label: 'Server Workloads' },
  { id: 'agent_controllers', label: 'Agent Controllers' },
  { id: 'access_authorization_events', label: 'Access Authorization Events' },
  { id: 'audit_logs', label: 'Audit Logs' },
  { id: 'workload_events', label: 'Workload Events' },
  { id: 'users', label: 'Users' },
  { id: 'roles', label: 'Roles' },
  { id: 'log_streams', label: 'Log Streams' },
  { id: 'identity_providers', label: 'Identity Providers' },
  { id: 'resource_sets', label: 'Resource Sets' },
  { id: 'global_policy_compliance', label: 'Global Policy Compliance' },
  { id: 'sign_on_policy', label: 'Sign-On Policy' },
  { id: 'tls_decrypt', label: 'TLS Decrypt' },
] as const

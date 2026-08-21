export const LOG_TYPES = [
  { value: 'access_authorization', label: 'Access Authorization Events' },
  { value: 'audit_logs', label: 'Audit Logs' },
  { value: 'global_policy_compliance', label: 'Global Policy Compliance' },
  { value: 'workload_events', label: 'Workload Events' },
] as const

export type LogType = (typeof LOG_TYPES)[number]['value']

export const SEVERITY_OPTIONS = [
  { value: '', label: 'All Severities' },
  { value: 'info', label: 'Info' },
  { value: 'warn', label: 'Warning' },
  { value: 'error', label: 'Error' },
] as const

export const EVENT_TYPE_OPTIONS = [
  { value: '', label: 'All Event Types' },
  { value: 'access.authorization', label: 'Authorization' },
  { value: 'access.credential', label: 'Credential' },
  { value: 'access.discovery', label: 'Discovery' },
  { value: 'access.request', label: 'Request' },
] as const

export const REPORTING_PAGE_SIZE_OPTIONS = [10, 25, 50] as const

export const AUDIT_LOG_CATEGORY_OPTIONS = [
  { value: '', label: 'All' },
  { value: 'AccessConditions', label: 'AccessConditions' },
  { value: 'AccessPolicies', label: 'AccessPolicies' },
  { value: 'AgentControllers', label: 'AgentControllers' },
  { value: 'Agents', label: 'Agents' },
  { value: 'Authentication', label: 'Authentication' },
  { value: 'CredentialProvider', label: 'CredentialProvider' },
  { value: 'CredentialProviderIntegrations', label: 'CredentialProviderIntegrations' },
  { value: 'DiscoveryIntegration', label: 'DiscoveryIntegration' },
  { value: 'GlobalPolicyCompliance', label: 'GlobalPolicyCompliance' },
  { value: 'IdentityProviders', label: 'IdentityProviders' },
  { value: 'Integrations', label: 'Integrations' },
  { value: 'LogStreams', label: 'LogStreams' },
  { value: 'PkiSettings', label: 'PkiSettings' },
  { value: 'ResourceSets', label: 'ResourceSets' },
  { value: 'Roles', label: 'Roles' },
  { value: 'Routing', label: 'Routing' },
  { value: 'SignOnPolicies', label: 'SignOnPolicies' },
  { value: 'StandaloneCertificateAuthorities', label: 'StandaloneCertificateAuthorities' },
  { value: 'Tenant', label: 'Tenant' },
  { value: 'TrustProvider', label: 'TrustProvider' },
  { value: 'Users', label: 'Users' },
  { value: 'Workloads', label: 'Workloads' },
] as const

export const AUDIT_LOG_SEVERITY_OPTIONS = [
  { value: '', label: 'All' },
  { value: 'Info', label: 'Info' },
  { value: 'Warn', label: 'Warn' },
  { value: 'Alert', label: 'Alert' },
] as const

export const COMPLIANCE_STATUS_OPTIONS = [
  { value: '', label: 'All' },
  { value: 'Compliant', label: 'Compliant' },
  { value: 'Missing Required', label: 'Missing Required' },
  { value: 'Missing Recommended', label: 'Missing Recommended' },
] as const

export const RESOURCE_SET_OPTIONS = [
  { value: '', label: 'All' },
  { value: 'Default', label: 'Default' },
  { value: 'Production', label: 'Production' },
  { value: 'Staging', label: 'Staging' },
] as const

export const WORKLOAD_PROTOCOL_OPTIONS = [
  { value: '', label: 'All Protocols' },
  { value: 'http', label: 'HTTP' },
  { value: 'postgres', label: 'PostgreSQL' },
  { value: 'mysql', label: 'MySQL' },
  { value: 'redis', label: 'Redis' },
  { value: 'snowflake', label: 'Snowflake' },
  { value: 'oracleDatabase', label: 'Oracle' },
  { value: 'mcp', label: 'MCP' },
  { value: 'tcp', label: 'TCP' },
  { value: 'redshift', label: 'Redshift' },
] as const

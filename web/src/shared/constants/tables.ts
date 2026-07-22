/**
 * Shared column widths for entity list tables (client workloads, server
 * workloads, trust providers, access conditions, credential providers).
 * All 5 have the same 7-column structure: #, Name, Type, Description, Tags, Policies, Status.
 * Using fixed widths ensures columns don't shift when navigating between pages.
 */
export const ENTITY_TABLE_COL_WIDTHS: Record<string, string> = {
  number: 'w-[4%]',
  name: 'w-[20%]',
  type: 'w-[20%]',
  description: 'w-[30%]',
  tags: 'w-[7%]',
  policies: 'w-[9%]',
  status: 'w-[10%]',
  // Server workloads uses 'protocol' key for the type column
  protocol: 'w-[20%]',
  // Access conditions / others may use 'host'
  host: 'w-[20%]',
}

export const CLIENT_WORKLOAD_TABLE_COLUMNS = [
  { key: 'number', label: '#' },
  { key: 'name', label: 'Client Workload Name' },
  { key: 'type', label: 'Client Workload Type' },
  { key: 'description', label: 'Description' },
  { key: 'tags', label: 'Tags' },
  { key: 'policies', label: 'Policies' },
  { key: 'status', label: 'Status' },
] as const

export const CLIENT_WORKLOAD_PAGE_SIZE_OPTIONS = [20, 50, 100] as const

export const ACCESS_CONDITION_TABLE_COLUMNS = [
  { key: 'number', label: '#' },
  { key: 'name', label: 'Access Condition Name' },
  { key: 'type', label: 'Access Condition Type' },
  { key: 'description', label: 'Description' },
  { key: 'tags', label: 'Tags' },
  { key: 'policies', label: 'Policies' },
  { key: 'status', label: 'Status' },
] as const

export const ACCESS_CONDITION_PAGE_SIZE_OPTIONS = [20, 50, 100] as const

export const CREDENTIAL_PROVIDER_TABLE_COLUMNS = [
  { key: 'number', label: '#' },
  { key: 'name', label: 'Credential Provider Name' },
  { key: 'type', label: 'Credential Provider Type' },
  { key: 'description', label: 'Description' },
  { key: 'tags', label: 'Tags' },
  { key: 'policies', label: 'Policies' },
  { key: 'status', label: 'Status' },
] as const

export const CREDENTIAL_PROVIDER_PAGE_SIZE_OPTIONS = [20, 50, 100] as const

export const TRUST_PROVIDER_TABLE_COLUMNS = [
  { key: 'number', label: '#' },
  { key: 'name', label: 'Trust Provider Name' },
  { key: 'type', label: 'Trust Provider Type' },
  { key: 'description', label: 'Description' },
  { key: 'tags', label: 'Tags' },
  { key: 'policies', label: 'Policies' },
  { key: 'status', label: 'Status' },
] as const

export const TRUST_PROVIDER_PAGE_SIZE_OPTIONS = [20, 50, 100] as const

export const SERVER_WORKLOAD_TABLE_COLUMNS = [
  { key: 'number', label: '#' },
  { key: 'name', label: 'Server Workload Name' },
  { key: 'protocol', label: 'Application Protocol' },
  { key: 'host', label: 'Host' },
  { key: 'tags', label: 'Tags' },
  { key: 'policies', label: 'Policies' },
  { key: 'status', label: 'Status' },
] as const

export const SERVER_WORKLOAD_PAGE_SIZE_OPTIONS = [20, 50, 100] as const

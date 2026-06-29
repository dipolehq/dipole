export const POLICY_TABLE_COLUMNS = [
  { key: 'number', label: '#' },
  { key: 'name', label: 'Access Policy Name' },
  { key: 'clientWorkload', label: 'Client Workload' },
  { key: 'trustProvider', label: 'Trust Provider' },
  { key: 'accessCondition', label: 'Access Condition' },
  { key: 'credentialProvider', label: 'Credential Provider' },
  { key: 'serverWorkload', label: 'Server Workload' },
  { key: 'active', label: 'Status' },
] as const

export const PAGE_SIZE_OPTIONS = [15, 25, 50] as const

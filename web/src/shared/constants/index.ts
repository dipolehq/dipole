export const DEFAULT_PAGE_SIZE = 20

export const ROUTES = {
  HOME: '/',
  IDENTITIES: '/identities',
  POLICIES: '/policies',
  AUDIT_LOGS: '/audit-logs',
  INTEGRATIONS: '/integrations',
  SETTINGS: '/settings',
} as const

export * from './policies'
export * from './credential-providers'
export * from './administration'
export * from './tables'
export * from './reporting'

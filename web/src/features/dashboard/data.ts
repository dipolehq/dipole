/**
 * Placeholder dashboard data — will be replaced with real API calls.
 */
export const dashboardData = {
  stats: {
    clientWorkloads: { count: 0, active: 0, inactive: 0 },
    serverWorkloads: { count: 0, active: 0, inactive: 0 },
    accessPolicies: { count: 0, active: 0, inactive: 0 },
    accessConditions: { count: 0, active: 0, inactive: 0 },
    trustProviders: { count: 0, active: 0, inactive: 0 },
    credentialProviders: { count: 0, active: 0, inactive: 0 },
  },
  accessDecisions: {
    authorized: Array(24).fill(0) as number[],
    denied: Array(24).fill(0) as number[],
  },
  workloadEvents: {
    info: Array(72).fill(0) as number[],
    warning: Array(72).fill(0) as number[],
    error: Array(72).fill(0) as number[],
  },
  clientConnections: [] as { name: string; count: number }[],
  serverConnections: [] as { name: string; count: number }[],
  credentialUsage: [] as number[],
  protocolUsage: [] as number[],
  denialReasons: [] as number[],
  attestationResults: [] as number[],
}

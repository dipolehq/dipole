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
    authorized: [] as number[],
    denied: [] as number[],
    timestamps: [] as string[],
  },
  workloadEvents: {
    info: [] as number[],
    warning: [] as number[],
    error: [] as number[],
    timestamps: [] as string[],
  },
  clientConnections: [] as { name: string; value: number }[],
  serverConnections: [] as { name: string; value: number }[],
  credentialUsage: [] as { label: string; value: number }[],
  protocolUsage: [] as { label: string; value: number }[],
  denialReasons: [] as { label: string; value: number }[],
  attestationResults: [] as { label: string; value: number }[],
}

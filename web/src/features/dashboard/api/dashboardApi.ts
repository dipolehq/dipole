import { apiClient } from '@/shared/lib/api-client'

export interface DashboardStatsDTO {
  counts: {
    clientWorkloads: number
    clientInactive: number
    clientUnmanaged: number
    serverWorkloads: number
    serverInactive: number
    serverUnmanaged: number
    trustProviders: number
    trustInactive: number
    credentialProviders: number
    credentialInactive: number
    accessPolicies: number
    accessPoliciesInactive: number
    accessConditions: number
    conditionInactive: number
  }
  recentEvents: {
    last6h: { total: number; authorized: number; denied: number }
    timeSeries: { timestamp: string; info: number; warning: number; error: number }[]
    accessDecisionSeries: { timestamp: string; authorized: number; denied: number }[]
  }
  topClientWorkloads: { id: string; name: string; connections: number }[]
  topServerWorkloads: { id: string; name: string; connections: number }[]
  credentialUsageByType: { type: string; count: number }[]
  connectionsByProtocol: { protocol: string; count: number }[]
  mostConditionFailures: { policyId: string; policyName: string; failures: number }[]
  attestationResults: { result: string; count: number }[]
}

export interface DashboardData {
  stats: {
    clientWorkloads: { count: number; active: number; inactive: number }
    serverWorkloads: { count: number; active: number; inactive: number }
    accessPolicies: { count: number; active: number; inactive: number }
    accessConditions: { count: number; active: number; inactive: number }
    trustProviders: { count: number; active: number; inactive: number }
    credentialProviders: { count: number; active: number; inactive: number }
  }
  accessDecisions: { authorized: number[]; denied: number[]; timestamps: string[] }
  workloadEvents: { info: number[]; warning: number[]; error: number[]; timestamps: string[] }
  clientConnections: { name: string; value: number }[]
  serverConnections: { name: string; value: number }[]
  credentialUsage: { label: string; value: number }[]
  protocolUsage: { label: string; value: number }[]
  denialReasons: { label: string; value: number }[]
  attestationResults: { label: string; value: number }[]
}

function mapToFrontend(dto: DashboardStatsDTO): DashboardData {
  const c = dto.counts

  const stats = {
    clientWorkloads: { count: c.clientWorkloads, active: c.clientWorkloads - c.clientInactive, inactive: c.clientInactive },
    serverWorkloads: { count: c.serverWorkloads, active: c.serverWorkloads - c.serverInactive, inactive: c.serverInactive },
    accessPolicies: { count: c.accessPolicies, active: c.accessPolicies - (c.accessPoliciesInactive ?? 0), inactive: c.accessPoliciesInactive ?? 0 },
    accessConditions: { count: c.accessConditions, active: c.accessConditions - c.conditionInactive, inactive: c.conditionInactive },
    trustProviders: { count: c.trustProviders, active: c.trustProviders - c.trustInactive, inactive: c.trustInactive },
    credentialProviders: { count: c.credentialProviders, active: c.credentialProviders - c.credentialInactive, inactive: c.credentialInactive },
  }

  // Map time series to arrays for charts
  const timeSeries = dto.recentEvents.timeSeries ?? []
  const workloadEvents = {
    info: timeSeries.map((p) => p.info),
    warning: timeSeries.map((p) => p.warning),
    error: timeSeries.map((p) => p.error),
    timestamps: timeSeries.map((p) => p.timestamp),
  }

  const last6h = dto.recentEvents.last6h
  const decisionSeries = dto.recentEvents.accessDecisionSeries ?? []
  const accessDecisions = {
    authorized: decisionSeries.length > 0
      ? decisionSeries.map((p) => p.authorized)
      : [last6h.authorized],
    denied: decisionSeries.length > 0
      ? decisionSeries.map((p) => p.denied)
      : [last6h.denied],
    timestamps: decisionSeries.map((p) => p.timestamp),
  }

  const clientConnections = (dto.topClientWorkloads ?? []).map((w) => ({ name: w.name, value: w.connections }))
  const serverConnections = (dto.topServerWorkloads ?? []).map((w) => ({ name: w.name, value: w.connections }))

  const credentialUsage = (dto.credentialUsageByType ?? []).map((item) => ({ label: item.type, value: item.count }))
  const protocolUsage = (dto.connectionsByProtocol ?? []).map((p) => ({ label: p.protocol, value: p.count }))

  // Denial reasons from mostConditionFailures
  const denialReasons = (dto.mostConditionFailures ?? []).map((f) => ({ label: f.policyName, value: f.failures }))

  return {
    stats,
    accessDecisions,
    workloadEvents,
    clientConnections,
    serverConnections,
    credentialUsage,
    protocolUsage,
    denialReasons,
    attestationResults: (dto.attestationResults ?? []).map((a) => ({ label: a.result, value: a.count })),
  }
}

export async function fetchDashboardStats(timeRange = '6h'): Promise<DashboardData> {
  const dto = await apiClient.get<DashboardStatsDTO>(`/dashboard?timeRange=${timeRange}`)
  return mapToFrontend(dto)
}

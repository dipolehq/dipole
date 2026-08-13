import { apiClient } from '@/shared/lib/api-client'

export interface EventQueryParams {
  page: number
  perPage: number
  severity?: string
  eventType?: string
  startDate?: string
  endDate?: string
  source?: string
}

interface AuthEventDTO {
  id: string
  eventType: string
  contextId: string
  severity: string
  clientIp: string
  timestamp: string
  payload: Record<string, unknown>
  resourceSetId: string | null
}

interface WorkloadEventDTO {
  id: string
  connectionId: string
  eventType: string
  policyId: string | null
  clientWorkloadId: string | null
  serverWorkloadId: string | null
  outcome: string
  severity: string
  timestamp: string
  payload: Record<string, unknown>
  resourceSetId: string | null
}

interface AuditLogDTO {
  id: string
  actorEmail: string
  category: string
  activity: string
  targetType: string
  targetName: string
  severity: string
  clientIp: string
  timestamp: string
}

interface PaginatedResponse<T> {
  items: T[]
  total: number
}

export interface AuthEventItem {
  id: string
  eventType: string
  contextId: string
  severity: string
  clientIp: string
  timestamp: string
  payload: Record<string, unknown>
  clientWorkloadName?: string
  serverWorkloadName?: string
}

export interface WorkloadEventItem {
  id: string
  connectionId: string
  eventType: string
  policyId: string | null
  clientWorkloadName: string
  serverWorkloadName: string
  outcome: string
  severity: string
  timestamp: string
  payload: Record<string, unknown>
}

export interface AuditLogItem {
  id: string
  actorEmail: string
  category: string
  activity: string
  targetType: string
  targetName: string
  severity: string
  clientIp: string
  timestamp: string
}

function buildEventQuery(params: EventQueryParams): string {
  const q = new URLSearchParams()
  q.set('page', String(params.page))
  q.set('perPage', String(params.perPage))
  if (params.severity) q.set('severity', params.severity)
  if (params.eventType) q.set('eventType', params.eventType)
  if (params.startDate) q.set('startDate', params.startDate)
  if (params.endDate) q.set('endDate', params.endDate)
  if (params.source) q.set('source', params.source)
  return q.toString()
}

export async function fetchAuthEvents(params: EventQueryParams): Promise<{ items: AuthEventItem[]; total: number }> {
  const res = await apiClient.get<PaginatedResponse<AuthEventDTO>>(`/events/authorization?${buildEventQuery(params)}`)
  const items: AuthEventItem[] = (res.items ?? []).map((dto) => {
    const payload = dto.payload ?? {}
    return {
      id: dto.id,
      eventType: dto.eventType,
      contextId: dto.contextId,
      severity: dto.severity,
      clientIp: dto.clientIp,
      timestamp: dto.timestamp,
      payload,
      clientWorkloadName: (payload.clientWorkload as Record<string, string>)?.name,
      serverWorkloadName: (payload.serverWorkload as Record<string, string>)?.name,
    }
  })
  return { items, total: res.total }
}

export async function fetchWorkloadEvents(params: EventQueryParams): Promise<{ items: WorkloadEventItem[]; total: number }> {
  const res = await apiClient.get<PaginatedResponse<WorkloadEventDTO>>(`/events/workload?${buildEventQuery(params)}`)
  const items: WorkloadEventItem[] = (res.items ?? []).map((dto) => {
    const payload = dto.payload ?? {}
    return {
      id: dto.id,
      connectionId: dto.connectionId,
      eventType: dto.eventType,
      policyId: dto.policyId,
      clientWorkloadName: (payload.workload as Record<string, Record<string, string>>)?.client?.name ?? (payload.clientWorkload as Record<string, string>)?.name ?? 'Unknown',
      serverWorkloadName: (payload.workload as Record<string, Record<string, string>>)?.server?.name ?? (payload.serverWorkload as Record<string, string>)?.name ?? 'Unknown',
      outcome: dto.outcome,
      severity: dto.severity,
      timestamp: dto.timestamp,
      payload,
    }
  })
  return { items, total: res.total }
}

export async function fetchAuditLogs(params: EventQueryParams): Promise<{ items: AuditLogItem[]; total: number }> {
  const res = await apiClient.get<PaginatedResponse<AuditLogDTO>>(`/events/audit?${buildEventQuery(params)}`)
  const items: AuditLogItem[] = (res.items ?? []).map((dto) => ({
    id: dto.id,
    actorEmail: dto.actorEmail,
    category: dto.category,
    activity: dto.activity,
    targetType: dto.targetType,
    targetName: dto.targetName,
    severity: dto.severity,
    clientIp: dto.clientIp,
    timestamp: dto.timestamp,
  }))
  return { items, total: res.total }
}

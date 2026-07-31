import { apiClient } from '@/shared/lib/api-client'

const BASE = '/agent-proxies'

export interface AgentProxyDTO {
  id: string
  status: string
  agentControllerId?: string
  lastHealthAt?: string
  resourceSetId?: string
  registeredAt?: string
}

interface AgentProxyListResponse {
  items: AgentProxyDTO[]
  total: number
}

export async function fetchAgentProxies(page = 1, perPage = 20, search = ''): Promise<AgentProxyListResponse> {
  const params = new URLSearchParams({ page: String(page), perPage: String(perPage) })
  if (search) params.set('search', search)
  return apiClient.get<AgentProxyListResponse>(`${BASE}?${params.toString()}`)
}

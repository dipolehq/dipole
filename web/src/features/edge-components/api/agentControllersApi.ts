import { apiClient, ApiError } from '@/shared/lib/api-client'
import type { FormTag } from '@/shared/components/forms/types'

const BASE = '/agent-controllers'

// --- DTOs ---

interface AgentControllerDTO {
  id: string
  name: string
  description: string
  isActive: boolean
  status: string
  deviceCode?: string
  trustProviderId?: string
  tlsHostname?: string
  tags: { key: string; value: string }[]
  resourceSetId?: string
  createdAt: string
  updatedAt: string
}

interface ListResponseDTO {
  items: AgentControllerDTO[]
  total: number
  page: number
  perPage: number
}

// --- Public types ---

export interface AgentControllerListItem {
  id: string
  name: string
  description: string
  status: string
  deviceCode?: string
  trustProviderId?: string
  tlsHostname?: string
  tags: { key: string; value: string }[]
  createdAt: string
  updatedAt: string
}

export interface AgentControllerDetail {
  id: string
  name: string
  description: string
  isActive: boolean
  status: string
  deviceCode?: string
  trustProviderId?: string
  tlsHostname?: string
  tags: { key: string; value: string }[]
  resourceSetId?: string
  createdAt: string
  updatedAt: string
}

export interface AgentControllerInput {
  name: string
  description: string
  isActive: boolean
  trustProviderId?: string
  tlsHostname?: string
  tags: FormTag[]
}

export interface FetchAgentControllersResult {
  items: AgentControllerListItem[]
  total: number
}

// --- API functions ---

export async function fetchAgentControllers(
  page = 1,
  perPage = 20,
  search = '',
): Promise<FetchAgentControllersResult> {
  const params = new URLSearchParams({ page: String(page), perPage: String(perPage) })
  if (search) params.set('search', search)
  const res = await apiClient.get<ListResponseDTO>(`${BASE}?${params}`)
  return { items: res.items, total: res.total }
}

export async function fetchAgentController(id: string): Promise<AgentControllerDetail | null> {
  try {
    return await apiClient.get<AgentControllerDetail>(`${BASE}/${id}`)
  } catch (err) {
    if (err instanceof ApiError && err.status === 404) return null
    throw err
  }
}

export async function createAgentController(input: AgentControllerInput): Promise<AgentControllerDetail> {
  return apiClient.post<AgentControllerDetail>(BASE, {
    name: input.name,
    description: input.description,
    isActive: input.isActive,
    trustProviderId: input.trustProviderId || undefined,
    tlsHostname: input.tlsHostname || undefined,
    tags: input.tags.map((t) => ({ key: t.key, value: t.value })),
  })
}

export async function saveAgentController(id: string, input: AgentControllerInput): Promise<AgentControllerDetail> {
  return apiClient.put<AgentControllerDetail>(`${BASE}/${id}`, {
    name: input.name,
    description: input.description,
    isActive: input.isActive,
    trustProviderId: input.trustProviderId || undefined,
    tlsHostname: input.tlsHostname || undefined,
    tags: input.tags.map((t) => ({ key: t.key, value: t.value })),
  })
}

export async function deleteAgentController(id: string): Promise<void> {
  await apiClient.delete(`${BASE}/${id}`)
}

export async function regenerateDeviceCode(controllerId: string): Promise<string> {
  const res = await apiClient.post<{ deviceCode: string }>(`${BASE}/${controllerId}/device-code`)
  return res.deviceCode
}

export { apiErrorMessage } from '@/shared/lib/api-client'

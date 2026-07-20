import { apiClient, ApiError } from '@/shared/lib/api-client'

const BASE = '/log-streams'

interface LogStreamDTO {
  id: string
  name: string
  description: string
  isActive: boolean
  eventType: string
  destinationType: string
  config: Record<string, unknown>
  createdAt: string
  updatedAt: string
}

interface ListResponseDTO {
  items: LogStreamDTO[]
  total: number
  page: number
  perPage: number
}

export interface LogStreamListItem {
  id: string
  name: string
  description: string
  isActive: boolean
  eventType: string
  destinationType: string
  config: Record<string, unknown>
  createdAt: string
  updatedAt: string
}

export interface LogStreamDetail {
  id: string
  name: string
  description: string
  isActive: boolean
  eventType: string
  destinationType: string
  config: Record<string, unknown>
  createdAt: string
  updatedAt: string
}

export interface LogStreamInput {
  name: string
  description: string
  isActive: boolean
  eventType: string
  destinationType: string
  config: Record<string, unknown>
}

export interface FetchLogStreamsResult {
  items: LogStreamListItem[]
  total: number
}

export async function fetchLogStreams(page = 1, perPage = 20, search = ''): Promise<FetchLogStreamsResult> {
  const params = new URLSearchParams({ page: String(page), perPage: String(perPage) })
  if (search) params.set('search', search)
  const res = await apiClient.get<ListResponseDTO>(`${BASE}?${params}`)
  return { items: res.items, total: res.total }
}

export async function fetchLogStream(id: string): Promise<LogStreamDetail | null> {
  try {
    return await apiClient.get<LogStreamDetail>(`${BASE}/${id}`)
  } catch (err) {
    if (err instanceof ApiError && err.status === 404) return null
    throw err
  }
}

export async function createLogStream(input: LogStreamInput): Promise<LogStreamDetail> {
  return apiClient.post<LogStreamDetail>(BASE, {
    name: input.name,
    description: input.description,
    isActive: input.isActive,
    eventType: input.eventType,
    destinationType: input.destinationType,
    config: input.config,
  })
}

export async function saveLogStream(id: string, input: LogStreamInput): Promise<LogStreamDetail> {
  return apiClient.put<LogStreamDetail>(`${BASE}/${id}`, {
    name: input.name,
    description: input.description,
    isActive: input.isActive,
    eventType: input.eventType,
    destinationType: input.destinationType,
    config: input.config,
  })
}

export async function deleteLogStream(id: string): Promise<void> {
  await apiClient.delete(`${BASE}/${id}`)
}

export { apiErrorMessage } from '@/shared/lib/api-client'

import { apiClient, ApiError } from '@/shared/lib/api-client'

const BASE = '/resource-sets'

interface ListResponseDTO {
  items: ResourceSetListItem[]
  total: number
  page: number
  perPage: number
}

export interface ResourceSetListItem {
  id: string
  name: string
  description: string
  accessPolicies: number
  clientWorkloads: number
  trustProviders: number
  accessConditions: number
  credentialProviders: number
  serverWorkloads: number
  createdAt: string
  updatedAt: string
}

export interface ResourceSetDetail {
  id: string
  name: string
  description: string
  isDefault: boolean
  createdAt: string
  updatedAt: string
}

export interface ResourceSetInput {
  name: string
  description: string
}

export interface FetchResourceSetsResult {
  items: ResourceSetListItem[]
  total: number
}

export async function fetchResourceSets(page = 1, perPage = 20, search = ''): Promise<FetchResourceSetsResult> {
  const params = new URLSearchParams({ page: String(page), perPage: String(perPage) })
  if (search) params.set('search', search)
  const res = await apiClient.get<ListResponseDTO>(`${BASE}?${params}`)
  return { items: res.items, total: res.total }
}

export async function fetchResourceSet(id: string): Promise<ResourceSetDetail | null> {
  try {
    return await apiClient.get<ResourceSetDetail>(`${BASE}/${id}`)
  } catch (err) {
    if (err instanceof ApiError && err.status === 404) return null
    throw err
  }
}

export async function createResourceSet(input: ResourceSetInput): Promise<ResourceSetDetail> {
  return apiClient.post<ResourceSetDetail>(BASE, {
    name: input.name,
    description: input.description,
  })
}

export async function saveResourceSet(id: string, input: ResourceSetInput): Promise<ResourceSetDetail> {
  return apiClient.put<ResourceSetDetail>(`${BASE}/${id}`, {
    name: input.name,
    description: input.description,
  })
}

export async function deleteResourceSet(id: string): Promise<void> {
  await apiClient.delete(`${BASE}/${id}`)
}

export { apiErrorMessage } from '@/shared/lib/api-client'

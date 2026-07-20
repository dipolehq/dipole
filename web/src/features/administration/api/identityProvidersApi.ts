import { apiClient, ApiError } from '@/shared/lib/api-client'

const BASE = '/identity-providers'

interface IdpDTO {
  id: string
  name: string
  description: string
  isActive: boolean
  type: string
  config: Record<string, unknown>
  createdAt: string
  updatedAt: string
}

interface ListResponseDTO {
  items: IdpDTO[]
  total: number
  page: number
  perPage: number
}

export interface IdpListItem {
  id: string
  name: string
  description: string
  isActive: boolean
  type: string
  config: Record<string, unknown>
  createdAt: string
  updatedAt: string
}

export interface IdpDetail {
  id: string
  name: string
  description: string
  isActive: boolean
  type: string
  config: Record<string, unknown>
  createdAt: string
  updatedAt: string
}

export interface IdpInput {
  name: string
  description: string
  isActive: boolean
  type: string
  config: Record<string, unknown>
}

export interface FetchIdpsResult {
  items: IdpListItem[]
  total: number
}

export async function fetchIdentityProviders(page = 1, perPage = 20, search = ''): Promise<FetchIdpsResult> {
  const params = new URLSearchParams({ page: String(page), perPage: String(perPage) })
  if (search) params.set('search', search)
  const res = await apiClient.get<ListResponseDTO>(`${BASE}?${params}`)
  return { items: res.items, total: res.total }
}

export async function fetchIdentityProvider(id: string): Promise<IdpDetail | null> {
  try {
    return await apiClient.get<IdpDetail>(`${BASE}/${id}`)
  } catch (err) {
    if (err instanceof ApiError && err.status === 404) return null
    throw err
  }
}

export async function createIdentityProvider(input: IdpInput): Promise<IdpDetail> {
  return apiClient.post<IdpDetail>(BASE, {
    name: input.name,
    description: input.description,
    isActive: input.isActive,
    type: input.type,
    config: input.config,
  })
}

export async function saveIdentityProvider(id: string, input: IdpInput): Promise<IdpDetail> {
  return apiClient.put<IdpDetail>(`${BASE}/${id}`, {
    name: input.name,
    description: input.description,
    isActive: input.isActive,
    type: input.type,
    config: input.config,
  })
}

export async function deleteIdentityProvider(id: string): Promise<void> {
  await apiClient.delete(`${BASE}/${id}`)
}

export { apiErrorMessage } from '@/shared/lib/api-client'

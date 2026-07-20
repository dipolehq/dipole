import { apiClient, ApiError } from '@/shared/lib/api-client'

const BASE = '/roles'

// --- DTOs ---

interface RoleDTO {
  id: string
  name: string
  description: string
  isActive: boolean
  permissions: Record<string, string>
  createdAt: string
  updatedAt: string
}

interface ListResponseDTO {
  items: RoleDTO[]
  total: number
  page: number
  perPage: number
}

// --- Public types ---

export interface RoleListItem {
  id: string
  name: string
  description: string
  isActive: boolean
  permissions: Record<string, string>
  createdAt: string
  updatedAt: string
}

export interface RoleDetail {
  id: string
  name: string
  description: string
  isActive: boolean
  permissions: Record<string, string>
  createdAt: string
  updatedAt: string
}

export interface RoleInput {
  name: string
  description: string
  isActive: boolean
  permissions: Record<string, string>
}

export interface FetchRolesResult {
  items: RoleListItem[]
  total: number
}

// --- API functions ---

export async function fetchRoles(page = 1, perPage = 20, search = ''): Promise<FetchRolesResult> {
  const params = new URLSearchParams({ page: String(page), perPage: String(perPage) })
  if (search) params.set('search', search)
  const res = await apiClient.get<ListResponseDTO>(`${BASE}?${params}`)
  return { items: res.items, total: res.total }
}

export async function fetchRole(id: string): Promise<RoleDetail | null> {
  try {
    return await apiClient.get<RoleDetail>(`${BASE}/${id}`)
  } catch (err) {
    if (err instanceof ApiError && err.status === 404) return null
    throw err
  }
}

export async function createRole(input: RoleInput): Promise<RoleDetail> {
  return apiClient.post<RoleDetail>(BASE, {
    name: input.name,
    description: input.description,
    isActive: input.isActive,
    permissions: input.permissions,
  })
}

export async function saveRole(id: string, input: RoleInput): Promise<RoleDetail> {
  return apiClient.put<RoleDetail>(`${BASE}/${id}`, {
    name: input.name,
    description: input.description,
    isActive: input.isActive,
    permissions: input.permissions,
  })
}

export async function deleteRole(id: string): Promise<void> {
  await apiClient.delete(`${BASE}/${id}`)
}

export { apiErrorMessage } from '@/shared/lib/api-client'

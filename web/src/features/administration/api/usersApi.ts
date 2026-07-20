import { apiClient, ApiError } from '@/shared/lib/api-client'

const BASE = '/users'

// --- DTOs ---

interface UserDTO {
  id: string
  firstName: string
  lastName: string
  email: string
  phone: string
  isActive: boolean
  roleId?: string
  createdAt: string
  updatedAt: string
}

interface ListResponseDTO {
  items: UserDTO[]
  total: number
  page: number
  perPage: number
}

// --- Public types ---

export interface UserListItem {
  id: string
  firstName: string
  lastName: string
  email: string
  phone: string
  isActive: boolean
  roleId?: string
  createdAt: string
  updatedAt: string
}

export interface UserDetail {
  id: string
  firstName: string
  lastName: string
  email: string
  phone: string
  isActive: boolean
  roleId?: string
  createdAt: string
  updatedAt: string
}

export interface UserInput {
  firstName: string
  lastName: string
  email: string
  phone: string
  isActive: boolean
  roleId?: string
}

export interface FetchUsersResult {
  items: UserListItem[]
  total: number
}

// --- API functions ---

export async function fetchUsers(page = 1, perPage = 20, search = ''): Promise<FetchUsersResult> {
  const params = new URLSearchParams({ page: String(page), perPage: String(perPage) })
  if (search) params.set('search', search)
  const res = await apiClient.get<ListResponseDTO>(`${BASE}?${params}`)
  return { items: res.items, total: res.total }
}

export async function fetchUser(id: string): Promise<UserDetail | null> {
  try {
    return await apiClient.get<UserDetail>(`${BASE}/${id}`)
  } catch (err) {
    if (err instanceof ApiError && err.status === 404) return null
    throw err
  }
}

export async function createUser(input: UserInput): Promise<UserDetail> {
  return apiClient.post<UserDetail>(BASE, {
    firstName: input.firstName,
    lastName: input.lastName,
    email: input.email,
    phone: input.phone,
    isActive: input.isActive,
    roleId: input.roleId || undefined,
  })
}

export async function saveUser(id: string, input: UserInput): Promise<UserDetail> {
  return apiClient.put<UserDetail>(`${BASE}/${id}`, {
    firstName: input.firstName,
    lastName: input.lastName,
    email: input.email,
    phone: input.phone,
    isActive: input.isActive,
    roleId: input.roleId || undefined,
  })
}

export async function deleteUser(id: string): Promise<void> {
  await apiClient.delete(`${BASE}/${id}`)
}

export { apiErrorMessage } from '@/shared/lib/api-client'

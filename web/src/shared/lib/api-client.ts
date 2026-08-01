/**
 * Centralized API client. All network requests go through this module.
 * Features use this via their own api/ layer — never import fetch/axios directly.
 */

const BASE_URL = '/api/v1'

const DEFAULT_RESOURCE_SET_ID = 'ffffffff-ffff-ffff-ffff-ffffffffffff'

let activeResourceSetId: string = DEFAULT_RESOURCE_SET_ID

export function setActiveResourceSet(id: string) {
  activeResourceSetId = id
}

export function getActiveResourceSet(): string {
  return activeResourceSetId
}

interface RequestOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH'
  body?: unknown
  headers?: Record<string, string>
}

class ApiError extends Error {
  status: number
  statusText: string
  data?: unknown

  constructor(status: number, statusText: string, data?: unknown) {
    super(`API Error ${status}: ${statusText}`)
    this.name = 'ApiError'
    this.status = status
    this.statusText = statusText
    this.data = data
  }
}

function getAuthHeaders(): Record<string, string> {
  const headers: Record<string, string> = {}
  const token = sessionStorage.getItem('dipole_token')
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }
  if (activeResourceSetId && activeResourceSetId !== DEFAULT_RESOURCE_SET_ID) {
    headers['X-Dipole-ResourceSet'] = activeResourceSetId
  }
  return headers
}

async function request<T>(endpoint: string, options: RequestOptions = {}): Promise<T> {
  const { method = 'GET', body, headers = {} } = options

  const response = await fetch(`${BASE_URL}${endpoint}`, {
    method,
    headers: {
      'Content-Type': 'application/json',
      ...getAuthHeaders(),
      ...headers,
    },
    body: body ? JSON.stringify(body) : undefined,
  })

  if (!response.ok) {
    if (response.status === 401) {
      sessionStorage.removeItem('dipole_token')
      sessionStorage.removeItem('dipole_user')
      if (window.location.pathname !== '/login') {
        window.location.href = '/login'
      }
    }
    const data = await response.json().catch(() => undefined)
    throw new ApiError(response.status, response.statusText, data)
  }

  // Handle 204 No Content
  if (response.status === 204) {
    return undefined as T
  }

  return response.json() as Promise<T>
}

export function apiErrorMessage(err: unknown, fallback = 'Something went wrong'): string {
  if (err instanceof ApiError) {
    const data = err.data as { error?: { message?: string } } | undefined
    if (data?.error?.message) return data.error.message
  }
  return fallback
}

export const apiClient = {
  get: <T>(endpoint: string, headers?: Record<string, string>) =>
    request<T>(endpoint, { method: 'GET', headers }),

  post: <T>(endpoint: string, body?: unknown, headers?: Record<string, string>) =>
    request<T>(endpoint, { method: 'POST', body, headers }),

  put: <T>(endpoint: string, body?: unknown, headers?: Record<string, string>) =>
    request<T>(endpoint, { method: 'PUT', body, headers }),

  patch: <T>(endpoint: string, body?: unknown, headers?: Record<string, string>) =>
    request<T>(endpoint, { method: 'PATCH', body, headers }),

  delete: <T>(endpoint: string, headers?: Record<string, string>) =>
    request<T>(endpoint, { method: 'DELETE', headers }),
}

export { ApiError }

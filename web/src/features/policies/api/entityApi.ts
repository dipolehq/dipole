/**
 * Shared API helpers for fetching entity lists used in the policy form selectors.
 * Each entity endpoint follows the same pagination pattern:
 *   GET /api/v1/<resource>?page=N&perPage=N&search=...
 *   Response: { items: [...], total, page, perPage }
 */
import { apiClient } from '@/shared/lib/api-client'

// ---------- Generic paginated list ----------

interface PaginatedResponse<T> {
  items: T[]
  total: number
  page: number
  perPage: number
}

export interface EntityListParams {
  page?: number
  perPage?: number
  search?: string
}

function buildQuery(params: EntityListParams): string {
  const q = new URLSearchParams()
  q.set('page', String(params.page ?? 1))
  q.set('perPage', String(params.perPage ?? 25))
  if (params.search) q.set('search', params.search)
  return q.toString()
}

// ---------- Client Workloads ----------

export interface ClientWorkloadListItem {
  id: string
  name: string
  description: string
  isActive: boolean
  identifiers: { type: string; value: string }[]
  tags: { key: string; value: string }[]
  policyCount: number
}

export async function fetchClientWorkloads(params: EntityListParams = {}) {
  return apiClient.get<PaginatedResponse<ClientWorkloadListItem>>(
    `/client-workloads?${buildQuery(params)}`,
  )
}

export async function fetchClientWorkload(id: string) {
  return apiClient.get<ClientWorkloadListItem>(`/client-workloads/${id}`)
}

// ---------- Server Workloads ----------

export interface ServerWorkloadListItem {
  id: string
  name: string
  description: string
  isActive: boolean
  host: string
  port: number
  appProtocol: string
  transportProtocol: string
  tls: boolean
  tlsVerification: string
  requestedPort: number
  requestedTls: boolean
  authMethod: string
  authScheme: string
  authConfig: Record<string, string>
  httpHeaders: Record<string, string>
  urlPath: string
  tags: { key: string; value: string }[]
  policyCount: number
}

export async function fetchServerWorkloads(params: EntityListParams = {}) {
  return apiClient.get<PaginatedResponse<ServerWorkloadListItem>>(
    `/server-workloads?${buildQuery(params)}`,
  )
}

export async function fetchServerWorkload(id: string) {
  return apiClient.get<ServerWorkloadListItem>(`/server-workloads/${id}`)
}

// ---------- Trust Providers ----------

export interface TrustProviderListItem {
  id: string
  name: string
  description: string
  isActive: boolean
  type: string
  config: unknown
  matchRules: { attribute: string; value: string }[]
  edgeSdkClientId: string
  edgeSdkAudience: string
  tags: { key: string; value: string }[]
  policyCount: number
}

export async function fetchTrustProviders(params: EntityListParams = {}) {
  return apiClient.get<PaginatedResponse<TrustProviderListItem>>(
    `/trust-providers?${buildQuery(params)}`,
  )
}

export async function fetchTrustProvider(id: string) {
  return apiClient.get<TrustProviderListItem>(`/trust-providers/${id}`)
}

// ---------- Credential Providers ----------

export interface CredentialProviderListItem {
  id: string
  name: string
  description: string
  isActive: boolean
  type: string
  tags: { key: string; value: string }[]
  policyCount: number
}

export async function fetchCredentialProviders(params: EntityListParams = {}) {
  return apiClient.get<PaginatedResponse<CredentialProviderListItem>>(
    `/credential-providers?${buildQuery(params)}`,
  )
}

export async function fetchCredentialProvider(id: string) {
  return apiClient.get<{ id: string; name: string; description: string; isActive: boolean; type: string; config: unknown; tags: { key: string; value: string }[] }>(
    `/credential-providers/${id}`,
  )
}

// ---------- Access Conditions ----------

export interface AccessConditionListItem {
  id: string
  name: string
  description: string
  isActive: boolean
  type: string
  config: unknown
  tags: { key: string; value: string }[]
  policyCount: number
}

export async function fetchAccessConditions(params: EntityListParams = {}) {
  return apiClient.get<PaginatedResponse<AccessConditionListItem>>(
    `/access-conditions?${buildQuery(params)}`,
  )
}

export async function fetchAccessCondition(id: string) {
  return apiClient.get<AccessConditionListItem>(`/access-conditions/${id}`)
}

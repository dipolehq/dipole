import { apiClient, ApiError } from '@/shared/lib/api-client'

export interface ServerWorkloadListItem {
  id: string
  name: string
  appProtocol: string
  description: string
  host: string
  tags: number
  policies: number
  isActive: boolean
}
import type { ServerWorkloadFormValues, FormTag, FormHeader } from '@/shared/components/forms/types'

const BASE = '/server-workloads'

interface TagDTO {
  key: string
  value: string
}

interface ServerWorkloadDTO {
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
  tags: TagDTO[]
  policyCount: number
  resourceSetId?: string
  createdAt: string
  updatedAt: string
}

interface ListResponseDTO {
  items: ServerWorkloadDTO[]
  total: number
  page: number
  perPage: number
}

function toListItem(dto: ServerWorkloadDTO): ServerWorkloadListItem {
  return {
    id: dto.id,
    name: dto.name,
    appProtocol: dto.appProtocol || '—',
    description: dto.description,
    host: dto.host,
    tags: dto.tags?.length ?? 0,
    policies: dto.policyCount ?? 0,
    isActive: dto.isActive,
  }
}

function toFormValues(dto: ServerWorkloadDTO): ServerWorkloadFormValues {
  const authConfig = dto.authConfig ?? {}
  return {
    name: dto.name,
    description: dto.description,
    host: dto.host,
    appProtocol: dto.appProtocol,
    transportProtocol: dto.transportProtocol || 'TCP',
    port: dto.port ? String(dto.port) : '',
    forwardToPort: dto.requestedPort ? String(dto.requestedPort) : '',
    tls: dto.tls,
    forwardTls: dto.requestedTls,
    forwardTlsVerification: dto.tlsVerification || 'full',
    authMethod: dto.authMethod,
    authScheme: dto.authScheme,
    headerName: authConfig.header_name ?? authConfig.query_param ?? '',
    urlPath: dto.urlPath,
    staticHeaders: mapToHeaders(dto.httpHeaders),
    tags: fromTagDTOs(dto.tags),
  }
}

function buildAuthConfig(v: ServerWorkloadFormValues): Record<string, string> {
  const name = v.headerName.trim()
  if (!name) return {}
  if (v.authMethod === 'api_key' && v.authScheme === 'query') return { query_param: name }
  if (v.authMethod === 'api_key' && v.authScheme === 'header') return { header_name: name }
  if (v.authMethod === 'http_auth' && v.authScheme === 'header') return { header_name: name }
  return {}
}

function headersToMap(headers: FormHeader[]): Record<string, string> {
  const map: Record<string, string> = {}
  for (const h of headers) {
    if (h.name.trim()) map[h.name.trim()] = h.value
  }
  return map
}

function mapToHeaders(map: Record<string, string> | undefined): FormHeader[] {
  return Object.entries(map ?? {}).map(([name, value]) => ({ name, value }))
}

function toTagDTOs(tags: FormTag[]): TagDTO[] {
  return tags.filter((t) => t.key.trim()).map((t) => ({ key: t.key.trim(), value: t.value }))
}

function fromTagDTOs(tags: TagDTO[] | undefined): FormTag[] {
  return (tags ?? []).map((t) => ({ key: t.key, value: t.value }))
}

function toMetadata(dto: ServerWorkloadDTO): ServerWorkloadMetadata {
  return {
    entityId: dto.id,
    createdAt: dto.createdAt,
    createdBy: '—',
    lastModifiedAt: dto.updatedAt,
    lastModifiedBy: '—',
    isActive: dto.isActive,
  }
}

function parsePort(value: string): number {
  const n = Number.parseInt(value, 10)
  return Number.isNaN(n) ? 0 : n
}

function toRequestBody(input: ServerWorkloadInput) {
  const v = input.values
  return {
    name: v.name,
    description: v.description,
    isActive: input.isActive,
    host: v.host,
    port: parsePort(v.port),
    appProtocol: v.appProtocol,
    transportProtocol: v.transportProtocol || 'TCP',
    tls: v.tls,
    tlsVerification: v.forwardTlsVerification,
    requestedPort: parsePort(v.forwardToPort),
    requestedTls: v.forwardTls,
    authMethod: v.authMethod,
    authScheme: v.authScheme,
    authConfig: buildAuthConfig(v),
    httpHeaders: headersToMap(v.staticHeaders),
    urlPath: v.urlPath,
    tags: toTagDTOs(v.tags),
  }
}

export interface ServerWorkloadInput {
  values: ServerWorkloadFormValues
  isActive: boolean
}

export interface FetchServerWorkloadsResult {
  items: ServerWorkloadListItem[]
  total: number
}

export interface ServerWorkloadMetadata {
  entityId: string
  createdAt: string
  createdBy: string
  lastModifiedAt: string
  lastModifiedBy: string
  isActive: boolean
}

export interface FetchServerWorkloadDetailResult {
  values: ServerWorkloadFormValues
  metadata: ServerWorkloadMetadata
}

export async function fetchServerWorkloads(
  page = 1,
  perPage = 20,
  search = '',
): Promise<FetchServerWorkloadsResult> {
  const params = new URLSearchParams({ page: String(page), perPage: String(perPage) })
  if (search) params.set('search', search)
  const res = await apiClient.get<ListResponseDTO>(`${BASE}?${params}`)
  return { items: res.items.map(toListItem), total: res.total }
}

export async function fetchServerWorkload(
  id: string,
): Promise<FetchServerWorkloadDetailResult | null> {
  try {
    const dto = await apiClient.get<ServerWorkloadDTO>(`${BASE}/${id}`)
    return { values: toFormValues(dto), metadata: toMetadata(dto) }
  } catch (err) {
    if (err instanceof ApiError && err.status === 404) return null
    throw err
  }
}

export async function createServerWorkload(input: ServerWorkloadInput): Promise<void> {
  await apiClient.post<ServerWorkloadDTO>(BASE, toRequestBody(input))
}

export async function saveServerWorkload(id: string, input: ServerWorkloadInput): Promise<void> {
  await apiClient.put<ServerWorkloadDTO>(`${BASE}/${id}`, toRequestBody(input))
}

export { apiErrorMessage } from '@/shared/lib/api-client'

export async function deleteServerWorkload(id: string): Promise<void> {
  await apiClient.delete(`${BASE}/${id}`)
}

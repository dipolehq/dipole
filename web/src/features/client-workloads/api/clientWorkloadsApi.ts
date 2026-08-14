import { apiClient, ApiError } from '@/shared/lib/api-client'
import { CLIENT_IDENTIFIER_TYPES } from '@/shared/constants/policies'
import { groupIdentifiers, flattenIdentifiers } from '@/shared/lib/identifiers'

export interface ClientWorkloadListItem {
  id: string
  name: string
  type: string
  description: string
  tags: number
  policies: number
  isActive: boolean
}
import type { ClientWorkloadFormValues, FormTag } from '@/shared/components/forms/types'

const BASE = '/client-workloads'

interface IdentifierDTO {
  type: string
  value: string
}

interface TagDTO {
  key: string
  value: string
}

interface ClientWorkloadDTO {
  id: string
  name: string
  description: string
  isActive: boolean
  identifiers: IdentifierDTO[]
  tags: TagDTO[]
  policyCount: number
  resourceSetId?: string
  createdAt: string
  updatedAt: string
}

interface ListResponseDTO {
  items: ClientWorkloadDTO[]
  total: number
  page: number
  perPage: number
}

const TYPE_LABELS: Record<string, string> = Object.fromEntries(
  CLIENT_IDENTIFIER_TYPES.map((t) => [t.value, t.label]),
)

function labelForType(type: string): string {
  return TYPE_LABELS[type] ?? type
}

function toTagDTOs(tags: FormTag[]): TagDTO[] {
  return tags.filter((t) => t.key.trim()).map((t) => ({ key: t.key.trim(), value: t.value }))
}

function fromTagDTOs(tags: TagDTO[] | undefined): FormTag[] {
  return (tags ?? []).map((t) => ({ key: t.key, value: t.value }))
}

function toListItem(dto: ClientWorkloadDTO): ClientWorkloadListItem {
  const firstIdentifier = dto.identifiers?.[0]
  return {
    id: dto.id,
    name: dto.name,
    type: firstIdentifier ? labelForType(firstIdentifier.type) : '—',
    description: dto.description,
    tags: dto.tags?.length ?? 0,
    policies: dto.policyCount ?? 0,
    isActive: dto.isActive,
  }
}

function toFormValues(dto: ClientWorkloadDTO): ClientWorkloadFormValues {
  return {
    name: dto.name,
    description: dto.description,
    identifiers: groupIdentifiers(dto.identifiers ?? []),
    tags: fromTagDTOs(dto.tags),
  }
}

function toMetadata(dto: ClientWorkloadDTO): ClientWorkloadMetadata {
  return {
    entityId: dto.id,
    createdAt: dto.createdAt,
    createdBy: '—',
    lastModifiedAt: dto.updatedAt,
    lastModifiedBy: '—',
    isActive: dto.isActive,
  }
}

function toRequestBody(input: ClientWorkloadInput) {
  return {
    name: input.values.name,
    description: input.values.description,
    isActive: input.isActive,
    identifiers: flattenIdentifiers(input.values.identifiers),
    tags: toTagDTOs(input.values.tags),
  }
}

export interface ClientWorkloadInput {
  values: ClientWorkloadFormValues
  isActive: boolean
}

export interface FetchClientWorkloadsResult {
  items: ClientWorkloadListItem[]
  total: number
}

export interface ClientWorkloadMetadata {
  entityId: string
  createdAt: string
  createdBy: string
  lastModifiedAt: string
  lastModifiedBy: string
  isActive: boolean
}

export interface FetchClientWorkloadDetailResult {
  values: ClientWorkloadFormValues
  metadata: ClientWorkloadMetadata
}

export async function fetchClientWorkloads(
  page = 1,
  perPage = 20,
  search = '',
): Promise<FetchClientWorkloadsResult> {
  const params = new URLSearchParams({ page: String(page), perPage: String(perPage) })
  if (search) params.set('search', search)
  const res = await apiClient.get<ListResponseDTO>(`${BASE}?${params}`)
  return { items: res.items.map(toListItem), total: res.total }
}

export async function fetchClientWorkload(
  id: string,
): Promise<FetchClientWorkloadDetailResult | null> {
  try {
    const dto = await apiClient.get<ClientWorkloadDTO>(`${BASE}/${id}`)
    return { values: toFormValues(dto), metadata: toMetadata(dto) }
  } catch (err) {
    if (err instanceof ApiError && err.status === 404) return null
    throw err
  }
}

export async function createClientWorkload(input: ClientWorkloadInput): Promise<void> {
  await apiClient.post<ClientWorkloadDTO>(BASE, toRequestBody(input))
}

export async function saveClientWorkload(id: string, input: ClientWorkloadInput): Promise<void> {
  await apiClient.put<ClientWorkloadDTO>(`${BASE}/${id}`, toRequestBody(input))
}

export { apiErrorMessage } from '@/shared/lib/api-client'

export async function deleteClientWorkload(id: string): Promise<void> {
  await apiClient.delete(`${BASE}/${id}`)
}

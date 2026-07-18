import { apiClient, ApiError } from '@/shared/lib/api-client'
import { CLIENT_IDENTIFIER_TYPES } from '@/data/policies/config'
import type { ClientWorkloadListItem } from '@/data/client-workloads'
import type { ClientWorkloadFormValues } from '@/shared/components/forms/types'

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

/** Backend stores flat {type,value} pairs; the UI groups values under a type. */
function groupIdentifiers(flat: IdentifierDTO[]): { type: string; values: string[] }[] {
  const grouped: { type: string; values: string[] }[] = []
  const indexByType = new Map<string, number>()
  for (const { type, value } of flat) {
    const existingIndex = indexByType.get(type)
    const entry = existingIndex === undefined ? undefined : grouped[existingIndex]
    if (entry) {
      entry.values.push(value)
    } else {
      indexByType.set(type, grouped.length)
      grouped.push({ type, values: [value] })
    }
  }
  return grouped.length > 0 ? grouped : [{ type: '', values: [''] }]
}

function flattenIdentifiers(groups: { type: string; values: string[] }[]): IdentifierDTO[] {
  const flat: IdentifierDTO[] = []
  for (const group of groups) {
    if (!group.type) continue
    for (const value of group.values) {
      if (value.trim()) flat.push({ type: group.type, value: value.trim() })
    }
  }
  return flat
}

function toListItem(dto: ClientWorkloadDTO): ClientWorkloadListItem {
  const firstIdentifier = dto.identifiers?.[0]
  return {
    id: dto.id,
    name: dto.name,
    type: firstIdentifier ? labelForType(firstIdentifier.type) : '—',
    description: dto.description,
    tags: dto.tags?.length ?? 0,
    policies: 0,
    isActive: dto.isActive,
  }
}

function toFormValues(dto: ClientWorkloadDTO): ClientWorkloadFormValues {
  return {
    name: dto.name,
    description: dto.description,
    identifiers: groupIdentifiers(dto.identifiers ?? []),
    standaloneCaId: '',
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
    tags: [] as TagDTO[],
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

export async function fetchClientWorkloads(): Promise<FetchClientWorkloadsResult> {
  const res = await apiClient.get<ListResponseDTO>(`${BASE}?perPage=100`)
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

export function apiErrorMessage(err: unknown, fallback = 'Something went wrong'): string {
  if (err instanceof ApiError) {
    const data = err.data as { error?: { message?: string } } | undefined
    if (data?.error?.message) return data.error.message
  }
  return fallback
}

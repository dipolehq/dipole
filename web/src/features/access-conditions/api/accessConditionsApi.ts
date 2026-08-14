import { apiClient, ApiError } from '@/shared/lib/api-client'
import { CONDITION_INTEGRATIONS } from '@/shared/constants/credential-providers'

export interface AccessConditionListItem {
  id: string
  name: string
  type: string
  description: string
  tags: number
  policies: number
  isActive: boolean
}
import type {
  AccessConditionFormValues,
  GeoIpCountryEntry,
  TimeConditionDay,
  FormTag,
} from '@/shared/components/forms/types'

const BASE = '/access-conditions'

interface TagDTO {
  key: string
  value: string
}

interface AccessConditionDTO {
  id: string
  name: string
  description: string
  isActive: boolean
  type: string
  config: {
    countries?: GeoIpCountryEntry[]
    timezone?: string
    days?: TimeConditionDay[]
  }
  tags: TagDTO[]
  policyCount: number
  resourceSetId?: string
  createdAt: string
  updatedAt: string
}

interface ListResponseDTO {
  items: AccessConditionDTO[]
  total: number
  page: number
  perPage: number
}

const TYPE_LABELS: Record<string, string> = Object.fromEntries(
  CONDITION_INTEGRATIONS.map((c) => [c.value, c.label]),
)

const DEFAULT_DAYS: TimeConditionDay[] = [
  { day: 'Monday', startTime: '09:00', endTime: '18:00', enabled: false },
  { day: 'Tuesday', startTime: '09:00', endTime: '18:00', enabled: false },
  { day: 'Wednesday', startTime: '09:00', endTime: '18:00', enabled: false },
  { day: 'Thursday', startTime: '09:00', endTime: '18:00', enabled: false },
  { day: 'Friday', startTime: '09:00', endTime: '18:00', enabled: false },
  { day: 'Saturday', startTime: '09:00', endTime: '18:00', enabled: false },
  { day: 'Sunday', startTime: '09:00', endTime: '18:00', enabled: false },
]

function toListItem(dto: AccessConditionDTO): AccessConditionListItem {
  return {
    id: dto.id,
    name: dto.name,
    type: TYPE_LABELS[dto.type] ?? dto.type,
    description: dto.description,
    tags: dto.tags?.length ?? 0,
    policies: dto.policyCount ?? 0,
    isActive: dto.isActive,
  }
}

function toFormValues(dto: AccessConditionDTO): AccessConditionFormValues {
  const cfg = dto.config ?? {}
  return {
    name: dto.name,
    description: dto.description,
    integration: dto.type,
    countries: cfg.countries?.length ? cfg.countries : [{ country: '', subdivisions: [''] }],
    timezone: cfg.timezone ?? '',
    days: cfg.days?.length ? cfg.days : DEFAULT_DAYS,
    tags: fromTagDTOs(dto.tags),
  }
}

function toMetadata(dto: AccessConditionDTO): AccessConditionMetadata {
  return {
    entityId: dto.id,
    createdAt: dto.createdAt,
    createdBy: '—',
    lastModifiedAt: dto.updatedAt,
    lastModifiedBy: '—',
    isActive: dto.isActive,
  }
}

function buildConfig(v: AccessConditionFormValues): Record<string, unknown> {
  if (v.integration === 'geoip') {
    const countries = v.countries.filter((c) => c.country)
    return { countries }
  }
  if (v.integration === 'time') {
    return { timezone: v.timezone, days: v.days }
  }
  return {}
}

function toRequestBody(input: AccessConditionInput) {
  const v = input.values
  return {
    name: v.name,
    description: v.description,
    isActive: input.isActive,
    type: v.integration,
    config: buildConfig(v),
    tags: toTagDTOs(v.tags),
  }
}

function toTagDTOs(tags: FormTag[]): TagDTO[] {
  return tags.filter((t) => t.key.trim()).map((t) => ({ key: t.key.trim(), value: t.value }))
}

function fromTagDTOs(tags: TagDTO[] | undefined): FormTag[] {
  return (tags ?? []).map((t) => ({ key: t.key, value: t.value }))
}

export interface AccessConditionInput {
  values: AccessConditionFormValues
  isActive: boolean
}

export interface FetchAccessConditionsResult {
  items: AccessConditionListItem[]
  total: number
}

export interface AccessConditionMetadata {
  entityId: string
  createdAt: string
  createdBy: string
  lastModifiedAt: string
  lastModifiedBy: string
  isActive: boolean
}

export interface FetchAccessConditionDetailResult {
  values: AccessConditionFormValues
  metadata: AccessConditionMetadata
}

export async function fetchAccessConditions(
  page = 1,
  perPage = 20,
  search = '',
): Promise<FetchAccessConditionsResult> {
  const params = new URLSearchParams({ page: String(page), perPage: String(perPage) })
  if (search) params.set('search', search)
  const res = await apiClient.get<ListResponseDTO>(`${BASE}?${params}`)
  return { items: res.items.map(toListItem), total: res.total }
}

export async function fetchAccessCondition(
  id: string,
): Promise<FetchAccessConditionDetailResult | null> {
  try {
    const dto = await apiClient.get<AccessConditionDTO>(`${BASE}/${id}`)
    return { values: toFormValues(dto), metadata: toMetadata(dto) }
  } catch (err) {
    if (err instanceof ApiError && err.status === 404) return null
    throw err
  }
}

export async function createAccessCondition(input: AccessConditionInput): Promise<void> {
  await apiClient.post<AccessConditionDTO>(BASE, toRequestBody(input))
}

export async function saveAccessCondition(id: string, input: AccessConditionInput): Promise<void> {
  await apiClient.put<AccessConditionDTO>(`${BASE}/${id}`, toRequestBody(input))
}

export { apiErrorMessage } from '@/shared/lib/api-client'

export async function deleteAccessCondition(id: string): Promise<void> {
  await apiClient.delete(`${BASE}/${id}`)
}

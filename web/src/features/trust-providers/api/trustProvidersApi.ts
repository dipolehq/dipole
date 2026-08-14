import { apiClient, ApiError } from '@/shared/lib/api-client'
import { TRUST_PROVIDER_TYPES } from '@/shared/constants/policies'

export interface TrustProviderListItem {
  id: string
  name: string
  type: string
  description: string
  tags: number
  policies: number
  isActive: boolean
}
import type {
  TrustProviderFormValues,
  TrustProviderMatchRule,
  FormTag,
} from '@/shared/components/forms/types'

const BASE = '/trust-providers'

interface TagDTO {
  key: string
  value: string
}

interface MatchRuleDTO {
  attribute: string
  value: string
}

interface TrustProviderDTO {
  id: string
  name: string
  description: string
  isActive: boolean
  type: string
  config: Record<string, unknown>
  matchRules: MatchRuleDTO[]
  edgeSdkClientId: string
  edgeSdkAudience: string
  tags: TagDTO[]
  policyCount: number
  resourceSetId?: string
  createdAt: string
  updatedAt: string
}

interface ListResponseDTO {
  items: TrustProviderDTO[]
  total: number
  page: number
  perPage: number
}

const TYPE_LABELS: Record<string, string> = Object.fromEntries(
  TRUST_PROVIDER_TYPES.map((t) => [t.value, t.label]),
)

function toListItem(dto: TrustProviderDTO): TrustProviderListItem {
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

function toFormValues(dto: TrustProviderDTO): TrustProviderFormValues {
  const cfg = dto.config ?? {}
  const matchRules: TrustProviderMatchRule[] =
    dto.matchRules?.length > 0
      ? dto.matchRules.map((mr) => ({ attribute: mr.attribute, value: mr.value }))
      : [{ attribute: '', value: '' }]

  return {
    name: dto.name,
    description: dto.description,
    trustProviderType: dto.type,
    matchRules,
    certificate: (cfg.certificate as string) ?? '',
    thumbprint: (cfg.thumbprint as string) ?? '',
    subject: (cfg.subject as string) ?? '',
    expiresAt: (cfg.expiresAt as string) ?? '',
    oidcEndpoint: (cfg.oidcEndpoint as string) ?? '',
    attestationMethod: (cfg.attestationMethod as string) ?? 'oidc_discovery',
    samlMetadataUrl: (cfg.samlMetadataUrl as string) ?? '',
    samlMetadataXml: (cfg.samlMetadataXml as string) ?? '',
    symmetricKey: (cfg.symmetricKey as string) ?? '',
    jwks: (cfg.jwks as string) ?? '',
    publicKey: (cfg.publicKey as string) ?? '',
    tags: fromTagDTOs(dto.tags),
  }
}

function toMetadata(dto: TrustProviderDTO): TrustProviderMetadata {
  return {
    entityId: dto.id,
    createdAt: dto.createdAt,
    createdBy: '—',
    lastModifiedAt: dto.updatedAt,
    lastModifiedBy: '—',
    isActive: dto.isActive,
    edgeSdkClientId: dto.edgeSdkClientId,
    edgeSdkAudience: dto.edgeSdkAudience,
  }
}

function buildConfig(v: TrustProviderFormValues): Record<string, unknown> {
  const cfg: Record<string, unknown> = {}

  if (v.certificate) cfg.certificate = v.certificate
  if (v.thumbprint) cfg.thumbprint = v.thumbprint
  if (v.subject) cfg.subject = v.subject
  if (v.expiresAt) cfg.expiresAt = v.expiresAt
  if (v.oidcEndpoint) cfg.oidcEndpoint = v.oidcEndpoint
  if (v.attestationMethod) cfg.attestationMethod = v.attestationMethod
  if (v.samlMetadataUrl) cfg.samlMetadataUrl = v.samlMetadataUrl
  if (v.samlMetadataXml) cfg.samlMetadataXml = v.samlMetadataXml
  if (v.symmetricKey) cfg.symmetricKey = v.symmetricKey
  if (v.jwks) cfg.jwks = v.jwks
  if (v.publicKey) cfg.publicKey = v.publicKey

  return cfg
}

function toRequestBody(input: TrustProviderInput) {
  const v = input.values
  const matchRules: MatchRuleDTO[] = v.matchRules
    .filter((r) => r.attribute.trim())
    .map((r) => ({ attribute: r.attribute.trim(), value: r.value }))

  return {
    name: v.name,
    description: v.description,
    isActive: input.isActive,
    type: v.trustProviderType,
    config: buildConfig(v),
    matchRules,
    tags: toTagDTOs(v.tags),
  }
}

function toTagDTOs(tags: FormTag[]): TagDTO[] {
  return tags.filter((t) => t.key.trim()).map((t) => ({ key: t.key.trim(), value: t.value }))
}

function fromTagDTOs(tags: TagDTO[] | undefined): FormTag[] {
  return (tags ?? []).map((t) => ({ key: t.key, value: t.value }))
}

export interface TrustProviderInput {
  values: TrustProviderFormValues
  isActive: boolean
}

export interface FetchTrustProvidersResult {
  items: TrustProviderListItem[]
  total: number
}

export interface TrustProviderMetadata {
  entityId: string
  createdAt: string
  createdBy: string
  lastModifiedAt: string
  lastModifiedBy: string
  isActive: boolean
  edgeSdkClientId: string
  edgeSdkAudience: string
}

export interface FetchTrustProviderDetailResult {
  values: TrustProviderFormValues
  metadata: TrustProviderMetadata
}

export async function fetchTrustProviders(
  page = 1,
  perPage = 20,
  search = '',
): Promise<FetchTrustProvidersResult> {
  const params = new URLSearchParams({ page: String(page), perPage: String(perPage) })
  if (search) params.set('search', search)
  const res = await apiClient.get<ListResponseDTO>(`${BASE}?${params}`)
  return { items: res.items.map(toListItem), total: res.total }
}

export async function fetchTrustProvider(
  id: string,
): Promise<FetchTrustProviderDetailResult | null> {
  try {
    const dto = await apiClient.get<TrustProviderDTO>(`${BASE}/${id}`)
    return { values: toFormValues(dto), metadata: toMetadata(dto) }
  } catch (err) {
    if (err instanceof ApiError && err.status === 404) return null
    throw err
  }
}

export async function createTrustProvider(input: TrustProviderInput): Promise<void> {
  await apiClient.post<TrustProviderDTO>(BASE, toRequestBody(input))
}

export async function saveTrustProvider(id: string, input: TrustProviderInput): Promise<void> {
  await apiClient.put<TrustProviderDTO>(`${BASE}/${id}`, toRequestBody(input))
}

export { apiErrorMessage } from '@/shared/lib/api-client'

export async function deleteTrustProvider(id: string): Promise<void> {
  await apiClient.delete(`${BASE}/${id}`)
}

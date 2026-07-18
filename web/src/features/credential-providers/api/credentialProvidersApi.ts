import { apiClient, ApiError } from '@/shared/lib/api-client'
import { CREDENTIAL_TYPES } from '@/data/policies/geoData'
import type { CredentialProviderListItem } from '@/data/credential-providers'
import type { CredentialProviderFormValues, FormTag } from '@/shared/components/forms/types'
import { EMPTY_CREDENTIAL_PROVIDER } from '@/shared/components/forms/types'

const BASE = '/credential-providers'

interface TagDTO {
  key: string
  value: string
}

interface CredentialProviderListDTO {
  id: string
  name: string
  description: string
  isActive: boolean
  type: string
  tags: TagDTO[]
  policyCount: number
  createdAt: string
  updatedAt: string
}

interface CredentialProviderDetailDTO {
  id: string
  name: string
  description: string
  isActive: boolean
  type: string
  config: Record<string, unknown>
  tags: TagDTO[]
  integrationId?: string
  policyCount: number
  createdAt: string
  updatedAt: string
}

interface ListResponseDTO {
  items: CredentialProviderListDTO[]
  total: number
  page: number
  perPage: number
}

const TYPE_LABELS: Record<string, string> = Object.fromEntries(
  CREDENTIAL_TYPES.map((t) => [t.value, t.label]),
)

function toListItem(dto: CredentialProviderListDTO): CredentialProviderListItem {
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

function toFormValues(dto: CredentialProviderDetailDTO): CredentialProviderFormValues {
  const cfg = dto.config ?? {}
  return {
    ...EMPTY_CREDENTIAL_PROVIDER,
    name: dto.name,
    description: dto.description,
    credentialType: dto.type,
    tags: fromTagDTOs(dto.tags),
    // Spread all config fields — they map 1:1 by key name
    apiKey: (cfg.apiKey as string) ?? '',
    username: (cfg.username as string) ?? '',
    password: (cfg.password as string) ?? '',
    tokenEndpointUrl: (cfg.tokenEndpointUrl as string) ?? '',
    authorizationUrl: (cfg.authorizationUrl as string) ?? '',
    introspectionUrl: (cfg.introspectionUrl as string) ?? '',
    clientId: (cfg.clientId as string) ?? '',
    clientSecret: (cfg.clientSecret as string) ?? '',
    scopes: (cfg.scopes as string) ?? '',
    credentialStyle: (cfg.credentialStyle as string) ?? 'authorization_header',
    mcpServerUrl: (cfg.mcpServerUrl as string) ?? '',
    mcpResource: (cfg.mcpResource as string) ?? '',
    mcpPkceRequired: (cfg.mcpPkceRequired as boolean) ?? false,
    oauthPkceRequired: (cfg.oauthPkceRequired as boolean) ?? false,
    oauthUrl: (cfg.oauthUrl as string) ?? '',
    oauthLifetimeValue: (cfg.oauthLifetimeValue as number) ?? 1,
    oauthLifetimeUnit: (cfg.oauthLifetimeUnit as string) ?? 'years',
    awsIamRoleArn: (cfg.awsIamRoleArn as string) ?? '',
    integrationId: (cfg.integrationId as string) ?? dto.integrationId ?? '',
    secretArn: (cfg.secretArn as string) ?? '',
    credentialValueType: (cfg.credentialValueType as string) ?? 'plain_text',
    secretKey: (cfg.secretKey as string) ?? '',
    usernameKey: (cfg.usernameKey as string) ?? '',
    passwordKey: (cfg.passwordKey as string) ?? '',
    privateNetworkAccess: (cfg.privateNetworkAccess as boolean) ?? false,
    azureTenantId: (cfg.azureTenantId as string) ?? '',
    azureClientId: (cfg.azureClientId as string) ?? '',
    jwtTokenSubject: (cfg.jwtTokenSubject as string) ?? '',
    jwtTokenScope: (cfg.jwtTokenScope as string) ?? '',
    keyVaultUrl: (cfg.keyVaultUrl as string) ?? '',
    keyVaultSecretName: (cfg.keyVaultSecretName as string) ?? '',
    gcpServiceAccountEmail: (cfg.gcpServiceAccountEmail as string) ?? '',
    gcpProjectNumber: (cfg.gcpProjectNumber as string) ?? '',
    gcpWorkloadIdentityPool: (cfg.gcpWorkloadIdentityPool as string) ?? '',
    gcpWifProvider: (cfg.gcpWifProvider as string) ?? '',
    claudeApiAudience: (cfg.claudeApiAudience as string) ?? '',
    claudeFederationRuleId: (cfg.claudeFederationRuleId as string) ?? '',
    claudeOrganizationId: (cfg.claudeOrganizationId as string) ?? '',
    claudeScope: (cfg.claudeScope as string) ?? 'workspace:developer',
    claudeServiceAccountId: (cfg.claudeServiceAccountId as string) ?? '',
    claudeWorkspaceId: (cfg.claudeWorkspaceId as string) ?? '',
    openaiIdentityProviderId: (cfg.openaiIdentityProviderId as string) ?? '',
    openaiServiceAccountId: (cfg.openaiServiceAccountId as string) ?? '',
    snowflakeAccountId: (cfg.snowflakeAccountId as string) ?? '',
    snowflakeUsername: (cfg.snowflakeUsername as string) ?? '',
    spiffeTrustDomain: (cfg.spiffeTrustDomain as string) ?? '',
    signingAlgorithm: (cfg.signingAlgorithm as string) ?? 'ES256',
    subject: (cfg.subject as string) ?? '',
    customClaims: (cfg.customClaims as { name: string; value: string; valueType: string }[]) ?? [],
    vaultHost: (cfg.vaultHost as string) ?? '',
    vaultPort: (cfg.vaultPort as string) ?? '8200',
    vaultTls: (cfg.vaultTls as boolean) ?? true,
    vaultAuthPath: (cfg.vaultAuthPath as string) ?? '',
    vaultRole: (cfg.vaultRole as string) ?? '',
    vaultNamespace: (cfg.vaultNamespace as string) ?? '',
    vaultForwardingConfig: (cfg.vaultForwardingConfig as string) ?? 'none',
    gitlabGroupId: (cfg.gitlabGroupId as string) ?? '',
    gitlabProjectId: (cfg.gitlabProjectId as string) ?? '',
    gitlabScopes: (cfg.gitlabScopes as string) ?? '',
    gitlabAccessLevel: (cfg.gitlabAccessLevel as string) ?? '',
    gitlabServiceAccountUsername: (cfg.gitlabServiceAccountUsername as string) ?? '',
    gitlabRotationTime: (cfg.gitlabRotationTime as number) ?? 1,
    gitlabRotationUnit: (cfg.gitlabRotationUnit as string) ?? 'days',
    standaloneCaId: (cfg.standaloneCaId as string) ?? '',
    spiffeId: (cfg.spiffeId as string) ?? '',
    keyUsage: (cfg.keyUsage as string) ?? 'digital_signature',
    extendedKeyUsage: (cfg.extendedKeyUsage as string[]) ?? ['id-kp-clientAuth'],
    audience: (cfg.audience as string) ?? '',
    lifetimeSeconds: (cfg.lifetimeSeconds as number) ?? 3600,
  }
}

function buildConfig(v: CredentialProviderFormValues): Record<string, unknown> {
  const cfg: Record<string, unknown> = {}
  const exclude = new Set(['name', 'description', 'credentialType', 'tags'])

  for (const [key, val] of Object.entries(v)) {
    if (exclude.has(key)) continue
    if (typeof val === 'boolean') {
      cfg[key] = val
      continue
    }
    if (val === '' || val === 0 || (Array.isArray(val) && val.length === 0)) continue
    cfg[key] = val
  }
  return cfg
}

function toMetadata(dto: CredentialProviderDetailDTO): CredentialProviderMetadata {
  return {
    entityId: dto.id,
    createdAt: dto.createdAt,
    createdBy: '—',
    lastModifiedAt: dto.updatedAt,
    lastModifiedBy: '—',
    isActive: dto.isActive,
  }
}

function toTagDTOs(tags: FormTag[]): TagDTO[] {
  return tags.filter((t) => t.key.trim()).map((t) => ({ key: t.key.trim(), value: t.value }))
}

function fromTagDTOs(tags: TagDTO[] | undefined): FormTag[] {
  return (tags ?? []).map((t) => ({ key: t.key, value: t.value }))
}

function toRequestBody(input: CredentialProviderInput) {
  const v = input.values
  return {
    name: v.name,
    description: v.description,
    isActive: input.isActive,
    type: v.credentialType,
    config: buildConfig(v),
    tags: toTagDTOs(v.tags),
  }
}

export interface CredentialProviderInput {
  values: CredentialProviderFormValues
  isActive: boolean
}

export interface FetchCredentialProvidersResult {
  items: CredentialProviderListItem[]
  total: number
}

export interface CredentialProviderMetadata {
  entityId: string
  createdAt: string
  createdBy: string
  lastModifiedAt: string
  lastModifiedBy: string
  isActive: boolean
}

export interface FetchCredentialProviderDetailResult {
  values: CredentialProviderFormValues
  metadata: CredentialProviderMetadata
}

export async function fetchCredentialProviders(
  page = 1,
  perPage = 20,
  search = '',
): Promise<FetchCredentialProvidersResult> {
  const params = new URLSearchParams({ page: String(page), perPage: String(perPage) })
  if (search) params.set('search', search)
  const res = await apiClient.get<ListResponseDTO>(`${BASE}?${params}`)
  return { items: res.items.map(toListItem), total: res.total }
}

export async function fetchCredentialProvider(
  id: string,
): Promise<FetchCredentialProviderDetailResult | null> {
  try {
    const dto = await apiClient.get<CredentialProviderDetailDTO>(`${BASE}/${id}`)
    return { values: toFormValues(dto), metadata: toMetadata(dto) }
  } catch (err) {
    if (err instanceof ApiError && err.status === 404) return null
    throw err
  }
}

export async function createCredentialProvider(input: CredentialProviderInput): Promise<void> {
  await apiClient.post<CredentialProviderDetailDTO>(BASE, toRequestBody(input))
}

export async function saveCredentialProvider(
  id: string,
  input: CredentialProviderInput,
): Promise<void> {
  await apiClient.put<CredentialProviderDetailDTO>(`${BASE}/${id}`, toRequestBody(input))
}

export { apiErrorMessage } from '@/shared/lib/api-client'

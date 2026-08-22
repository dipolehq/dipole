import { apiClient, ApiError, apiErrorMessage } from '@/shared/lib/api-client'
import { groupIdentifiers, flattenIdentifiers } from '@/shared/lib/identifiers'
import type { PolicyFormValues, PolicyMetadata } from '@/features/policies/types/formValues'

const BASE = '/access-policies'

interface PolicyDTO {
  id: string
  name: string
  description: string
  isActive: boolean
  clientWorkloadId: string
  serverWorkloadId: string
  trustProviderIds: string[]
  credentialProviderIds: {
    credentialProviderId: string
    mappingType: string
    mappingValue: string
  }[]
  accessConditionIds: string[]
  tags: { key: string; value: string }[]
  createdAt: string
  updatedAt: string
}

interface ListResponseDTO {
  items: PolicyDTO[]
  total: number
  page: number
  perPage: number
}

export interface PolicyListItem {
  id: string
  name: string
  isActive: boolean
  clientWorkloadId: string
  serverWorkloadId: string
  trustProviderIds: string[]
  accessConditionIds: string[]
  credentialProviderIds: string[]
  createdAt: string
}

function toListItem(dto: PolicyDTO): PolicyListItem {
  return {
    id: dto.id,
    name: dto.name,
    isActive: dto.isActive,
    clientWorkloadId: dto.clientWorkloadId,
    serverWorkloadId: dto.serverWorkloadId,
    trustProviderIds: dto.trustProviderIds ?? [],
    accessConditionIds: dto.accessConditionIds ?? [],
    credentialProviderIds: (dto.credentialProviderIds ?? []).map((c) => c.credentialProviderId),
    createdAt: dto.createdAt,
  }
}

export interface FetchPoliciesResult {
  items: PolicyListItem[]
  total: number
}

export interface FetchPolicyResult {
  values: PolicyFormValues
  metadata: PolicyMetadata
}

function toFormValues(dto: PolicyDTO): PolicyFormValues {
  return {
    name: dto.name,
    description: dto.description,
    status: dto.isActive ? 'active' : 'inactive',
    tags: (dto.tags ?? []).map((t) => ({ key: t.key, value: t.value })),
    clientWorkloadId: dto.clientWorkloadId ?? '',
    client: { name: '', description: '', identifiers: [{ type: '', values: [''] }], tags: [] },
    serverWorkloadId: dto.serverWorkloadId ?? '',
    server: {
      name: '',
      description: '',
      host: '',
      appProtocol: '',
      transportProtocol: 'TCP',
      port: '',
      forwardToPort: '',
      tls: false,
      forwardTls: false,
      forwardTlsVerification: 'full',
      authMethod: '',
      authScheme: '',
      headerName: '',
      urlPath: '',
      staticHeaders: [],
      tags: [],
    },
    trustProviderIds: dto.trustProviderIds ?? [],
    trustProviders: [],
    accessConditionIds: dto.accessConditionIds ?? [],
    accessConditions: [],
    credentialProviderId: (dto.credentialProviderIds ?? [])[0]?.credentialProviderId ?? '',
    credential: {
      name: '',
      description: '',
      credentialType: '',
      tags: [],
      apiKey: '',
      username: '',
      password: '',
      tokenEndpointUrl: '',
      authorizationUrl: '',
      introspectionUrl: '',
      clientId: '',
      clientSecret: '',
      scopes: '',
      credentialStyle: 'authorization_header',
      mcpServerUrl: '',
      mcpResource: '',
      mcpPkceRequired: false,
      oauthPkceRequired: false,
      oauthUrl: '',
      oauthLifetimeValue: 1,
      oauthLifetimeUnit: 'years',
      awsIamRoleArn: '',
      integrationId: '',
      secretArn: '',
      credentialValueType: 'plain_text',
      secretKey: '',
      usernameKey: '',
      passwordKey: '',
      privateNetworkAccess: false,
      azureTenantId: '',
      azureClientId: '',
      jwtTokenSubject: '',
      jwtTokenScope: '',
      keyVaultUrl: '',
      keyVaultSecretName: '',
      gcpServiceAccountEmail: '',
      gcpProjectNumber: '',
      gcpWorkloadIdentityPool: '',
      gcpWifProvider: '',
      claudeApiAudience: '',
      claudeFederationRuleId: '',
      claudeOrganizationId: '',
      claudeScope: 'workspace:developer',
      claudeServiceAccountId: '',
      claudeWorkspaceId: '',
      openaiIdentityProviderId: '',
      openaiServiceAccountId: '',
      snowflakeAccountId: '',
      snowflakeUsername: '',
      spiffeTrustDomain: '',
      signingAlgorithm: 'ES256',
      subject: '',
      subjectValueType: 'literal',
      audienceValueType: 'literal',
      customClaims: [],
      vaultHost: '',
      vaultPort: '8200',
      vaultTls: true,
      vaultAuthPath: '',
      vaultRole: '',
      vaultNamespace: '',
      vaultForwardingConfig: 'none',
      gitlabGroupId: '',
      gitlabProjectId: '',
      gitlabScopes: '',
      gitlabAccessLevel: '',
      gitlabServiceAccountUsername: '',
      gitlabRotationTime: 1,
      gitlabRotationUnit: 'days',
      standaloneCaId: '',
      spiffeId: '',
      keyUsage: 'digital_signature',
      extendedKeyUsage: ['id-kp-clientAuth'],
      audience: '',
      lifetimeSeconds: 3600,
    },
  }
}

function toRequestBody(values: PolicyFormValues) {
  const cpIds = values.credentialProviderId
    ? [{ credentialProviderId: values.credentialProviderId, mappingType: 'None', mappingValue: '' }]
    : []

  return {
    name: values.name,
    description: values.description,
    isActive: values.status !== 'inactive',
    clientWorkloadId: values.clientWorkloadId,
    serverWorkloadId: values.serverWorkloadId,
    trustProviderIds: values.trustProviderIds,
    credentialProviderIds: cpIds,
    accessConditionIds: values.accessConditionIds,
    tags: values.tags.map((t) => ({ key: t.key, value: t.value })),
  }
}

export async function fetchPolicies(
  page = 1,
  perPage = 20,
  search = '',
): Promise<FetchPoliciesResult> {
  const params = new URLSearchParams({ page: String(page), perPage: String(perPage) })
  if (search) params.set('search', search)
  const res = await apiClient.get<ListResponseDTO>(`${BASE}?${params}`)
  return { items: res.items.map(toListItem), total: res.total }
}

export async function fetchPolicy(id: string): Promise<FetchPolicyResult | null> {
  try {
    const dto = await apiClient.get<PolicyDTO>(`${BASE}/${id}`)

    // Hydrate sub-entities in parallel
    const [cwData, swData, tpData, acData, cpData] = await Promise.all([
      dto.clientWorkloadId
        ? apiClient.get<{ id: string; name: string; description: string; identifiers: { type: string; value: string }[]; tags: { key: string; value: string }[] }>(`/client-workloads/${dto.clientWorkloadId}`).catch(() => null)
        : Promise.resolve(null),
      dto.serverWorkloadId
        ? apiClient.get<{ id: string; name: string; description: string; host: string; port: number; appProtocol: string; transportProtocol: string; tls: boolean; tlsVerification: string; requestedPort: number; requestedTls: boolean; authMethod: string; authScheme: string; authConfig: Record<string, string>; httpHeaders: Record<string, string>; urlPath: string; tags: { key: string; value: string }[] }>(`/server-workloads/${dto.serverWorkloadId}`).catch(() => null)
        : Promise.resolve(null),
      Promise.all(
        (dto.trustProviderIds ?? []).map((tpId) =>
          apiClient.get<{ id: string; name: string; description: string; type: string; config: Record<string, unknown>; matchRules: { attribute: string; value: string }[]; tags: { key: string; value: string }[] }>(`/trust-providers/${tpId}`).catch(() => null),
        ),
      ),
      Promise.all(
        (dto.accessConditionIds ?? []).map((acId) =>
          apiClient.get<{ id: string; name: string; description: string; type: string; config: Record<string, unknown>; tags: { key: string; value: string }[] }>(`/access-conditions/${acId}`).catch(() => null),
        ),
      ),
      (dto.credentialProviderIds ?? [])[0]?.credentialProviderId
        ? apiClient.get<{ id: string; name: string; description: string; type: string; config: Record<string, unknown>; tags: { key: string; value: string }[] }>(`/credential-providers/${(dto.credentialProviderIds ?? [])[0]!.credentialProviderId}`).catch(() => null)
        : Promise.resolve(null),
    ])

    const client = cwData
      ? {
          name: cwData.name,
          description: cwData.description,
          identifiers: groupIdentifiers(cwData.identifiers ?? []),
          tags: (cwData.tags ?? []).map((t) => ({ key: t.key, value: t.value })),
        }
      : { name: '', description: '', identifiers: [{ type: '', values: [''] }], tags: [] }

    const server = swData
      ? {
          name: swData.name,
          description: swData.description,
          host: swData.host,
          appProtocol: swData.appProtocol,
          transportProtocol: swData.transportProtocol || 'TCP',
          port: String(swData.port || ''),
          forwardToPort: String(swData.requestedPort || ''),
          tls: swData.tls,
          forwardTls: swData.requestedTls,
          forwardTlsVerification: swData.tlsVerification || 'full',
          authMethod: swData.authMethod,
          authScheme: swData.authScheme,
          headerName: swData.authConfig?.header_name ?? '',
          urlPath: swData.urlPath,
          staticHeaders: Object.entries(swData.httpHeaders ?? {}).map(([name, value]) => ({ name, value })),
          tags: (swData.tags ?? []).map((t) => ({ key: t.key, value: t.value })),
        }
      : { name: '', description: '', host: '', appProtocol: '', transportProtocol: 'TCP', port: '', forwardToPort: '', tls: false, forwardTls: false, forwardTlsVerification: 'full', authMethod: '', authScheme: '', headerName: '', urlPath: '', staticHeaders: [], tags: [] }

    const trustProviders = tpData
      .filter((tp): tp is NonNullable<typeof tp> => tp !== null)
      .map((tp) => ({
        existingId: tp.id,
        name: tp.name,
        description: tp.description,
        trustProviderType: tp.type,
        matchRules: (tp.matchRules ?? []).length > 0 ? tp.matchRules : [{ attribute: '', value: '' }],
        certificate: (tp.config?.certificate as string) ?? '',
        thumbprint: (tp.config?.thumbprint as string) ?? '',
        subject: (tp.config?.subject as string) ?? '',
        expiresAt: (tp.config?.expiresAt as string) ?? '',
        oidcEndpoint: (tp.config?.oidcEndpoint as string) ?? '',
        attestationMethod: (tp.config?.attestationMethod as string) ?? 'oidc_discovery',
        samlMetadataUrl: (tp.config?.samlMetadataUrl as string) ?? '',
        samlMetadataXml: (tp.config?.samlMetadataXml as string) ?? '',
        symmetricKey: (tp.config?.symmetricKey as string) ?? '',
        jwks: (tp.config?.jwks as string) ?? '',
        publicKey: (tp.config?.publicKey as string) ?? '',
        tags: (tp.tags ?? []).map((t) => ({ key: t.key, value: t.value })),
      }))

    const accessConditions = acData
      .filter((ac): ac is NonNullable<typeof ac> => ac !== null)
      .map((ac) => ({
        // Marks this entry as already existing so savePolicy/createPolicy
        // never re-POST it as a new Access Condition record.
        existingId: ac.id,
        name: ac.name,
        description: ac.description,
        integration: ac.type,
        timezone: (ac.config?.timezone as string) ?? '',
        days: (ac.config?.days as { day: string; startTime: string; endTime: string; enabled: boolean }[]) ?? [],
        countries: (ac.config?.countries as { country: string; subdivisions: string[] }[]) ?? [],
        tags: (ac.tags ?? []).map((t) => ({ key: t.key, value: t.value })),
      }))

    const credential = cpData
      ? {
          name: cpData.name,
          description: cpData.description,
          credentialType: cpData.type,
          tags: (cpData.tags ?? []).map((t) => ({ key: t.key, value: t.value })),
          apiKey: (cpData.config as Record<string, string>)?.apiKey ?? '',
          username: (cpData.config as Record<string, string>)?.username ?? '',
          password: (cpData.config as Record<string, string>)?.password ?? '',
          tokenEndpointUrl: (cpData.config as Record<string, string>)?.tokenEndpointUrl ?? '',
          authorizationUrl: (cpData.config as Record<string, string>)?.authorizationUrl ?? '',
          introspectionUrl: (cpData.config as Record<string, string>)?.introspectionUrl ?? '',
          clientId: (cpData.config as Record<string, string>)?.clientId ?? '',
          clientSecret: (cpData.config as Record<string, string>)?.clientSecret ?? '',
          scopes: (cpData.config as Record<string, string>)?.scopes ?? '',
          credentialStyle: (cpData.config as Record<string, string>)?.credentialStyle ?? 'authorization_header',
          mcpServerUrl: '', mcpResource: '', mcpPkceRequired: false, oauthPkceRequired: false,
          oauthUrl: '', oauthLifetimeValue: 1, oauthLifetimeUnit: 'years',
          awsIamRoleArn: (cpData.config as Record<string, string>)?.awsIamRoleArn ?? '',
          integrationId: '', secretArn: '', credentialValueType: 'plain_text',
          secretKey: '', usernameKey: '', passwordKey: '', privateNetworkAccess: false,
          azureTenantId: '', azureClientId: '', jwtTokenSubject: '', jwtTokenScope: '',
          keyVaultUrl: '', keyVaultSecretName: '',
          gcpServiceAccountEmail: '', gcpProjectNumber: '', gcpWorkloadIdentityPool: '', gcpWifProvider: '',
          claudeApiAudience: '', claudeFederationRuleId: '', claudeOrganizationId: '',
          claudeScope: 'workspace:developer', claudeServiceAccountId: '', claudeWorkspaceId: '',
          openaiIdentityProviderId: '', openaiServiceAccountId: '',
          snowflakeAccountId: '', snowflakeUsername: '', spiffeTrustDomain: '',
          signingAlgorithm: (cpData.config as Record<string, string>)?.signingAlgorithm ?? 'ES256',
          subject: (cpData.config as Record<string, string>)?.subject ?? '',
          subjectValueType: (cpData.config as Record<string, string>)?.subjectValueType ?? 'literal',
          audienceValueType: (cpData.config as Record<string, string>)?.audienceValueType ?? 'literal',
          customClaims: [],
          vaultHost: '', vaultPort: '8200', vaultTls: true, vaultAuthPath: '', vaultRole: '', vaultNamespace: '', vaultForwardingConfig: 'none',
          gitlabGroupId: '', gitlabProjectId: '', gitlabScopes: '', gitlabAccessLevel: '',
          gitlabServiceAccountUsername: '', gitlabRotationTime: 1, gitlabRotationUnit: 'days',
          standaloneCaId: '', spiffeId: '', keyUsage: 'digital_signature', extendedKeyUsage: ['id-kp-clientAuth'],
          audience: (cpData.config as Record<string, string>)?.audience ?? '',
          lifetimeSeconds: Number((cpData.config as Record<string, string>)?.lifetimeSeconds) || 3600,
        }
      : toFormValues({ ...dto, credentialProviderIds: [] }).credential

    return {
      values: {
        name: dto.name,
        description: dto.description,
        status: dto.isActive ? 'active' : 'inactive',
        tags: (dto.tags ?? []).map((t) => ({ key: t.key, value: t.value })),
        clientWorkloadId: dto.clientWorkloadId ?? '',
        client,
        serverWorkloadId: dto.serverWorkloadId ?? '',
        server,
        trustProviderIds: dto.trustProviderIds ?? [],
        trustProviders,
        accessConditionIds: dto.accessConditionIds ?? [],
        accessConditions,
        credentialProviderId: (dto.credentialProviderIds ?? [])[0]?.credentialProviderId ?? '',
        credential,
      },
      metadata: {
        policyId: dto.id,
        createdAt: dto.createdAt,
        createdBy: '—',
        lastModifiedAt: dto.updatedAt,
        lastModifiedBy: '—',
      },
    }
  } catch (err) {
    if (err instanceof ApiError && err.status === 404) return null
    throw err
  }
}

export async function createPolicy(values: PolicyFormValues): Promise<void> {
  // Create sub-entities inline if IDs are not already set (user chose "Add New").
  // Each step is wrapped to provide clear error context on failure.
  // Note: already-created entities remain valid standalone resources if a later step fails.

  let clientWorkloadId = values.clientWorkloadId
  if (!clientWorkloadId && values.client.name) {
    try {
      const cw = await apiClient.post<{ id: string }>('/client-workloads', {
        name: values.client.name,
        description: values.client.description,
        isActive: true,
        identifiers: flattenIdentifiers(values.client.identifiers),
        tags: (values.client.tags ?? []).map((t) => ({ key: t.key, value: t.value })),
      })
      clientWorkloadId = cw.id
    } catch (err) {
      throw new Error(`Failed to create Client Workload "${values.client.name}": ${apiErrorMessage(err)}`)
    }
  }

  let serverWorkloadId = values.serverWorkloadId
  if (!serverWorkloadId && values.server.name) {
    try {
      const sw = await apiClient.post<{ id: string }>('/server-workloads', {
        name: values.server.name,
        description: values.server.description,
        isActive: true,
        host: values.server.host,
        port: Number(values.server.port) || 443,
        appProtocol: values.server.appProtocol,
        transportProtocol: values.server.transportProtocol || 'TCP',
        tls: values.server.tls,
        tlsVerification: values.server.forwardTlsVerification,
        requestedPort: Number(values.server.forwardToPort) || 0,
        requestedTls: values.server.forwardTls,
        authMethod: values.server.authMethod,
        authScheme: values.server.authScheme,
        authConfig: values.server.headerName ? { header_name: values.server.headerName } : {},
        httpHeaders: Object.fromEntries(
          (values.server.staticHeaders ?? []).filter((h) => h.name).map((h) => [h.name, h.value]),
        ),
        urlPath: values.server.urlPath,
        tags: (values.server.tags ?? []).map((t) => ({ key: t.key, value: t.value })),
      })
      serverWorkloadId = sw.id
    } catch (err) {
      throw new Error(`Failed to create Server Workload "${values.server.name}": ${apiErrorMessage(err)}`)
    }
  }

  const trustProviderIds = [...values.trustProviderIds]
  for (const tp of values.trustProviders ?? []) {
    // Entries hydrated from "Select Existing" carry existingId and are
    // already present in trustProviderIds — skip re-creating them, or
    // every save would mint a duplicate Trust Provider record.
    if (tp.existingId) continue
    if (tp.name) {
      try {
        const created = await apiClient.post<{ id: string }>('/trust-providers', {
          name: tp.name,
          description: tp.description,
          isActive: true,
          type: tp.trustProviderType,
          config: {
            certificate: tp.certificate,
            thumbprint: tp.thumbprint,
            subject: tp.subject,
            expiresAt: tp.expiresAt,
            oidcEndpoint: tp.oidcEndpoint,
            attestationMethod: tp.attestationMethod,
            samlMetadataUrl: tp.samlMetadataUrl,
            samlMetadataXml: tp.samlMetadataXml,
            symmetricKey: tp.symmetricKey,
            jwks: tp.jwks,
            publicKey: tp.publicKey,
          },
          matchRules: (tp.matchRules ?? []).filter((r) => r.attribute),
          tags: (tp.tags ?? []).map((t) => ({ key: t.key, value: t.value })),
        })
        trustProviderIds.push(created.id)
      } catch (err) {
        throw new Error(`Failed to create Trust Provider "${tp.name}": ${apiErrorMessage(err)}`)
      }
    }
  }

  const accessConditionIds = [...values.accessConditionIds]
  for (const ac of values.accessConditions ?? []) {
    // Entries hydrated from "Select Existing" carry existingId and are
    // already present in accessConditionIds — skip re-creating them, or
    // every save would mint a duplicate Access Condition record.
    if (ac.existingId) continue
    if (ac.name) {
      try {
        const config: Record<string, unknown> = {}
        if (ac.integration === 'time' || ac.days?.length) {
          config.timezone = ac.timezone
          config.days = ac.days
        }
        if (ac.integration === 'geoip' || ac.countries?.length) {
          config.countries = ac.countries
        }
        const created = await apiClient.post<{ id: string }>('/access-conditions', {
          name: ac.name,
          description: ac.description,
          isActive: true,
          type: ac.integration,
          config,
          tags: (ac.tags ?? []).map((t) => ({ key: t.key, value: t.value })),
        })
        accessConditionIds.push(created.id)
      } catch (err) {
        throw new Error(`Failed to create Access Condition "${ac.name}": ${apiErrorMessage(err)}`)
      }
    }
  }

  let credentialProviderId = values.credentialProviderId
  if (!credentialProviderId && values.credential.name) {
    try {
      const cp = await apiClient.post<{ id: string }>('/credential-providers', {
        name: values.credential.name,
        description: values.credential.description,
        isActive: true,
        type: values.credential.credentialType,
        config: {
          apiKey: values.credential.apiKey,
          username: values.credential.username,
          password: values.credential.password,
          tokenEndpointUrl: values.credential.tokenEndpointUrl,
          clientId: values.credential.clientId,
          clientSecret: values.credential.clientSecret,
          scopes: values.credential.scopes,
          credentialStyle: values.credential.credentialStyle,
          awsIamRoleArn: values.credential.awsIamRoleArn,
          audience: values.credential.audience,
          lifetimeSeconds: values.credential.lifetimeSeconds,
          subject: values.credential.subject,
          signingAlgorithm: values.credential.signingAlgorithm,
        },
        tags: (values.credential.tags ?? []).map((t) => ({ key: t.key, value: t.value })),
      })
      credentialProviderId = cp.id
    } catch (err) {
      throw new Error(`Failed to create Credential Provider "${values.credential.name}": ${apiErrorMessage(err)}`)
    }
  }

  const cpIds = credentialProviderId
    ? [{ credentialProviderId, mappingType: 'None', mappingValue: '' }]
    : []

  await apiClient.post<PolicyDTO>(BASE, {
    name: values.name,
    description: values.description,
    isActive: values.status !== 'inactive',
    clientWorkloadId,
    serverWorkloadId,
    trustProviderIds,
    credentialProviderIds: cpIds,
    accessConditionIds,
    tags: values.tags.map((t) => ({ key: t.key, value: t.value })),
  })
}

export async function savePolicy(id: string, values: PolicyFormValues): Promise<PolicyFormValues> {
  await apiClient.put<PolicyDTO>(`${BASE}/${id}`, toRequestBody(values))
  return values
}

export async function deletePolicy(id: string): Promise<void> {
  await apiClient.delete(`${BASE}/${id}`)
}

export { apiErrorMessage } from '@/shared/lib/api-client'

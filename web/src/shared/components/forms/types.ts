export interface FormTag {
  key: string
  value: string
}

export interface FormHeader {
  name: string
  value: string
}

export interface ClientWorkloadFormValues {
  name: string
  description: string
  identifiers: { type: string; values: string[] }[]
  tags: FormTag[]
}

export interface ServerWorkloadFormValues {
  name: string
  description: string
  host: string
  appProtocol: string
  transportProtocol: string
  port: string
  forwardToPort: string
  tls: boolean
  forwardTls: boolean
  forwardTlsVerification: string
  authMethod: string
  authScheme: string
  headerName: string
  urlPath: string
  staticHeaders: FormHeader[]
  tags: FormTag[]
}

export interface TrustProviderMatchRule {
  attribute: string
  value: string
}

export interface TrustProviderFormValues {
  name: string
  description: string
  trustProviderType: string
  matchRules: TrustProviderMatchRule[]
  certificate: string
  thumbprint: string
  subject: string
  expiresAt: string
  oidcEndpoint: string
  attestationMethod: string
  samlMetadataUrl: string
  samlMetadataXml: string
  symmetricKey: string
  jwks: string
  publicKey: string
  tags: FormTag[]
}

export interface GeoIpCountryEntry {
  country: string
  subdivisions: string[]
}

export interface TimeConditionDay {
  day: string
  startTime: string
  endTime: string
  enabled: boolean
}

export interface AccessConditionFormValues {
  name: string
  description: string
  integration: string
  countries: GeoIpCountryEntry[]
  timezone: string
  days: TimeConditionDay[]
  tags: FormTag[]
}

export interface CredentialProviderFormValues {
  name: string
  description: string
  credentialType: string
  tags: FormTag[]

  // API Key
  apiKey: string

  // Username & Password
  username: string
  password: string

  // OAuth Client Credentials / Authorization Code / MCP User-Based
  tokenEndpointUrl: string
  authorizationUrl: string
  introspectionUrl: string
  clientId: string
  clientSecret: string
  scopes: string
  credentialStyle: string
  mcpServerUrl: string
  mcpResource: string
  mcpPkceRequired: boolean
  oauthPkceRequired: boolean
  oauthUrl: string
  oauthLifetimeValue: number
  oauthLifetimeUnit: string

  // AWS STS Federation
  awsIamRoleArn: string

  // AWS Secrets Manager
  integrationId: string
  secretArn: string
  credentialValueType: string
  secretKey: string
  usernameKey: string
  passwordKey: string
  privateNetworkAccess: boolean

  // Azure Entra WIF
  azureTenantId: string
  azureClientId: string
  jwtTokenSubject: string
  jwtTokenScope: string

  // Azure Key Vault
  keyVaultUrl: string
  keyVaultSecretName: string

  // Google GCP WIF
  gcpServiceAccountEmail: string
  gcpProjectNumber: string
  gcpWorkloadIdentityPool: string
  gcpWifProvider: string

  // Claude WIF
  claudeApiAudience: string
  claudeFederationRuleId: string
  claudeOrganizationId: string
  claudeScope: string
  claudeServiceAccountId: string
  claudeWorkspaceId: string

  // OpenAI WIF
  openaiIdentityProviderId: string
  openaiServiceAccountId: string

  // JWT (Snowflake)
  snowflakeAccountId: string
  snowflakeUsername: string

  // JWT-SVID
  spiffeTrustDomain: string
  signingAlgorithm: string

  // OIDC ID Token
  subject: string
  customClaims: { name: string; value: string; valueType: string }[]

  // Vault Client Token
  vaultHost: string
  vaultPort: string
  vaultTls: boolean
  vaultAuthPath: string
  vaultRole: string
  vaultNamespace: string
  vaultForwardingConfig: string

  // Managed GitLab
  gitlabGroupId: string
  gitlabProjectId: string
  gitlabScopes: string
  gitlabAccessLevel: string
  gitlabServiceAccountUsername: string
  gitlabRotationTime: number
  gitlabRotationUnit: string

  // X.509-SVID
  standaloneCaId: string
  spiffeId: string
  keyUsage: string
  extendedKeyUsage: string[]

  // Shared
  audience: string
  lifetimeSeconds: number
}

export const EMPTY_CLIENT_WORKLOAD: ClientWorkloadFormValues = {
  name: '',
  description: '',
  identifiers: [{ type: '', values: [''] }],
  tags: [],
}

export const EMPTY_SERVER_WORKLOAD: ServerWorkloadFormValues = {
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
}

export const EMPTY_TRUST_PROVIDER: TrustProviderFormValues = {
  name: '',
  description: '',
  trustProviderType: '',
  matchRules: [{ attribute: '', value: '' }],
  certificate: '',
  thumbprint: '',
  subject: '',
  expiresAt: '',
  oidcEndpoint: '',
  attestationMethod: 'oidc_discovery',
  samlMetadataUrl: '',
  samlMetadataXml: '',
  symmetricKey: '',
  jwks: '',
  publicKey: '',
  tags: [],
}

export const EMPTY_ACCESS_CONDITION: AccessConditionFormValues = {
  name: '',
  description: '',
  integration: '',
  countries: [{ country: '', subdivisions: [''] }],
  timezone: '',
  days: [
    { day: 'Monday', startTime: '09:00', endTime: '18:00', enabled: false },
    { day: 'Tuesday', startTime: '09:00', endTime: '18:00', enabled: false },
    { day: 'Wednesday', startTime: '09:00', endTime: '18:00', enabled: false },
    { day: 'Thursday', startTime: '09:00', endTime: '18:00', enabled: false },
    { day: 'Friday', startTime: '09:00', endTime: '18:00', enabled: false },
    { day: 'Saturday', startTime: '09:00', endTime: '18:00', enabled: false },
    { day: 'Sunday', startTime: '09:00', endTime: '18:00', enabled: false },
  ],
  tags: [],
}

export const EMPTY_CREDENTIAL_PROVIDER: CredentialProviderFormValues = {
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
}

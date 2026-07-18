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
  // Stored config values (flat shape for simplicity)
  apiKey: string
  username: string
  password: string
  tokenEndpointUrl: string
  authorizationUrl: string
  introspectionUrl: string
  clientId: string
  clientSecret: string
  scopes: string
  awsIamRoleArn: string
  audience: string
  lifetimeSeconds: number
  vaultAddress: string
  vaultRole: string
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
  apiKey: '',
  username: '',
  password: '',
  tokenEndpointUrl: '',
  authorizationUrl: '',
  introspectionUrl: '',
  clientId: '',
  clientSecret: '',
  scopes: '',
  awsIamRoleArn: '',
  audience: '',
  lifetimeSeconds: 3600,
  vaultAddress: '',
  vaultRole: '',
}

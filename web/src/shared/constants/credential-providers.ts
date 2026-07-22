export interface CredentialTypeOption {
  value: string
  label: string
}

export const CREDENTIAL_TYPES: CredentialTypeOption[] = [
  { value: 'dipole_access_token', label: 'Dipole Access Token' },
  { value: 'api_key', label: 'API Key' },
  { value: 'aws_secrets_manager', label: 'AWS Secrets Manager Value' },
  { value: 'aws_sts_federation', label: 'AWS Security Token Service Federation' },
  { value: 'azure_entra_wif', label: 'Azure Entra Identity Federation' },
  { value: 'azure_key_vault', label: 'Azure Key Vault Secret Value' },
  { value: 'claude_wif', label: 'Claude Workload Identity Federation' },
  { value: 'google_wif', label: 'Google GCP Workload Identity Federation' },
  { value: 'jwt', label: 'JSON Web Token (JWT)' },
  { value: 'jwt_svid', label: 'JWT-SVID Token' },
  { value: 'managed_gitlab', label: 'Managed GitLab Account' },
  { value: 'mcp_user_access_token', label: 'MCP User-Based Access Token' },
  { value: 'oauth_authorization_code', label: 'OAuth 2.0 Authorization Code' },
  { value: 'oauth_client_credentials', label: 'OAuth 2.0 Client Credentials' },
  { value: 'oidc_id_token', label: 'OIDC ID Token' },
  { value: 'openai_wif', label: 'OpenAI Workload Identity Federation' },
  { value: 'username_password', label: 'Username & Password' },
  { value: 'vault_client_token', label: 'Vault Client Token' },
  { value: 'x509_svid', label: 'X.509-SVID Certificate' },
]

export const CONDITION_INTEGRATIONS: CredentialTypeOption[] = [
  { value: 'geoip', label: 'Dipole GeoIP Condition' },
  { value: 'time', label: 'Dipole Time Condition' },
]

export const COUNTRIES: CredentialTypeOption[] = [
  { value: 'IN', label: 'India' },
  { value: 'US', label: 'United States of America (the)' },
  { value: 'GB', label: 'United Kingdom' },
  { value: 'DE', label: 'Germany' },
  { value: 'FR', label: 'France' },
  { value: 'JP', label: 'Japan' },
  { value: 'AU', label: 'Australia' },
  { value: 'CA', label: 'Canada' },
  { value: 'SG', label: 'Singapore' },
  { value: 'AE', label: 'United Arab Emirates' },
]

export const SUBDIVISIONS: Record<string, { value: string; label: string }[]> = {
  IN: [
    { value: 'KA', label: 'Karnataka' },
    { value: 'MH', label: 'Maharashtra' },
    { value: 'DL', label: 'Delhi' },
    { value: 'TN', label: 'Tamil Nadu' },
    { value: 'TG', label: 'Telangana' },
    { value: 'UP', label: 'Uttar Pradesh' },
    { value: 'GJ', label: 'Gujarat' },
  ],
  US: [
    { value: 'WA', label: 'Washington' },
    { value: 'CA', label: 'California' },
    { value: 'NY', label: 'New York' },
    { value: 'TX', label: 'Texas' },
    { value: 'VA', label: 'Virginia' },
    { value: 'OR', label: 'Oregon' },
  ],
  GB: [
    { value: 'ENG', label: 'England' },
    { value: 'SCT', label: 'Scotland' },
    { value: 'WLS', label: 'Wales' },
  ],
}

export const TIMEZONES: CredentialTypeOption[] = [
  { value: 'Asia/Kolkata', label: 'Asia/Kolkata (IST)' },
  { value: 'America/New_York', label: 'America/New_York (EST)' },
  { value: 'America/Los_Angeles', label: 'America/Los_Angeles (PST)' },
  { value: 'Europe/London', label: 'Europe/London (GMT)' },
  { value: 'Asia/Tokyo', label: 'Asia/Tokyo (JST)' },
  { value: 'UTC', label: 'UTC' },
]

export const WEEKDAYS = ['Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday', 'Sunday'] as const

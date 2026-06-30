import { useState } from 'react'
import { Dropdown } from '@/shared/components/Dropdown'
import { TagsField } from '@/shared/components/TagsField'
import { CredentialTypeConfig } from '@/features/policies/components/tabs/credential-provider/CredentialProviderTab'

const CREDENTIAL_TYPES = [
  { value: 'aembit_access_token', label: 'Dipole Access Token' },
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
  { value: 'username_password', label: 'Username & Password' },
  { value: 'vault_client_token', label: 'Vault Client Token' },
  { value: 'x509_svid', label: 'X.509-SVID Certificate' },
]

export function CredentialProviderForm() {
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [credentialType, setCredentialType] = useState('')

  return (
    <div className="space-y-5">
      <div>
        <label className="block text-xs font-medium text-gray-700 mb-1.5">Name</label>
        <input
          type="text"
          placeholder="Name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-gray-300"
        />
      </div>

      <div>
        <label className="block text-xs font-medium text-gray-700 mb-1.5">Description</label>
        <textarea
          placeholder="Description (Optional)"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          rows={2}
          className="w-full px-3 py-2 border border-border rounded-md text-sm resize-y focus:outline-none focus:ring-1 focus:ring-gray-300"
        />
      </div>

      <div>
        <h4 className="text-xs font-semibold text-gray-800 mb-2">Credential Type</h4>
        <Dropdown
          options={CREDENTIAL_TYPES}
          value={credentialType}
          onChange={setCredentialType}
          placeholder=""
        />
      </div>

      {credentialType && <CredentialTypeConfig type={credentialType} />}

      <div>
        <label className="block text-xs font-medium text-gray-700 mb-1.5">Tags</label>
        <TagsField />
      </div>
    </div>
  )
}

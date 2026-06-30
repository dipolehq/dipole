import { useState } from 'react'
import { Dropdown } from '@/shared/components/Dropdown'
import { ExistingCredentialProviderSelector } from '@/features/policies/components/tabs/credential-provider/ExistingCredentialProviderSelector'
import { CredentialProviderForm } from '@/shared/components/forms/CredentialProviderForm'

interface CredentialProviderTabProps {
  selectedId: string
  onSelect: (id: string) => void
}

export function CredentialProviderTab({ selectedId, onSelect }: CredentialProviderTabProps) {
  const [mode, setMode] = useState<'new' | 'existing'>('new')

  return (
    <div className="space-y-6">
      <h3 className="text-sm font-semibold text-gray-900">Credential Provider</h3>

      <div className="flex items-center gap-4 border-b border-border pb-3">
        <button
          onClick={() => setMode('new')}
          className={`text-xs font-medium transition-colors ${mode === 'new' ? 'text-orange-600' : 'text-gray-500 hover:text-gray-700'}`}
        >
          Add New
        </button>
        <button
          onClick={() => setMode('existing')}
          className={`text-xs font-medium transition-colors ${mode === 'existing' ? 'text-orange-600' : 'text-gray-500 hover:text-gray-700'}`}
        >
          Select Existing
        </button>
      </div>

      {mode === 'existing' ? (
        <ExistingCredentialProviderSelector
          selectedId={selectedId}
          onSelect={onSelect}
          onCancel={() => setMode('new')}
        />
      ) : (
        <CredentialProviderForm />
      )}
    </div>
  )
}

export function CredentialTypeConfig({ type }: { type: string }) {
  switch (type) {
    case 'aembit_access_token':
      return <DipoleAccessTokenConfig />
    case 'api_key':
      return (
        <div className="space-y-3">
          <Field label="API Key" placeholder="Enter API key" type="password" />
        </div>
      )
    case 'aws_secrets_manager':
      return <AwsSecretsManagerConfig />
    case 'aws_sts_federation':
      return (
        <div className="space-y-4">
          <ReadOnlyField label="OIDC Issuer URL" value="https://<tenant>.id.ap-south-1.dipolehq.com" />
          <Field label="AWS IAM Role ARN" placeholder="AWS IAM Role Arn" />
          <ReadOnlyField label="Dipole IdP Token Audience" value="sts.amazonaws.com" />
          <div>
            <label className="block text-[11px] text-gray-500 mb-1">Lifetime</label>
            <div className="flex items-center border border-border rounded-md overflow-visible w-64">
              <input type="number" defaultValue={3600} className="flex-1 px-3 py-2 text-sm focus:outline-none min-w-0" />
              <span className="px-4 py-2 text-sm text-gray-500 bg-gray-50 border-l border-border whitespace-nowrap">Seconds</span>
            </div>
          </div>
        </div>
      )
    case 'azure_entra_wif':
      return (
        <div className="space-y-3">
          <ReadOnlyField label="OIDC Issuer URL" value="https://<tenant>.id.ap-south-1.dipolehq.com" />
          <Field label="Audience" placeholder="Audience" />
          <Field label="JWT Token Subject" placeholder="JWT Token Subject" />
          <Field label="JWT Token Scope" placeholder="JWT Token Scope" />
          <Field label="Azure Tenant ID" placeholder="Azure Tenant ID" />
          <Field label="Azure Client ID" placeholder="Azure Client ID" />
        </div>
      )
    case 'azure_key_vault':
      return <AzureKeyVaultConfig />
    case 'google_wif':
      return (
        <div className="space-y-4">
          <ReadOnlyField label="OIDC Issuer URL" value="https://<tenant>.id.ap-south-1.dipolehq.com/" />
          <Field label="Audience" placeholder="Audience" />
          <Field label="Service Account Email" placeholder="Service Account Email" />
          <div>
            <label className="block text-[11px] text-gray-500 mb-1">Lifetime</label>
            <div className="flex items-center border border-border rounded-md overflow-visible w-64">
              <input type="number" defaultValue={3600} className="flex-1 px-3 py-2 text-sm focus:outline-none min-w-0" />
              <span className="px-4 py-2 text-sm text-gray-500 bg-gray-50 border-l border-border whitespace-nowrap">Seconds</span>
            </div>
          </div>
        </div>
      )
    case 'jwt':
      return <JwtConfig />
    case 'managed_gitlab':
      return <ManagedGitLabConfig />
    case 'oauth_authorization_code':
      return <OAuthAuthorizationCodeConfig />
    case 'oauth_client_credentials':
      return <OAuthClientCredentialsConfig />
    case 'oidc_id_token':
      return <OidcIdTokenConfig />
    case 'username_password':
      return (
        <div className="space-y-3">
          <Field label="Username" placeholder="Username" />
          <Field label="Password" placeholder="Password" type="password" />
        </div>
      )
    case 'claude_wif':
      return (
        <div className="space-y-4">
          <ReadOnlyField label="OIDC Issuer URL" value="https://<tenant>.id.ap-south-1.dipolehq.com" />
          <Field label="Federation Rule ID" placeholder="Federation Rule ID" />
          <Field label="Organization ID" placeholder="Organization ID" />
          <Field label="Scope" placeholder="workspace:developer" />
          <Field label="Service Account ID" placeholder="Service Account ID" />
          <Field label="Audience" placeholder="Audience" />
          <Field label="Workspace ID" placeholder="Workspace ID" />
          <div>
            <label className="block text-[11px] text-gray-500 mb-1">Lifetime</label>
            <div className="flex items-center border border-border rounded-md overflow-visible w-64">
              <input type="number" defaultValue={3600} className="flex-1 px-3 py-2 text-sm focus:outline-none min-w-0" />
              <span className="px-4 py-2 text-sm text-gray-500 bg-gray-50 border-l border-border whitespace-nowrap">Seconds</span>
            </div>
          </div>
        </div>
      )
    case 'jwt_svid':
      return <JwtSvidConfig />
    case 'mcp_user_access_token':
      return (
        <div className="space-y-4">
          <Field label="Token Endpoint URL" placeholder="Token Endpoint URL" />
          <Field label="Client ID" placeholder="Client Id" />
          <Field label="Client Secret" placeholder="Client Secret" type="password" />
          <Field label="Scopes (Optional)" placeholder="Scopes (Optional)" />
          <Field label="Authorization URL" placeholder="Authorization URL" />
        </div>
      )
    case 'x509_svid':
      return <X509SvidConfig />
    case 'vault_client_token':
      return <VaultClientTokenConfig />
    default:
      return null
  }
}

function Field({ label, placeholder, type = 'text' }: { label: string; placeholder: string; type?: string }) {
  return (
    <div>
      <label className="block text-[11px] text-gray-500 mb-1">{label}</label>
      <input
        type={type}
        placeholder={placeholder}
        className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-gray-300"
      />
    </div>
  )
}

function ReadOnlyField({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <label className="block text-[11px] text-gray-500 mb-1">{label}</label>
      <div className="flex items-center border border-border rounded-md overflow-hidden bg-gray-50">
        <input type="text" readOnly value={value} className="flex-1 px-3 py-2 text-sm text-gray-500 bg-gray-50 focus:outline-none" />
        <button
          onClick={() => navigator.clipboard.writeText(value)}
          className="px-3 py-2 border-l border-border text-gray-400 hover:text-gray-600"
          title="Copy"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><rect x="9" y="9" width="13" height="13" rx="2" ry="2" strokeWidth="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" strokeWidth="2"/></svg>
        </button>
      </div>
    </div>
  )
}



function DipoleAccessTokenConfig() {
  const [refreshToken, setRefreshToken] = useState(false)
  const [role, setRole] = useState('')

  return (
    <div className="space-y-4">
      <ReadOnlyField label="Audience" value="<tenant>.api.ap-south-1.dipolehq.com" />

      <div>
        <label className="block text-[11px] text-gray-500 mb-1">Role</label>
        <Dropdown
          options={[
            { value: 'superadmin', label: 'SuperAdmin' },
            { value: 'auditor', label: 'Auditor' },
          ]}
          value={role}
          onChange={setRole}
          placeholder="Role"
        />
      </div>

      <div>
        <label className="block text-[11px] text-gray-500 mb-1">Lifetime</label>
        <div className="flex items-center border border-border rounded-md overflow-visible w-64">
          <input type="number" defaultValue={300} className="flex-1 px-3 py-2 text-sm focus:outline-none min-w-0" />
          <span className="px-4 py-2 text-sm text-gray-500 bg-gray-50 border-l border-border whitespace-nowrap">Seconds</span>
        </div>
      </div>

      <label className="flex items-center gap-2 cursor-pointer">
        <div
          onClick={() => setRefreshToken(!refreshToken)}
          className={`relative w-9 h-5 rounded-full transition-colors ${refreshToken ? 'bg-orange-500' : 'bg-gray-300'}`}
        >
          <span className={`absolute top-0.5 left-0.5 w-4 h-4 bg-white rounded-full shadow transition-transform ${refreshToken ? 'translate-x-4' : ''}`} />
        </div>
        <span className="text-xs text-gray-700">Enable Refresh Token Support</span>
        <span className="text-gray-400 text-xs" title="When enabled, a refresh token will be issued alongside the access token">ⓘ</span>
      </label>
    </div>
  )
}


function AwsSecretsManagerConfig() {
  const [credValueType, setCredValueType] = useState('single_value')
  const [privateNetwork, setPrivateNetwork] = useState(false)

  return (
    <div className="space-y-4">
      <div>
        <label className="block text-[11px] text-gray-500 mb-1">Credential Provider Integration</label>
        <Dropdown
          options={[]}
          value=""
          onChange={() => {}}
          placeholder="Credential Provider Integration"
        />
      </div>

      <Field label="AWS Secrets Manager Secret ARN" placeholder="AWS Secrets Manager Secret Arn" />

      <div>
        <label className="block text-[11px] text-gray-500 mb-1">Credential Value Type</label>
        <Dropdown
          options={[
            { value: 'plain_text', label: 'Plain Text' },
            { value: 'single_value', label: 'Single Value' },
            { value: 'username_password', label: 'Username/Password' },
          ]}
          value={credValueType}
          onChange={setCredValueType}
          placeholder="Single Value"
        />
      </div>

      <Field label="Secret Key" placeholder="Secret Key" />

      <label className="flex items-center gap-2 cursor-pointer">
        <input
          type="checkbox"
          checked={privateNetwork}
          onChange={(e) => setPrivateNetwork(e.target.checked)}
          className="w-4 h-4 rounded border-gray-300 text-orange-500 focus:ring-orange-200"
        />
        <div>
          <span className="text-xs text-gray-700">Private Network Access</span>
          <p className="text-[10px] text-gray-400">Requires Dipole AgentProxy 1.26.* or greater</p>
        </div>
      </label>
    </div>
  )
}


function AzureKeyVaultConfig() {
  const [credValueType, setCredValueType] = useState('single_value')
  const [privateNetwork, setPrivateNetwork] = useState(false)

  return (
    <div className="space-y-4">
      <div>
        <label className="block text-[11px] text-gray-500 mb-1">Credential Provider Integration</label>
        <Dropdown
          options={[]}
          value=""
          onChange={() => {}}
          placeholder="Credential Provider Integration"
        />
      </div>

      <div>
        <label className="block text-[11px] text-gray-500 mb-1">Credential Value Type</label>
        <Dropdown
          options={[
            { value: 'single_value', label: 'Single Value' },
            { value: 'username_password', label: 'Username/Password' },
          ]}
          value={credValueType}
          onChange={setCredValueType}
          placeholder="Single Value"
        />
      </div>

      {credValueType === 'single_value' && (
        <Field label="Secret Name" placeholder="Secret Name" />
      )}

      {credValueType === 'username_password' && (
        <>
          <Field label="Username Secret Name" placeholder="Username Secret Name" />
          <Field label="Password Secret Name" placeholder="Password Secret Name" />
        </>
      )}

      <label className="flex items-center gap-2 cursor-pointer">
        <input
          type="checkbox"
          checked={privateNetwork}
          onChange={(e) => setPrivateNetwork(e.target.checked)}
          className="w-4 h-4 rounded border-gray-300 text-orange-500 focus:ring-orange-200"
        />
        <div>
          <span className="text-xs text-gray-700">Private Network Access</span>
          <p className="text-[10px] text-gray-400">Requires Dipole AgentProxy 1.26.* or greater</p>
        </div>
      </label>
    </div>
  )
}


function JwtConfig() {
  const [tokenConfig, setTokenConfig] = useState('snowflake_keypair')

  return (
    <div className="space-y-4">
      <div>
        <label className="block text-[11px] text-gray-500 mb-1">Token Configuration</label>
        <Dropdown
          options={[
            { value: 'snowflake_keypair', label: 'Snowflake Key Pair Authentication' },
            { value: 'custom', label: 'Custom' },
          ]}
          value={tokenConfig}
          onChange={setTokenConfig}
          placeholder="Select configuration..."
        />
      </div>

      {tokenConfig === 'snowflake_keypair' && (
        <>
          <div className="grid grid-cols-2 gap-3">
            <Field label="Snowflake Account ID" placeholder="Snowflake Account ID" />
            <Field label="Username" placeholder="Username" />
          </div>
          <div>
            <label className="block text-[11px] text-gray-500 mb-1">Snowflake Alter User Command</label>
            <div className="relative">
              <textarea
                placeholder="Snowflake Alter User Command"
                rows={5}
                readOnly
                className="w-full px-3 py-2 border border-border rounded-md text-sm resize-y bg-gray-50 focus:outline-none"
              />
              <button
                onClick={() => {}}
                className="absolute top-2 right-2 text-gray-400 hover:text-gray-600"
                title="Copy"
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><rect x="9" y="9" width="13" height="13" rx="2" ry="2" strokeWidth="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" strokeWidth="2"/></svg>
              </button>
            </div>
          </div>
        </>
      )}

      {tokenConfig === 'custom' && (
        <>
          <Field label="Audience" placeholder="Audience" />
          <Field label="Subject" placeholder="Subject" />
          <div>
            <label className="block text-[11px] text-gray-500 mb-1">Lifetime</label>
            <div className="flex items-center border border-border rounded-md overflow-visible w-64">
              <input type="number" defaultValue={3600} className="flex-1 px-3 py-2 text-sm focus:outline-none min-w-0" />
              <span className="px-4 py-2 text-sm text-gray-500 bg-gray-50 border-l border-border whitespace-nowrap">Seconds</span>
            </div>
          </div>
          <Field label="Custom Claims (JSON)" placeholder='{"key": "value"}' />
        </>
      )}
    </div>
  )
}


function ManagedGitLabConfig() {
  const [accessLevel, setAccessLevel] = useState('')
  const [rotationUnit, setRotationUnit] = useState('days')

  return (
    <div className="space-y-4">
      <div>
        <label className="block text-[11px] text-gray-500 mb-1">Select GitLab Integration</label>
        <Dropdown
          options={[]}
          value=""
          onChange={() => {}}
          placeholder="Select GitLab Integration"
        />
      </div>

      <Field label="Service Account Username" placeholder="Service Account Username" />
      <Field label="GitLab Group IDs or Paths" placeholder="GitLab Group IDs or Paths" />
      <Field label="GitLab Project IDs or Paths" placeholder="GitLab Project IDs or Paths" />

      <div>
        <label className="block text-[11px] text-gray-500 mb-1">Access Level</label>
        <Dropdown
          options={[
            { value: 'minimal_access', label: 'Minimal Access' },
            { value: 'no_access', label: 'No access' },
            { value: 'guest', label: 'Guest' },
            { value: 'planner', label: 'Planner' },
            { value: 'reporter', label: 'Reporter' },
            { value: 'developer', label: 'Developer' },
            { value: 'maintainer', label: 'Maintainer' },
            { value: 'owner', label: 'Owner' },
          ]}
          value={accessLevel}
          onChange={setAccessLevel}
          placeholder="Access Level"
        />
      </div>

      <Field label="Scopes" placeholder="Scopes" />

      <div>
        <label className="block text-[11px] text-gray-500 mb-1">Rotation Time</label>
        <div className="flex items-center border border-border rounded-md overflow-visible w-64">
          <input type="number" defaultValue={1} className="flex-1 px-3 py-2 text-sm focus:outline-none min-w-0" />
          <select
            value={rotationUnit}
            onChange={(e) => setRotationUnit(e.target.value)}
            className="px-3 py-2 text-sm text-gray-500 bg-gray-50 border-l border-border focus:outline-none"
          >
            <option value="days">days</option>
            <option value="hours">hours</option>
          </select>
        </div>
      </div>
    </div>
  )
}


function OAuthAuthorizationCodeConfig() {
  const [pkceRequired, setPkceRequired] = useState(false)
  const [lifetimeUnit, setLifetimeUnit] = useState('years')
  const [params, setParams] = useState<{ name: string; value: string }[]>([{ name: '', value: '' }])

  function addParam() {
    setParams([...params, { name: '', value: '' }])
  }

  function updateParam(idx: number, field: 'name' | 'value', val: string) {
    setParams(params.map((p, i) => i === idx ? { ...p, [field]: val } : p))
  }

  function removeParam(idx: number) {
    if (params.length > 1) {
      setParams(params.filter((_, i) => i !== idx))
    }
  }

  return (
    <div className="space-y-4">
      <ReadOnlyField label="Callback URL" value="https://api.ap-south-1.dipolehq.com/api/v2/credential-providers/callback" />
      <Field label="Client ID" placeholder="Client Id" />
      <Field label="Client Secret" placeholder="Client Secret" type="password" />
      <Field label="Scopes" placeholder="Scopes" />

      <div>
        <div className="flex items-center border border-border rounded-md overflow-hidden">
          <span className="px-3 py-2 text-sm text-gray-500 bg-gray-50 border-r border-border shrink-0">https://</span>
          <input type="text" placeholder="OAuth URL" className="flex-1 px-3 py-2 text-sm focus:outline-none" />
          <button className="px-3 py-2 text-sm text-gray-400 bg-gray-50 border-l border-border shrink-0 hover:text-gray-600">URL Discovery</button>
        </div>
      </div>

      <div className="pl-6 space-y-3">
        <Field label="Authorization URL" placeholder="Authorization URL" />
        <Field label="Token URL" placeholder="Token URL" />
        <Field label="Introspection URL" placeholder="Introspection URL" />
      </div>

      <label className="flex items-center gap-2 cursor-pointer">
        <input
          type="checkbox"
          checked={pkceRequired}
          onChange={(e) => setPkceRequired(e.target.checked)}
          className="w-4 h-4 rounded border-gray-300 text-orange-500 focus:ring-orange-200"
        />
        <span className="text-xs text-gray-700">PKCE Required</span>
      </label>

      <div className="flex items-center gap-3">
        <span className="text-xs font-medium text-gray-700">Lifetime:</span>
        <div className="flex items-center border border-border rounded-md overflow-visible">
          <input type="number" defaultValue={1} className="w-20 px-3 py-2 text-sm focus:outline-none min-w-0" />
          <select
            value={lifetimeUnit}
            onChange={(e) => setLifetimeUnit(e.target.value)}
            className="px-3 py-2 text-sm text-gray-600 bg-gray-50 border-l border-border focus:outline-none"
          >
            <option value="years">Years</option>
            <option value="months">Months</option>
            <option value="days">Days</option>
            <option value="hours">Hours</option>
          </select>
        </div>
      </div>

      <div className="border border-border rounded-md p-4">
        <h4 className="text-xs font-semibold text-gray-800 mb-3">Additional Parameters</h4>
        <div className="space-y-2">
          {params.map((param, idx) => (
            <div key={idx} className="flex items-center gap-2">
              <input
                type="text"
                placeholder="Name"
                value={param.name}
                onChange={(e) => updateParam(idx, 'name', e.target.value)}
                className="flex-1 px-3 py-2 border border-border rounded-md text-xs focus:outline-none focus:ring-1 focus:ring-gray-300"
              />
              <input
                type="text"
                placeholder="Value"
                value={param.value}
                onChange={(e) => updateParam(idx, 'value', e.target.value)}
                className="flex-1 px-3 py-2 border border-border rounded-md text-xs focus:outline-none focus:ring-1 focus:ring-gray-300"
              />
              <button onClick={() => removeParam(idx)} className="text-orange-400 hover:text-red-500 text-sm">×</button>
            </div>
          ))}
        </div>
        <button onClick={addParam} className="mt-2 text-xs text-orange-500 hover:text-orange-600 font-medium">
          + New Parameter
        </button>
      </div>
    </div>
  )
}


function OAuthClientCredentialsConfig() {
  const [credStyle, setCredStyle] = useState('authorization_header')
  const [params, setParams] = useState<{ name: string; value: string }[]>([])

  function addParam() {
    setParams([...params, { name: '', value: '' }])
  }

  function updateParam(idx: number, field: 'name' | 'value', val: string) {
    setParams(params.map((p, i) => i === idx ? { ...p, [field]: val } : p))
  }

  function removeParam(idx: number) {
    setParams(params.filter((_, i) => i !== idx))
  }

  return (
    <div className="space-y-4">
      <Field label="Token Endpoint URL" placeholder="Token Endpoint Url" />
      <Field label="Client ID" placeholder="Client Id" />
      <Field label="Client Secret" placeholder="Client Secret" type="password" />
      <Field label="Scopes (Optional)" placeholder="Scopes (Optional)" />

      <div className="border border-border rounded-md p-4">
        <h4 className="text-xs font-semibold text-gray-800 mb-3">Credential Style</h4>
        <div className="flex items-center gap-6">
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="radio"
              name="credStyle"
              checked={credStyle === 'authorization_header'}
              onChange={() => setCredStyle('authorization_header')}
              className="w-4 h-4 text-orange-500 focus:ring-orange-200"
            />
            <span className="text-xs text-gray-700">Authorization Header</span>
          </label>
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="radio"
              name="credStyle"
              checked={credStyle === 'post_body'}
              onChange={() => setCredStyle('post_body')}
              className="w-4 h-4 text-orange-500 focus:ring-orange-200"
            />
            <span className="text-xs text-gray-700">POST Body</span>
          </label>
        </div>
      </div>

      <div className="border border-border rounded-md p-4">
        <h4 className="text-xs font-semibold text-gray-800 mb-3">Additional Parameters</h4>
        {params.length > 0 && (
          <div className="space-y-2 mb-3">
            {params.map((param, idx) => (
              <div key={idx} className="flex items-center gap-2">
                <input
                  type="text"
                  placeholder="Name"
                  value={param.name}
                  onChange={(e) => updateParam(idx, 'name', e.target.value)}
                  className="flex-1 px-3 py-2 border border-border rounded-md text-xs focus:outline-none focus:ring-1 focus:ring-gray-300"
                />
                <input
                  type="text"
                  placeholder="Value"
                  value={param.value}
                  onChange={(e) => updateParam(idx, 'value', e.target.value)}
                  className="flex-1 px-3 py-2 border border-border rounded-md text-xs focus:outline-none focus:ring-1 focus:ring-gray-300"
                />
                <button onClick={() => removeParam(idx)} className="text-orange-400 hover:text-red-500 text-sm">×</button>
              </div>
            ))}
          </div>
        )}
        <button onClick={addParam} className="text-xs text-orange-500 hover:text-orange-600 font-medium">
          + New Parameter
        </button>
      </div>
    </div>
  )
}


function OidcIdTokenConfig() {
  const [subjectType, setSubjectType] = useState('literal')
  const [audienceType, setAudienceType] = useState('literal')
  const [signingAlgorithm, setSigningAlgorithm] = useState('')
  const [lifetimeUnit, setLifetimeUnit] = useState('minutes')
  const [refreshToken, setRefreshToken] = useState(false)
  const [absoluteLifetimeUnit, setAbsoluteLifetimeUnit] = useState('month')
  const [claims, setClaims] = useState<{ name: string; value: string; valueType: string }[]>([])

  function addClaim() {
    setClaims([...claims, { name: '', value: '', valueType: 'literal' }])
  }

  function updateClaim(idx: number, field: string, val: string) {
    setClaims(claims.map((c, i) => i === idx ? { ...c, [field]: val } : c))
  }

  function removeClaim(idx: number) {
    setClaims(claims.filter((_, i) => i !== idx))
  }

  return (
    <div className="space-y-4">
      <div>
        <label className="block text-[11px] text-gray-500 mb-1">Subject</label>
        <div className="flex items-center border border-border rounded-md overflow-hidden">
          <input type="text" placeholder="Subject" className="flex-1 px-3 py-2 text-sm focus:outline-none" />
          <select
            value={subjectType}
            onChange={(e) => setSubjectType(e.target.value)}
            className="px-3 py-2 text-sm text-gray-600 bg-gray-50 border-l border-border focus:outline-none"
          >
            <option value="literal">Literal</option>
            <option value="dynamic">Dynamic</option>
          </select>
        </div>
      </div>

      <div className="flex items-start gap-3">
        <div className="flex-1">
          <ReadOnlyField label="Issuer" value="https://<tenant>.id.ap-south-1.dipolehq.com" />
        </div>
        <div className="w-40">
          <label className="block text-[11px] text-gray-500 mb-1">Lifetime</label>
          <div className="flex items-center border border-border rounded-md overflow-visible">
            <input type="number" defaultValue={15} className="w-16 px-3 py-2 text-sm focus:outline-none min-w-0" />
            <select
              value={lifetimeUnit}
              onChange={(e) => setLifetimeUnit(e.target.value)}
              className="px-2 py-2 text-sm text-gray-600 bg-gray-50 border-l border-border focus:outline-none"
            >
              <option value="minutes">Minutes</option>
              <option value="hours">Hours</option>
              <option value="days">Days</option>
            </select>
          </div>
        </div>
      </div>

      <div>
        <label className="block text-[11px] text-gray-500 mb-1">Signing Algorithm Type</label>
        <Dropdown
          options={[
            { value: 'rs256', label: 'RSASSA-PKCS1-v1_5 using SHA-256' },
            { value: 'es256', label: 'ECDSA using P-256 and SHA-256' },
          ]}
          value={signingAlgorithm}
          onChange={setSigningAlgorithm}
          placeholder="Signing Algorithm Type"
        />
      </div>

      <div>
        <label className="block text-[11px] text-gray-500 mb-1">Audience</label>
        <div className="flex items-center border border-border rounded-md overflow-hidden">
          <input type="text" placeholder="Audience" className="flex-1 px-3 py-2 text-sm focus:outline-none" />
          <select
            value={audienceType}
            onChange={(e) => setAudienceType(e.target.value)}
            className="px-3 py-2 text-sm text-gray-600 bg-gray-50 border-l border-border focus:outline-none"
          >
            <option value="literal">Literal</option>
            <option value="dynamic">Dynamic</option>
          </select>
        </div>
      </div>

      <div>
        <label className="flex items-center gap-2 cursor-pointer">
          <div
            onClick={() => setRefreshToken(!refreshToken)}
            className={`relative w-9 h-5 rounded-full transition-colors ${refreshToken ? 'bg-orange-500' : 'bg-gray-300'}`}
          >
            <span className={`absolute top-0.5 left-0.5 w-4 h-4 bg-white rounded-full shadow transition-transform ${refreshToken ? 'translate-x-4' : ''}`} />
          </div>
          <span className="text-xs text-gray-700">Enable Refresh Token Support</span>
          <span className="text-gray-400 text-xs" title="When enabled, a refresh token will be issued alongside the access token">ⓘ</span>
        </label>

        {refreshToken && (
          <div className="flex items-center gap-3 mt-3">
            <span className="text-xs font-medium text-gray-700">Absolute Token Lifetime</span>
            <div className="flex items-center border border-border rounded-md overflow-visible">
              <input type="number" defaultValue={1} className="w-16 px-3 py-2 text-sm focus:outline-none min-w-0" />
              <select
                value={absoluteLifetimeUnit}
                onChange={(e) => setAbsoluteLifetimeUnit(e.target.value)}
                className="px-3 py-2 text-sm text-gray-600 bg-gray-50 border-l border-border focus:outline-none"
              >
                <option value="hour">Hour</option>
                <option value="day">Day</option>
                <option value="week">Week</option>
                <option value="month">Month</option>
              </select>
            </div>
          </div>
        )}
      </div>

      <div className="border border-border rounded-md p-4">
        <h4 className="text-xs font-semibold text-gray-800 mb-3">Custom Claims</h4>
        {claims.length > 0 && (
          <div className="space-y-2 mb-3">
            {claims.map((claim, idx) => (
              <div key={idx} className="flex items-center gap-2">
                <input
                  type="text"
                  placeholder="Claim Name"
                  value={claim.name}
                  onChange={(e) => updateClaim(idx, 'name', e.target.value)}
                  className="flex-1 px-3 py-2 border border-border rounded-md text-xs focus:outline-none focus:ring-1 focus:ring-gray-300"
                />
                <div className="flex-1 flex items-center border border-border rounded-md overflow-hidden">
                  <input
                    type="text"
                    placeholder="Value"
                    value={claim.value}
                    onChange={(e) => updateClaim(idx, 'value', e.target.value)}
                    className="flex-1 px-3 py-2 text-xs focus:outline-none"
                  />
                  <select
                    value={claim.valueType}
                    onChange={(e) => updateClaim(idx, 'valueType', e.target.value)}
                    className="px-2 py-2 text-xs text-gray-600 bg-gray-50 border-l border-border focus:outline-none"
                  >
                    <option value="literal">Literal</option>
                    <option value="dynamic">Dynamic</option>
                  </select>
                </div>
                <button onClick={() => removeClaim(idx)} className="text-orange-400 hover:text-red-500 text-sm">×</button>
              </div>
            ))}
          </div>
        )}
        <button onClick={addClaim} className="text-xs text-orange-500 hover:text-orange-600 font-medium">
          + New Claim
        </button>
      </div>
    </div>
  )
}


function VaultClientTokenConfig() {
  const [subjectType, setSubjectType] = useState('literal')
  const [lifetimeUnit, setLifetimeUnit] = useState('minutes')
  const [forwardingConfig, setForwardingConfig] = useState('no_forwarding')
  const [privateNetwork, setPrivateNetwork] = useState(false)
  const [tls, setTls] = useState(false)
  const [claims, setClaims] = useState<{ name: string; value: string; valueType: string }[]>([{ name: '', value: '', valueType: 'literal' }])

  function addClaim() {
    setClaims([...claims, { name: '', value: '', valueType: 'literal' }])
  }

  function updateClaim(idx: number, field: string, val: string) {
    setClaims(claims.map((c, i) => i === idx ? { ...c, [field]: val } : c))
  }

  function removeClaim(idx: number) {
    if (claims.length > 1) {
      setClaims(claims.filter((_, i) => i !== idx))
    }
  }

  return (
    <div className="space-y-5">
      {/* JSON Web Token (JWT) section */}
      <div>
        <h4 className="text-xs font-semibold text-gray-800 mb-3">JSON Web Token (JWT)</h4>
        <div className="space-y-3">
          <div>
            <label className="block text-[11px] text-gray-500 mb-1">Subject</label>
            <div className="flex items-center border border-border rounded-md overflow-hidden">
              <input type="text" placeholder="Subject" className="flex-1 px-3 py-2 text-sm focus:outline-none" />
              <select
                value={subjectType}
                onChange={(e) => setSubjectType(e.target.value)}
                className="px-3 py-2 text-sm text-gray-600 bg-gray-50 border-l border-border focus:outline-none"
              >
                <option value="literal">Literal</option>
                <option value="dynamic">Dynamic</option>
              </select>
            </div>
          </div>

          <div className="flex items-start gap-3">
            <div className="flex-1">
              <ReadOnlyField label="Issuer" value="https://<tenant>.id.ap-south-1.dipolehq.com" />
            </div>
            <div className="w-40">
              <label className="block text-[11px] text-gray-500 mb-1">Lifetime</label>
              <div className="flex items-center border border-border rounded-md overflow-visible">
                <input type="number" defaultValue={1} className="w-14 px-3 py-2 text-sm focus:outline-none min-w-0" />
                <select
                  value={lifetimeUnit}
                  onChange={(e) => setLifetimeUnit(e.target.value)}
                  className="px-2 py-2 text-sm text-gray-600 bg-gray-50 border-l border-border focus:outline-none"
                >
                  <option value="minutes">Minutes</option>
                  <option value="hours">Hours</option>
                  <option value="days">Days</option>
                </select>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Custom Claims */}
      <div className="border border-border rounded-md p-4">
        <h4 className="text-xs font-semibold text-gray-800 mb-3">Custom Claims</h4>
        <div className="space-y-2 mb-3">
          {claims.map((claim, idx) => (
            <div key={idx} className="flex items-center gap-2">
              <input
                type="text"
                placeholder="Claim Name"
                value={claim.name}
                onChange={(e) => updateClaim(idx, 'name', e.target.value)}
                className="flex-1 px-3 py-2 border border-border rounded-md text-xs focus:outline-none focus:ring-1 focus:ring-gray-300"
              />
              <div className="flex-1 flex items-center border border-border rounded-md overflow-hidden">
                <input
                  type="text"
                  placeholder="Value"
                  value={claim.value}
                  onChange={(e) => updateClaim(idx, 'value', e.target.value)}
                  className="flex-1 px-3 py-2 text-xs focus:outline-none"
                />
                <select
                  value={claim.valueType}
                  onChange={(e) => updateClaim(idx, 'valueType', e.target.value)}
                  className="px-2 py-2 text-xs text-gray-600 bg-gray-50 border-l border-border focus:outline-none"
                >
                  <option value="literal">Literal</option>
                  <option value="dynamic">Dynamic</option>
                </select>
              </div>
              <button onClick={() => removeClaim(idx)} className="text-orange-400 hover:text-red-500 text-sm">×</button>
            </div>
          ))}
        </div>
        <button onClick={addClaim} className="text-xs text-orange-500 hover:text-orange-600 font-medium">
          + New Claim
        </button>
      </div>

      {/* Vault Authentication */}
      <div className="border border-border rounded-md p-4">
        <h4 className="text-xs font-semibold text-gray-800 mb-3">Vault Authentication</h4>
        <div className="space-y-3">
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-[11px] text-gray-500 mb-1">Host</label>
              <input type="text" placeholder="Host" className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-gray-300" />
            </div>
            <div>
              <label className="block text-[11px] text-gray-500 mb-1">Port</label>
              <div className="flex items-center border border-border rounded-md overflow-hidden">
                <input type="number" defaultValue={8200} className="flex-1 px-3 py-2 text-sm focus:outline-none" />
                <label className="flex items-center gap-1.5 px-3 border-l border-border cursor-pointer">
                  <input type="checkbox" checked={tls} onChange={(e) => setTls(e.target.checked)} className="w-3.5 h-3.5 rounded border-gray-300" />
                  <span className="text-xs text-gray-600">TLS</span>
                </label>
              </div>
            </div>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <Field label="Authentication Path" placeholder="Authentication Path" />
            <Field label="Role" placeholder="Role" />
          </div>

          <Field label="Namespace" placeholder="Namespace" />

          <div className="flex items-center gap-4">
            <div className="flex-1">
              <label className="block text-[11px] text-gray-500 mb-1">Forwarding Configuration</label>
              <Dropdown
                options={[
                  { value: 'no_forwarding', label: 'No Forwarding (default)' },
                  { value: 'unconditional', label: 'Unconditional Forwarding' },
                  { value: 'conditional', label: 'Conditional Forwarding' },
                ]}
                value={forwardingConfig}
                onChange={setForwardingConfig}
                placeholder="No Forwarding (default)"
              />
            </div>
            <label className="flex items-center gap-2 cursor-pointer mt-4">
              <input
                type="checkbox"
                checked={privateNetwork}
                onChange={(e) => setPrivateNetwork(e.target.checked)}
                className="w-4 h-4 rounded border-gray-300 text-orange-500 focus:ring-orange-200"
              />
              <div>
                <span className="text-xs text-gray-700">Private Network Access</span>
                <p className="text-[10px] text-gray-400">Requires Dipole AgentProxy 1.20.2629 or greater</p>
              </div>
            </label>
          </div>
        </div>
      </div>
    </div>
  )
}


function JwtSvidConfig() {
  const [subjectType, setSubjectType] = useState('literal')
  const [audienceType, setAudienceType] = useState('literal')
  const [signingAlgorithm, setSigningAlgorithm] = useState('')
  const [lifetimeUnit, setLifetimeUnit] = useState('minutes')
  const [claims, setClaims] = useState<{ name: string; value: string; valueType: string }[]>([{ name: '', value: '', valueType: 'literal' }])

  function addClaim() {
    setClaims([...claims, { name: '', value: '', valueType: 'literal' }])
  }

  function updateClaim(idx: number, field: string, val: string) {
    setClaims(claims.map((c, i) => i === idx ? { ...c, [field]: val } : c))
  }

  function removeClaim(idx: number) {
    if (claims.length > 1) {
      setClaims(claims.filter((_, i) => i !== idx))
    }
  }

  return (
    <div className="space-y-4">
      <div>
        <label className="block text-[11px] text-gray-500 mb-1">Subject</label>
        <div className="flex items-center border border-border rounded-md overflow-hidden">
          <input type="text" placeholder="Subject" className="flex-1 px-3 py-2 text-sm focus:outline-none" />
          <select
            value={subjectType}
            onChange={(e) => setSubjectType(e.target.value)}
            className="px-3 py-2 text-sm text-gray-600 bg-gray-50 border-l border-border focus:outline-none"
          >
            <option value="literal">Literal</option>
            <option value="dynamic">Dynamic</option>
          </select>
        </div>
      </div>

      <div className="flex items-start gap-3">
        <div className="flex-1">
          <ReadOnlyField label="Issuer" value="https://<tenant>.id.ap-south-1.dipolehq.com" />
        </div>
        <div className="w-40">
          <label className="block text-[11px] text-gray-500 mb-1">Lifetime</label>
          <div className="flex items-center border border-border rounded-md overflow-visible">
            <input type="number" defaultValue={15} className="w-14 px-3 py-2 text-sm focus:outline-none min-w-0" />
            <select
              value={lifetimeUnit}
              onChange={(e) => setLifetimeUnit(e.target.value)}
              className="px-2 py-2 text-sm text-gray-600 bg-gray-50 border-l border-border focus:outline-none"
            >
              <option value="minutes">Minutes</option>
              <option value="hours">Hours</option>
              <option value="days">Days</option>
            </select>
          </div>
        </div>
      </div>

      <div>
        <label className="block text-[11px] text-gray-500 mb-1">Signing Algorithm Type</label>
        <Dropdown
          options={[
            { value: 'rs256', label: 'RSASSA-PKCS1-v1_5 using SHA-256' },
            { value: 'es256', label: 'ECDSA using P-256 and SHA-256' },
          ]}
          value={signingAlgorithm}
          onChange={setSigningAlgorithm}
          placeholder="Signing Algorithm Type"
        />
      </div>

      <div>
        <label className="block text-[11px] text-gray-500 mb-1">Audience</label>
        <div className="flex items-center border border-border rounded-md overflow-hidden">
          <input type="text" placeholder="Audience" className="flex-1 px-3 py-2 text-sm focus:outline-none" />
          <select
            value={audienceType}
            onChange={(e) => setAudienceType(e.target.value)}
            className="px-3 py-2 text-sm text-gray-600 bg-gray-50 border-l border-border focus:outline-none"
          >
            <option value="literal">Literal</option>
            <option value="dynamic">Dynamic</option>
          </select>
        </div>
      </div>

      <div className="border border-border rounded-md p-4">
        <h4 className="text-xs font-semibold text-gray-800 mb-3">Custom Claims</h4>
        <div className="space-y-2 mb-3">
          {claims.map((claim, idx) => (
            <div key={idx} className="flex items-center gap-2">
              <input
                type="text"
                placeholder="Claim Name"
                value={claim.name}
                onChange={(e) => updateClaim(idx, 'name', e.target.value)}
                className="flex-1 px-3 py-2 border border-border rounded-md text-xs focus:outline-none focus:ring-1 focus:ring-gray-300"
              />
              <div className="flex-1 flex items-center border border-border rounded-md overflow-hidden">
                <input
                  type="text"
                  placeholder="Value"
                  value={claim.value}
                  onChange={(e) => updateClaim(idx, 'value', e.target.value)}
                  className="flex-1 px-3 py-2 text-xs focus:outline-none"
                />
                <select
                  value={claim.valueType}
                  onChange={(e) => updateClaim(idx, 'valueType', e.target.value)}
                  className="px-2 py-2 text-xs text-gray-600 bg-gray-50 border-l border-border focus:outline-none"
                >
                  <option value="literal">Literal</option>
                  <option value="dynamic">Dynamic</option>
                </select>
              </div>
              <button onClick={() => removeClaim(idx)} className="text-orange-400 hover:text-red-500 text-sm">×</button>
            </div>
          ))}
        </div>
        <button onClick={addClaim} className="text-xs text-orange-500 hover:text-orange-600 font-medium">
          + New Claim
        </button>
      </div>
    </div>
  )
}


function X509SvidConfig() {
  const [subjectType, setSubjectType] = useState('literal')
  const [spiffeIdType, setSpiffeIdType] = useState('literal')
  const [lifetimeUnit, setLifetimeUnit] = useState('minutes')
  const [keyUsage, setKeyUsage] = useState('digital_signature')
  const [extKeyUsage, setExtKeyUsage] = useState('id_kp_clientauth')

  return (
    <div className="space-y-4">
      <div>
        <label className="block text-[11px] text-gray-500 mb-1">Subject</label>
        <div className="flex items-center border border-border rounded-md overflow-hidden">
          <input type="text" placeholder="Subject" className="flex-1 px-3 py-2 text-sm focus:outline-none" />
          <select
            value={subjectType}
            onChange={(e) => setSubjectType(e.target.value)}
            className="px-3 py-2 text-sm text-gray-600 bg-gray-50 border-l border-border focus:outline-none"
          >
            <option value="literal">Literal</option>
            <option value="dynamic">Dynamic</option>
          </select>
        </div>
      </div>

      <div>
        <label className="block text-[11px] text-gray-500 mb-1">Spiffe ID</label>
        <div className="flex items-center border border-border rounded-md overflow-hidden">
          <input type="text" placeholder="Spiffe ID" className="flex-1 px-3 py-2 text-sm focus:outline-none" />
          <select
            value={spiffeIdType}
            onChange={(e) => setSpiffeIdType(e.target.value)}
            className="px-3 py-2 text-sm text-gray-600 bg-gray-50 border-l border-border focus:outline-none"
          >
            <option value="literal">Literal</option>
            <option value="dynamic">Dynamic</option>
          </select>
        </div>
      </div>

      <div>
        <label className="block text-[11px] text-gray-500 mb-1">Lifetime</label>
        <div className="flex items-center border border-border rounded-md overflow-visible w-48">
          <input type="number" defaultValue={15} className="flex-1 px-3 py-2 text-sm focus:outline-none min-w-0" />
          <select
            value={lifetimeUnit}
            onChange={(e) => setLifetimeUnit(e.target.value)}
            className="px-2 py-2 text-sm text-gray-600 bg-gray-50 border-l border-border focus:outline-none"
          >
            <option value="minutes">Minutes</option>
            <option value="hours">Hours</option>
            <option value="days">Days</option>
          </select>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-3">
        <div>
          <label className="block text-[11px] text-gray-500 mb-1">Key Usage</label>
          <Dropdown
            options={[
              { value: 'digital_signature', label: 'Digital Signature' },
            ]}
            value={keyUsage}
            onChange={setKeyUsage}
            placeholder="Key Usage"
          />
        </div>
        <div>
          <label className="block text-[11px] text-gray-500 mb-1">Extended Key Usage</label>
          <Dropdown
            options={[
              { value: 'id_kp_serverauth', label: 'id-kp-serverAuth' },
              { value: 'id_kp_clientauth', label: 'id-kp-clientAuth' },
            ]}
            value={extKeyUsage}
            onChange={setExtKeyUsage}
            placeholder="Extended Key Usage"
          />
        </div>
      </div>
    </div>
  )
}

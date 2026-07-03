import { useState } from 'react'
import { ExistingCredentialProviderSelector } from '@/features/policies/components/tabs/credential-provider/ExistingCredentialProviderSelector'
import { CredentialProviderForm } from '@/shared/components/forms/CredentialProviderForm'
import { buildCredentialProviderValues } from '@/data/policies/policyFormMappers'
import type { CredentialProviderFormValues } from '@/shared/components/forms/types'

interface CredentialProviderTabProps {
  selectedId: string
  onSelect: (id: string) => void
  values: CredentialProviderFormValues
  onChange: (values: CredentialProviderFormValues) => void
  disabled?: boolean
}

export function CredentialProviderTab({
  selectedId,
  onSelect,
  values,
  onChange,
  disabled = false,
}: CredentialProviderTabProps) {
  const [mode, setMode] = useState<'new' | 'existing'>('new')

  function handleSelectExisting(id: string) {
    onSelect(id)
    onChange(buildCredentialProviderValues(id))
    setMode('new')
  }

  return (
    <div className="space-y-6">
      {!disabled && (
        <div className="flex items-center gap-4 border-b border-border pb-3">
          <button
            type="button"
            onClick={() => setMode('new')}
            className={`text-xs font-medium transition-colors ${mode === 'new' ? 'text-orange-600' : 'text-gray-500 hover:text-gray-700'}`}
          >
            Add New
          </button>
          <button
            type="button"
            onClick={() => setMode('existing')}
            className={`text-xs font-medium transition-colors ${mode === 'existing' ? 'text-orange-600' : 'text-gray-500 hover:text-gray-700'}`}
          >
            Select Existing
          </button>
        </div>
      )}

      {!disabled && mode === 'existing' ? (
        <ExistingCredentialProviderSelector
          selectedId={selectedId}
          onSelect={handleSelectExisting}
          onCancel={() => setMode('new')}
        />
      ) : (
        <CredentialProviderForm values={values} onChange={onChange} disabled={disabled} />
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Credential Type Config — fully controlled, renders type-specific fields
// ---------------------------------------------------------------------------

interface CredentialTypeConfigProps {
  type: string
  values: CredentialProviderFormValues
  onChange: (values: CredentialProviderFormValues) => void
}

export function CredentialTypeConfig({ type, values, onChange }: CredentialTypeConfigProps) {
  function update(partial: Partial<CredentialProviderFormValues>) {
    onChange({ ...values, ...partial })
  }

  switch (type) {
    case 'api_key':
      return (
        <div className="space-y-3">
          <div>
            <label className="block text-[11px] text-gray-500 mb-1">API Key</label>
            <input
              type="password"
              placeholder="Enter API key"
              value={values.apiKey}
              onChange={(e) => update({ apiKey: e.target.value })}
              className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-gray-300"
            />
          </div>
        </div>
      )

    case 'username_password':
      return (
        <div className="space-y-3">
          <div>
            <label className="block text-[11px] text-gray-500 mb-1">Username</label>
            <input
              type="text"
              placeholder="Username"
              value={values.username}
              onChange={(e) => update({ username: e.target.value })}
              className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-gray-300"
            />
          </div>
          <div>
            <label className="block text-[11px] text-gray-500 mb-1">Password</label>
            <input
              type="password"
              placeholder="Password"
              value={values.password}
              onChange={(e) => update({ password: e.target.value })}
              className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-gray-300"
            />
          </div>
        </div>
      )

    case 'aws_sts_federation':
      return (
        <div className="space-y-4">
          <ReadOnlyField label="OIDC Issuer URL" value="https://<tenant>.id.ap-south-1.dipolehq.com" />
          <div>
            <label className="block text-[11px] text-gray-500 mb-1">AWS IAM Role ARN</label>
            <input
              type="text"
              placeholder="arn:aws:iam::123456789012:role/example"
              value={values.awsIamRoleArn}
              onChange={(e) => update({ awsIamRoleArn: e.target.value })}
              className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-gray-300"
            />
          </div>
          <div>
            <label className="block text-[11px] text-gray-500 mb-1">Audience</label>
            <input
              type="text"
              placeholder="sts.amazonaws.com"
              value={values.audience}
              onChange={(e) => update({ audience: e.target.value })}
              className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-gray-300"
            />
          </div>
          <div>
            <label className="block text-[11px] text-gray-500 mb-1">Lifetime</label>
            <div className="flex items-center border border-border rounded-md overflow-visible w-64">
              <input
                type="number"
                value={values.lifetimeSeconds}
                onChange={(e) => update({ lifetimeSeconds: Number(e.target.value) })}
                className="flex-1 px-3 py-2 text-sm focus:outline-none min-w-0"
              />
              <span className="px-4 py-2 text-sm text-gray-500 bg-gray-50 border-l border-border whitespace-nowrap">Seconds</span>
            </div>
          </div>
        </div>
      )

    case 'oauth_client_credentials':
      return (
        <div className="space-y-4">
          <div>
            <label className="block text-[11px] text-gray-500 mb-1">Token Endpoint URL</label>
            <input
              type="text"
              placeholder="https://provider.com/oauth/token"
              value={values.tokenEndpointUrl}
              onChange={(e) => update({ tokenEndpointUrl: e.target.value })}
              className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-gray-300"
            />
          </div>
          <div>
            <label className="block text-[11px] text-gray-500 mb-1">Client ID</label>
            <input
              type="text"
              placeholder="Client ID"
              value={values.clientId}
              onChange={(e) => update({ clientId: e.target.value })}
              className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-gray-300"
            />
          </div>
          <div>
            <label className="block text-[11px] text-gray-500 mb-1">Client Secret</label>
            <input
              type="password"
              placeholder="Client Secret"
              value={values.clientSecret}
              onChange={(e) => update({ clientSecret: e.target.value })}
              className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-gray-300"
            />
          </div>
          <div>
            <label className="block text-[11px] text-gray-500 mb-1">Scopes (Optional)</label>
            <input
              type="text"
              placeholder="read write"
              value={values.scopes}
              onChange={(e) => update({ scopes: e.target.value })}
              className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-gray-300"
            />
          </div>
          <div>
            <label className="block text-[11px] text-gray-500 mb-1">Lifetime</label>
            <div className="flex items-center border border-border rounded-md overflow-visible w-64">
              <input
                type="number"
                value={values.lifetimeSeconds}
                onChange={(e) => update({ lifetimeSeconds: Number(e.target.value) })}
                className="flex-1 px-3 py-2 text-sm focus:outline-none min-w-0"
              />
              <span className="px-4 py-2 text-sm text-gray-500 bg-gray-50 border-l border-border whitespace-nowrap">Seconds</span>
            </div>
          </div>
        </div>
      )

    default:
      return (
        <p className="text-xs text-gray-400 italic">
          Configuration fields for this credential type are not yet implemented.
        </p>
      )
  }
}

function ReadOnlyField({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <label className="block text-[11px] text-gray-500 mb-1">{label}</label>
      <div className="flex items-center border border-border rounded-md overflow-hidden bg-gray-50">
        <input type="text" readOnly value={value} className="flex-1 px-3 py-2 text-sm text-gray-500 bg-gray-50 focus:outline-none" />
        <button
          type="button"
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

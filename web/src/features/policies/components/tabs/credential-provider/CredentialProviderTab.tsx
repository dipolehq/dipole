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

import { FloatingInput } from '@/shared/components/FloatingInput'

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
        <div className="space-y-4">
          <FloatingInput
            label="API Key"
            value={values.apiKey}
            onChange={(v) => update({ apiKey: v })}
            type="password"
          />
        </div>
      )

    case 'username_password':
      return (
        <div className="space-y-4">
          <FloatingInput
            label="Username"
            value={values.username}
            onChange={(v) => update({ username: v })}
          />
          <FloatingInput
            label="Password"
            value={values.password}
            onChange={(v) => update({ password: v })}
            type="password"
          />
        </div>
      )

    case 'aws_sts_federation':
      return (
        <div className="space-y-4">
          <FloatingInput
            label="OIDC Issuer URL"
            value="https://<tenant>.id.ap-south-1.dipolehq.com"
            disabled
          />
          <FloatingInput
            label="AWS IAM Role ARN"
            value={values.awsIamRoleArn}
            onChange={(v) => update({ awsIamRoleArn: v })}
          />
          <FloatingInput
            label="Audience"
            value={values.audience}
            onChange={(v) => update({ audience: v })}
          />
          <FloatingInput
            label="Lifetime (Seconds)"
            value={String(values.lifetimeSeconds)}
            onChange={(v) => update({ lifetimeSeconds: Number(v) || 0 })}
            type="number"
          />
        </div>
      )

    case 'oauth_client_credentials':
      return (
        <div className="space-y-4">
          <FloatingInput
            label="Token Endpoint URL"
            value={values.tokenEndpointUrl}
            onChange={(v) => update({ tokenEndpointUrl: v })}
          />
          <FloatingInput
            label="Client ID"
            value={values.clientId}
            onChange={(v) => update({ clientId: v })}
          />
          <FloatingInput
            label="Client Secret"
            value={values.clientSecret}
            onChange={(v) => update({ clientSecret: v })}
            type="password"
          />
          <FloatingInput
            label="Scopes (Optional)"
            value={values.scopes}
            onChange={(v) => update({ scopes: v })}
          />
          <FloatingInput
            label="Lifetime (Seconds)"
            value={String(values.lifetimeSeconds)}
            onChange={(v) => update({ lifetimeSeconds: Number(v) || 0 })}
            type="number"
          />
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

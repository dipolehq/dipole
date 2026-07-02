import { Dropdown } from '@/shared/components/Dropdown'
import { TagsField } from '@/shared/components/TagsField'
import { CredentialTypeConfig } from '@/features/policies/components/tabs/credential-provider/CredentialProviderTab'
import { CREDENTIAL_TYPES } from '@/data/policies/geoData'
import type { CredentialProviderFormValues } from '@/shared/components/forms/types'

interface CredentialProviderFormProps {
  values: CredentialProviderFormValues
  onChange: (values: CredentialProviderFormValues) => void
  disabled?: boolean
}

export function CredentialProviderForm({ values, onChange, disabled = false }: CredentialProviderFormProps) {
  function update(partial: Partial<CredentialProviderFormValues>) {
    onChange({ ...values, ...partial })
  }

  const disabledClass = disabled ? 'opacity-60 pointer-events-none' : ''

  return (
    <fieldset disabled={disabled} className={disabledClass}>
      <div className="space-y-5">
        <div>
          <label className="block text-xs font-medium text-gray-700 mb-1.5">Name</label>
          <input
            type="text"
            placeholder="Name"
            value={values.name}
            onChange={(e) => update({ name: e.target.value })}
            className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-gray-300"
          />
        </div>

        <div>
          <label className="block text-xs font-medium text-gray-700 mb-1.5">Description</label>
          <textarea
            placeholder="Description (Optional)"
            value={values.description}
            onChange={(e) => update({ description: e.target.value })}
            rows={2}
            className="w-full px-3 py-2 border border-border rounded-md text-sm resize-y focus:outline-none focus:ring-1 focus:ring-gray-300"
          />
        </div>

        <div>
          <h4 className="text-xs font-semibold text-gray-800 mb-2">Credential Type</h4>
          <Dropdown
            options={CREDENTIAL_TYPES}
            value={values.credentialType}
            onChange={(v) => update({ credentialType: v })}
            placeholder=""
          />
        </div>

        {values.credentialType && (
          <CredentialTypeConfig
            type={values.credentialType}
            values={values}
            onChange={onChange}
          />
        )}

        <div>
          <label className="block text-xs font-medium text-gray-700 mb-1.5">Tags</label>
          <TagsField />
        </div>
      </div>
    </fieldset>
  )
}

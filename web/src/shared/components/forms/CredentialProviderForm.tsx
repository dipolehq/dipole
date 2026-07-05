import { type ReactNode } from 'react'
import { TagsField } from '@/shared/components/TagsField'
import { FloatingInput, FloatingTextarea, FloatingDropdown } from '@/shared/components/FloatingInput'
import { CredentialTypeConfig } from '@/features/policies/components/tabs/credential-provider/CredentialProviderTab'
import { CREDENTIAL_TYPES } from '@/data/policies/geoData'
import type { CredentialProviderFormValues } from '@/shared/components/forms/types'

interface CredentialProviderFormProps {
  values: CredentialProviderFormValues
  onChange: (values: CredentialProviderFormValues) => void
  disabled?: boolean
  statusSlot?: ReactNode
}

export function CredentialProviderForm({ values, onChange, disabled = false, statusSlot }: CredentialProviderFormProps) {
  function update(partial: Partial<CredentialProviderFormValues>) {
    onChange({ ...values, ...partial })
  }

  const disabledClass = disabled ? 'opacity-60 pointer-events-none' : ''

  return (
    <fieldset disabled={disabled} className={disabledClass}>
      <div className="space-y-5">
        <div className="border border-border rounded-lg p-5">
          <h4 className="text-sm font-semibold text-gray-800 mb-4">Details</h4>
          <div className="space-y-4">
            {statusSlot}
            <FloatingInput
              label="Name"
              value={values.name}
              onChange={(v) => update({ name: v })}
              disabled={disabled}
              required
            />
            <FloatingTextarea
              label="Description (Optional)"
              value={values.description}
              onChange={(v) => update({ description: v })}
              disabled={disabled}
              rows={2}
            />
          </div>
        </div>

        <div className="border border-border rounded-lg p-5">
          <h4 className="text-sm font-semibold text-gray-800 mb-4">Credential Configuration</h4>
          <div className="space-y-4">
            <FloatingDropdown
              label="Credential Type"
              options={CREDENTIAL_TYPES}
              value={values.credentialType}
              onChange={(v) => update({ credentialType: v })}
              disabled={disabled}
            />

            {values.credentialType && (
              <CredentialTypeConfig
                type={values.credentialType}
                values={values}
                onChange={onChange}
              />
            )}
          </div>
        </div>

        <div className="border border-border rounded-lg p-5">
          <h4 className="text-sm font-semibold text-gray-800 mb-4">Tags</h4>
          <TagsField />
        </div>
      </div>
    </fieldset>
  )
}

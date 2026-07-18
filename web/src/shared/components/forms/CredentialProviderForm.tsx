import { type ReactNode } from 'react'
import { TagsField } from '@/shared/components/TagsField'
import {
  FloatingInput,
  FloatingTextarea,
  FloatingDropdown,
} from '@/shared/components/FloatingInput'
import { CredentialTypeConfig } from '@/features/policies/components/tabs/credential-provider/CredentialProviderTab'
import { CREDENTIAL_TYPES } from '@/data/policies/geoData'
import { Dropdown } from '@/shared/components/Dropdown'
import type { CredentialProviderFormValues } from '@/shared/components/forms/types'

interface CredentialProviderFormProps {
  values: CredentialProviderFormValues
  onChange: (values: CredentialProviderFormValues) => void
  disabled?: boolean
  statusSlot?: ReactNode
}

export function CredentialProviderForm({
  values,
  onChange,
  disabled = false,
  statusSlot,
}: CredentialProviderFormProps) {
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

        {['oidc_id_token', 'jwt_svid', 'vault_client_token'].includes(values.credentialType) && (
          <div className="border border-border rounded-lg p-5">
            <h4 className="text-sm font-semibold text-gray-800 mb-4">Custom Claims</h4>
            <div className="space-y-2">
              {values.customClaims.map((claim, idx) => (
                <div key={idx} className="flex items-center gap-2">
                  <div className="flex-1">
                    <FloatingInput
                      label="Claim Name"
                      value={claim.name}
                      onChange={(v) =>
                        update({
                          customClaims: values.customClaims.map((c, i) =>
                            i === idx ? { ...c, name: v } : c,
                          ),
                        })
                      }
                      disabled={disabled}
                    />
                  </div>
                  <div className="flex-1">
                    <FloatingInput
                      label="Value"
                      value={claim.value}
                      onChange={(v) =>
                        update({
                          customClaims: values.customClaims.map((c, i) =>
                            i === idx ? { ...c, value: v } : c,
                          ),
                        })
                      }
                      disabled={disabled}
                    />
                  </div>
                  <div className="w-28 shrink-0">
                    <Dropdown
                      options={[
                        { value: 'literal', label: 'Literal' },
                        { value: 'dynamic', label: 'Dynamic' },
                      ]}
                      value={claim.valueType || 'literal'}
                      onChange={(v) =>
                        update({
                          customClaims: values.customClaims.map((c, i) =>
                            i === idx ? { ...c, valueType: v } : c,
                          ),
                        })
                      }
                    />
                  </div>
                  <button
                    type="button"
                    onClick={() =>
                      update({ customClaims: values.customClaims.filter((_, i) => i !== idx) })
                    }
                    className="text-orange-400 hover:text-red-500 text-sm"
                  >
                    ×
                  </button>
                </div>
              ))}
            </div>
            <button
              type="button"
              onClick={() =>
                update({
                  customClaims: [
                    ...values.customClaims,
                    { name: '', value: '', valueType: 'literal' },
                  ],
                })
              }
              className="mt-3 text-xs text-orange-500 hover:text-orange-600 font-medium"
            >
              + New Claim
            </button>
          </div>
        )}

        <div className="border border-border rounded-lg p-5">
          <h4 className="text-sm font-semibold text-gray-800 mb-4">Tags</h4>
          <TagsField
            value={values.tags}
            onChange={(tags) => update({ tags })}
            disabled={disabled}
          />
        </div>
      </div>
    </fieldset>
  )
}

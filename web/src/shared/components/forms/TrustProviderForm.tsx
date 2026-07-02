import { Dropdown } from '@/shared/components/Dropdown'
import { TagsField } from '@/shared/components/TagsField'
import { TRUST_PROVIDER_TYPES } from '@/data/policies/config'
import { TrustProviderConfig } from '@/features/policies/components/tabs/trust-provider/TrustProviderConfig'
import type { TrustProviderFormValues } from '@/shared/components/forms/types'

interface TrustProviderFormProps {
  values: TrustProviderFormValues
  onChange: (values: TrustProviderFormValues) => void
  disabled?: boolean
}

export function TrustProviderForm({ values, onChange, disabled = false }: TrustProviderFormProps) {
  function update(partial: Partial<TrustProviderFormValues>) {
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
          <h4 className="text-xs font-semibold text-gray-800 uppercase tracking-wide mb-2">Trust Provider</h4>
          <Dropdown
            options={TRUST_PROVIDER_TYPES.map((t) => ({ value: t.value, label: t.label }))}
            value={values.trustProviderType}
            onChange={(v) => update({ trustProviderType: v })}
            placeholder=""
          />
        </div>

        {values.trustProviderType && (
          <TrustProviderConfig
            type={values.trustProviderType}
            matchRules={values.matchRules}
            onMatchRulesChange={(rules) => update({ matchRules: rules })}
            certificate={values.certificate}
            onCertificateChange={(v) => update({ certificate: v })}
            thumbprint={values.thumbprint}
            onThumbprintChange={(v) => update({ thumbprint: v })}
            subject={values.subject}
            onSubjectChange={(v) => update({ subject: v })}
            expiresAt={values.expiresAt}
            onExpiresAtChange={(v) => update({ expiresAt: v })}
            oidcEndpoint={values.oidcEndpoint}
            onOidcEndpointChange={(v) => update({ oidcEndpoint: v })}
            attestationMethod={values.attestationMethod}
            onAttestationMethodChange={(v) => update({ attestationMethod: v })}
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

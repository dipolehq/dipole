import { type ReactNode } from 'react'
import { TagsField } from '@/shared/components/TagsField'
import {
  FloatingInput,
  FloatingTextarea,
  FloatingDropdown,
} from '@/shared/components/FloatingInput'
import { TRUST_PROVIDER_TYPES } from '@/shared/constants/policies'
import { TrustProviderConfig } from '@/features/policies/components/tabs/trust-provider/TrustProviderConfig'
import type { TrustProviderFormValues } from '@/shared/components/forms/types'

interface TrustProviderFormProps {
  values: TrustProviderFormValues
  onChange: (values: TrustProviderFormValues) => void
  disabled?: boolean
  statusSlot?: ReactNode
}

export function TrustProviderForm({
  values,
  onChange,
  disabled = false,
  statusSlot,
}: TrustProviderFormProps) {
  function update(partial: Partial<TrustProviderFormValues>) {
    onChange({ ...values, ...partial })
  }

  const disabledClass = disabled ? 'opacity-60 pointer-events-none' : ''

  return (
    <fieldset disabled={disabled} className={disabledClass}>
      <div className="space-y-6">
        <div>
          <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider mb-4">Details</h4>
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

        <div>
          <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider mb-4">Trust Provider Configuration</h4>
          <div className="space-y-4">
            <FloatingDropdown
              label="Trust Provider Type"
              options={TRUST_PROVIDER_TYPES.map((t) => ({ value: t.value, label: t.label }))}
              value={values.trustProviderType}
              onChange={(v) => update({ trustProviderType: v })}
              disabled={disabled}
            />

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
                samlMetadataUrl={values.samlMetadataUrl}
                onSamlMetadataUrlChange={(v) => update({ samlMetadataUrl: v })}
                samlMetadataXml={values.samlMetadataXml}
                onSamlMetadataXmlChange={(v) => update({ samlMetadataXml: v })}
                symmetricKey={values.symmetricKey}
                onSymmetricKeyChange={(v) => update({ symmetricKey: v })}
                jwks={values.jwks}
                onJwksChange={(v) => update({ jwks: v })}
                publicKey={values.publicKey}
                onPublicKeyChange={(v) => update({ publicKey: v })}
              />
            )}
          </div>
        </div>

        <div>
          <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider mb-4">Tags</h4>
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

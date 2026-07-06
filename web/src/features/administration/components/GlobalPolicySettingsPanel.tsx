import { useState } from 'react'
import { FloatingInput } from '@/shared/components/FloatingInput'

type EnforcementLevel = 'required' | 'recommended' | 'optional'

const ENFORCEMENT_LEVELS: { value: EnforcementLevel; label: string }[] = [
  { value: 'required', label: 'Required' },
  { value: 'recommended', label: 'Recommended' },
  { value: 'optional', label: 'Optional' },
]

export function GlobalPolicySettingsPanel() {
  // Access Policy Compliance
  const [apTrustProvider, setApTrustProvider] = useState<EnforcementLevel>('required')
  const [apAccessCondition, setApAccessCondition] = useState<EnforcementLevel>('recommended')

  // Agent Controller Compliance
  const [acTrustProvider, setAcTrustProvider] = useState<EnforcementLevel>('recommended')
  const [acTlsHostname, setAcTlsHostname] = useState<EnforcementLevel>('recommended')

  // Access Token Compliance
  const [tokenLifetime, setTokenLifetime] = useState('604800')

  function handleSave() {
    console.info('[global-policy-compliance] saved', {
      apTrustProvider, apAccessCondition, acTrustProvider, acTlsHostname, tokenLifetime,
    })
  }

  return (
    <div className="flex h-full">
      {/* Left 60% — settings */}
      <div className="w-[60%] px-6 py-6 overflow-y-auto">
        {/* Save row */}
        <div className="flex items-center justify-between border-b border-border pb-3 mb-6">
          <h3 className="text-sm font-semibold text-gray-900">Global Policy Compliance</h3>
          <button
            type="button"
            onClick={handleSave}
            className="px-4 py-1.5 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors"
          >
            Save
          </button>
        </div>

        <div className="space-y-5">
          {/* Access Policy Compliance */}
          <div className="border border-border rounded-lg p-5">
            <h4 className="text-sm font-semibold text-gray-800 mb-4">Access Policy Compliance</h4>
            <div className="space-y-3">
              <EnforcementRow label="Trust Providers" value={apTrustProvider} onChange={setApTrustProvider} />
              <EnforcementRow label="Access Conditions" value={apAccessCondition} onChange={setApAccessCondition} />
            </div>
          </div>

          {/* Agent Controller Compliance */}
          <div className="border border-border rounded-lg p-5">
            <h4 className="text-sm font-semibold text-gray-800 mb-4">Agent Controller Compliance</h4>
            <div className="space-y-3">
              <EnforcementRow label="Trust Providers" value={acTrustProvider} onChange={setAcTrustProvider} />
              <EnforcementRow label="Allowed TLS Hostname" value={acTlsHostname} onChange={setAcTlsHostname} />
            </div>
          </div>

          {/* Access Token Compliance */}
          <div className="border border-border rounded-lg p-5">
            <h4 className="text-sm font-semibold text-gray-800 mb-4">Access Token Compliance</h4>
            <div className="flex items-end gap-3">
              <div className="w-48">
                <FloatingInput
                  label="API Access Token Lifetime"
                  value={tokenLifetime}
                  onChange={setTokenLifetime}
                  type="number"
                />
              </div>
              <span className="text-sm text-gray-500 pb-2.5">Seconds</span>
            </div>
            <p className="text-[11px] text-gray-400 mt-2">Maximum allowed: 2,592,000 seconds (30 days)</p>
          </div>
        </div>
      </div>

      {/* Right 40% — help */}
      <div className="w-[40%] px-6 py-6 overflow-y-auto">
        <div className="border border-border rounded-lg p-5 sticky top-0">
          <p className="text-xs text-gray-700 leading-relaxed mb-3">
            <span className="font-semibold text-orange-600">Global Policy Compliance</span> defines organization-wide security standards for Access Policies and Agent Controllers.
          </p>
          <p className="text-xs text-gray-700 leading-relaxed mb-3">
            Each attribute can be set to <span className="font-semibold">Required</span> (must be present), <span className="font-semibold">Recommended</span> (flagged if missing), or <span className="font-semibold">Optional</span> (no enforcement).
          </p>
          <p className="text-xs text-gray-700 leading-relaxed">
            The <span className="font-semibold">Access Token Lifetime</span> sets the maximum validity period for API access tokens issued across the tenant.
          </p>
        </div>
      </div>
    </div>
  )
}

function EnforcementRow({
  label,
  value,
  onChange,
}: {
  label: string
  value: EnforcementLevel
  onChange: (v: EnforcementLevel) => void
}) {
  return (
    <div className="flex items-center justify-between gap-4">
      <span className="text-sm text-gray-700">{label}</span>
      <div className="flex items-center rounded-md border border-border overflow-hidden shrink-0">
        {ENFORCEMENT_LEVELS.map((level) => (
          <button
            key={level.value}
            type="button"
            onClick={() => onChange(level.value)}
            className={`px-3 py-1.5 text-xs font-medium transition-colors ${
              value === level.value
                ? 'bg-orange-500 text-white'
                : 'bg-white text-gray-600 hover:bg-gray-50'
            }`}
          >
            {level.label}
          </button>
        ))}
      </div>
    </div>
  )
}

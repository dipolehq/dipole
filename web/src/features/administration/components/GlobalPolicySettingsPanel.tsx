import { useState } from 'react'

type EnforcementLevel = 'required' | 'recommended' | 'optional'

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
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-sm font-semibold text-gray-900">Global Policy Compliance</h2>
        <button
          type="button"
          onClick={handleSave}
          className="px-4 py-1.5 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors"
        >
          Save
        </button>
      </div>

      {/* Access Policy Compliance */}
      <div className="mb-8">
        <h3 className="text-sm font-semibold text-orange-600 mb-1">Access Policy Compliance (Global)</h3>
        <div className="border-b border-border mb-4" />

        <div className="space-y-4">
          <EnforcementRow
            label="Trust Providers"
            value={apTrustProvider}
            onChange={setApTrustProvider}
          />
          <EnforcementRow
            label="Access Conditions"
            value={apAccessCondition}
            onChange={setApAccessCondition}
          />
        </div>
      </div>

      {/* Agent Controller Compliance */}
      <div className="mb-8">
        <h3 className="text-sm font-semibold text-orange-600 mb-1">Agent Controller Compliance (Global)</h3>
        <div className="border-b border-border mb-4" />

        <div className="space-y-4">
          <EnforcementRow
            label="Trust Providers"
            value={acTrustProvider}
            onChange={setAcTrustProvider}
          />
          <EnforcementRow
            label="Allowed TLS Hostname"
            value={acTlsHostname}
            onChange={setAcTlsHostname}
          />
        </div>
      </div>

      {/* Access Token Compliance */}
      <div>
        <h3 className="text-sm font-semibold text-orange-600 mb-1">Access Token Compliance (Global)</h3>
        <div className="border-b border-border mb-4" />

        <div className="flex items-center gap-4">
          <span className="text-sm font-medium text-gray-700 w-40">API Access Token</span>
          <div>
            <div className="flex items-center gap-1">
              <div>
                <input
                  type="number"
                  value={tokenLifetime}
                  onChange={(e) => setTokenLifetime(e.target.value)}
                  className="w-28 px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400"
                />
              </div>
              <span className="text-sm text-gray-600">Secs.</span>
            </div>
            <p className="text-[11px] text-gray-400 mt-1">Max 2,592,000</p>
          </div>
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
  name?: string
}) {
  return (
    <div className="flex items-center">
      <span className="text-sm font-medium text-gray-700 w-40">{label}</span>
      <div className="flex items-center gap-5">
        <label className="flex items-center gap-1.5 cursor-pointer">
          <input
            type="checkbox"
            checked={value === 'required'}
            onChange={() => onChange('required')}
            className="w-4 h-4 accent-orange-500 cursor-pointer"
          />
          <span className="text-sm text-gray-700">Required</span>
        </label>
        <label className="flex items-center gap-1.5 cursor-pointer">
          <input
            type="checkbox"
            checked={value === 'recommended'}
            onChange={() => onChange('recommended')}
            className="w-4 h-4 accent-orange-500 cursor-pointer"
          />
          <span className="text-sm text-gray-700">Recommended</span>
        </label>
        <label className="flex items-center gap-1.5 cursor-pointer">
          <input
            type="checkbox"
            checked={value === 'optional'}
            onChange={() => onChange('optional')}
            className="w-4 h-4 accent-orange-500 cursor-pointer"
          />
          <span className="text-sm text-gray-700">Optional</span>
        </label>
      </div>
    </div>
  )
}

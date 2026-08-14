import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '@/shared/lib/api-client'
import { FloatingInput } from '@/shared/components/FloatingInput'

type EnforcementLevel = 'required' | 'recommended' | 'optional'

const ENFORCEMENT_LEVELS: { value: EnforcementLevel; label: string }[] = [
  { value: 'required', label: 'Required' },
  { value: 'recommended', label: 'Recommended' },
  { value: 'optional', label: 'Optional' },
]

interface GlobalPolicyData {
  apTrustProvider: string
  apAccessCondition: string
  acTrustProvider: string
  acTlsHostname: string
  accessTokenLifetimeSeconds: number
}

export function GlobalPolicySettingsPanel() {
  const qc = useQueryClient()

  const { data } = useQuery<GlobalPolicyData>({
    queryKey: ['global-policy-compliance'],
    queryFn: () => apiClient.get<GlobalPolicyData>('/global-policy-compliance'),
  })

  if (!data) {
    return <div className="flex-1" />
  }

  return <GlobalPolicyForm data={data} queryClient={qc} />
}

function GlobalPolicyForm({ data, queryClient }: { data: GlobalPolicyData; queryClient: ReturnType<typeof useQueryClient> }) {
  const [apTrustProvider, setApTrustProvider] = useState<EnforcementLevel>(data.apTrustProvider as EnforcementLevel)
  const [apAccessCondition, setApAccessCondition] = useState<EnforcementLevel>(data.apAccessCondition as EnforcementLevel)
  const [acTrustProvider, setAcTrustProvider] = useState<EnforcementLevel>(data.acTrustProvider as EnforcementLevel)
  const [acTlsHostname, setAcTlsHostname] = useState<EnforcementLevel>(data.acTlsHostname as EnforcementLevel)
  const [tokenLifetime, setTokenLifetime] = useState(String(data.accessTokenLifetimeSeconds))
  const [isEditing, setIsEditing] = useState(false)

  const saveMutation = useMutation({
    mutationFn: () =>
      apiClient.put<GlobalPolicyData>('/global-policy-compliance', {
        apTrustProvider,
        apAccessCondition,
        acTrustProvider,
        acTlsHostname,
        accessTokenLifetimeSeconds: Number(tokenLifetime) || 604800,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['global-policy-compliance'] })
      setIsEditing(false)
    },
    onError: () => {
      setApTrustProvider(data.apTrustProvider as EnforcementLevel)
      setApAccessCondition(data.apAccessCondition as EnforcementLevel)
      setAcTrustProvider(data.acTrustProvider as EnforcementLevel)
      setAcTlsHostname(data.acTlsHostname as EnforcementLevel)
      setTokenLifetime(String(data.accessTokenLifetimeSeconds))
      setIsEditing(false)
    },
  })

  const disabled = !isEditing

  return (
    <div className="flex h-full">
      {/* Left 60% — settings */}
      <div className="w-[60%] px-6 py-6 overflow-y-auto">
        {/* Header row */}
        <div className="flex items-center justify-between border-b border-border pb-3 mb-6">
          <h3 className="text-sm font-semibold text-gray-900">Global Policy Compliance</h3>
          <button
            type="button"
            onClick={isEditing ? () => saveMutation.mutate() : () => setIsEditing(true)}
            disabled={saveMutation.isPending}
            className="px-4 py-1.5 bg-gray-800 hover:bg-gray-900 text-white text-xs font-medium rounded-md transition-colors disabled:opacity-60 w-[60px] text-center"
          >
            {saveMutation.isPending ? 'Saving…' : isEditing ? 'Save' : 'Edit'}
          </button>
        </div>

        <div className={`space-y-5 ${disabled ? 'opacity-60 pointer-events-none' : ''}`}>
          {/* Access Policy Compliance */}
          <div className="border border-border rounded-lg p-5">
            <h4 className="text-sm font-semibold text-gray-800 mb-4">Access Policy Compliance</h4>
            <div className="space-y-3">
              <EnforcementRow label="Trust Providers" value={apTrustProvider} onChange={setApTrustProvider} disabled={disabled} />
              <EnforcementRow label="Access Conditions" value={apAccessCondition} onChange={setApAccessCondition} disabled={disabled} />
            </div>
          </div>

          {/* Agent Controller Compliance */}
          <div className="border border-border rounded-lg p-5">
            <h4 className="text-sm font-semibold text-gray-800 mb-4">Agent Controller Compliance</h4>
            <div className="space-y-3">
              <EnforcementRow label="Trust Providers" value={acTrustProvider} onChange={setAcTrustProvider} disabled={disabled} />
              <EnforcementRow label="Allowed TLS Hostname" value={acTlsHostname} onChange={setAcTlsHostname} disabled={disabled} />
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
                  disabled={disabled}
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
            <span className="font-semibold text-gray-700">Global Policy Compliance</span> defines organization-wide security standards for Access Policies and Agent Controllers.
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
  disabled = false,
}: {
  label: string
  value: EnforcementLevel
  onChange: (v: EnforcementLevel) => void
  disabled?: boolean
}) {
  return (
    <div className="flex items-center justify-between gap-4">
      <span className="text-sm text-gray-700">{label}</span>
      <div className="flex items-center rounded-md border border-border overflow-hidden shrink-0">
        {ENFORCEMENT_LEVELS.map((level) => (
          <button
            key={level.value}
            type="button"
            onClick={() => !disabled && onChange(level.value)}
            disabled={disabled}
            className={`px-3 py-1.5 text-xs font-medium transition-colors ${
              value === level.value
                ? 'bg-gray-800 text-white'
                : 'bg-white text-gray-600 hover:bg-gray-50'
            } ${disabled ? 'cursor-default' : 'cursor-pointer'}`}
          >
            {level.label}
          </button>
        ))}
      </div>
    </div>
  )
}

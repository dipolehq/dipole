import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Pencil } from 'lucide-react'
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

  const { data, isLoading, isError } = useQuery<GlobalPolicyData>({
    queryKey: ['global-policy-compliance'],
    queryFn: () => apiClient.get<GlobalPolicyData>('/global-policy-compliance'),
  })

  if (isLoading) {
    return (
      <div className="flex-1 flex items-center justify-center">
        <p className="text-sm text-gray-400">Loading global policy compliance…</p>
      </div>
    )
  }

  if (isError || !data) {
    return (
      <div className="flex-1 flex items-center justify-center">
        <p className="text-sm text-gray-400">Unable to load global policy compliance settings.</p>
      </div>
    )
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
  const [error, setError] = useState<string | null>(null)

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
      setError(null)
      setIsEditing(false)
    },
    onError: () => {
      setError('Failed to save global policy compliance settings')
      setApTrustProvider(data.apTrustProvider as EnforcementLevel)
      setApAccessCondition(data.apAccessCondition as EnforcementLevel)
      setAcTrustProvider(data.acTrustProvider as EnforcementLevel)
      setAcTlsHostname(data.acTlsHostname as EnforcementLevel)
      setTokenLifetime(String(data.accessTokenLifetimeSeconds))
      setIsEditing(false)
    },
  })

  const disabled = !isEditing

  function handleCancel() {
    setApTrustProvider(data.apTrustProvider as EnforcementLevel)
    setApAccessCondition(data.apAccessCondition as EnforcementLevel)
    setAcTrustProvider(data.acTrustProvider as EnforcementLevel)
    setAcTlsHostname(data.acTlsHostname as EnforcementLevel)
    setTokenLifetime(String(data.accessTokenLifetimeSeconds))
    setIsEditing(false)
  }

  return (
    <>
      {/* Toolbar */}
      <div className="flex items-center justify-between gap-3 px-4 py-3 shrink-0">
        <h2 className="text-sm font-medium text-gray-900">Global Policy Compliance</h2>
        <div className="flex items-center gap-2">
          {isEditing ? (
            <>
              <button type="button" onClick={() => saveMutation.mutate()} disabled={saveMutation.isPending} className="inline-flex items-center gap-2 h-9 px-4 rounded-md border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors disabled:opacity-50">
                {saveMutation.isPending ? 'Saving…' : 'Save'}
              </button>
              <button type="button" onClick={handleCancel} className="inline-flex items-center gap-2 h-9 px-4 rounded-md border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors">
                Cancel
              </button>
            </>
          ) : (
            <button type="button" onClick={() => setIsEditing(true)} className="inline-flex items-center gap-2 h-9 px-4 rounded-md border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors">
              <Pencil size={14} strokeWidth={2} className="opacity-60" /> Edit
            </button>
          )}
        </div>
      </div>

      {error && (
        <div className="mx-4 mb-2 px-4 py-2 bg-red-50 border border-red-200 rounded-md text-sm text-red-700">
          {error}
        </div>
      )}

      {/* Content box */}
      <div className="flex-1 min-h-0 mx-4 border border-border bg-white overflow-y-auto">
        <div className="grid grid-cols-[1fr_380px] h-full">
          <div className="px-6 py-5 overflow-y-auto border-r border-border">
            <fieldset disabled={disabled} className={disabled ? 'opacity-60 pointer-events-none' : ''}>
              <div className="space-y-6">
                <div>
                  <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider mb-4">Access Policy Compliance</h4>
                  <div className="space-y-3">
                    <EnforcementRow label="Trust Providers" value={apTrustProvider} onChange={setApTrustProvider} disabled={disabled} />
                    <EnforcementRow label="Access Conditions" value={apAccessCondition} onChange={setApAccessCondition} disabled={disabled} />
                  </div>
                </div>

                <div>
                  <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider mb-4">Agent Controller Compliance</h4>
                  <div className="space-y-3">
                    <EnforcementRow label="Trust Providers" value={acTrustProvider} onChange={setAcTrustProvider} disabled={disabled} />
                    <EnforcementRow label="Allowed TLS Hostname" value={acTlsHostname} onChange={setAcTlsHostname} disabled={disabled} />
                  </div>
                </div>

                <div>
                  <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider mb-4">Access Token Compliance</h4>
                  <div className="flex items-end gap-3">
                    <div className="w-48">
                      <FloatingInput
                        label="API Access Token Lifetime"
                        value={tokenLifetime}
                        onChange={(v) => {
                          const n = Number(v)
                          if (v === '' || (n >= 0 && n <= 2592000)) setTokenLifetime(v)
                        }}
                        type="number"
                        disabled={disabled}
                      />
                    </div>
                    <span className="text-sm text-gray-500 pb-2.5">Seconds</span>
                  </div>
                  <p className="text-[11px] text-gray-400 mt-2">Maximum allowed: 2,592,000 seconds (30 days)</p>
                </div>
              </div>
            </fieldset>
          </div>

          <div className="px-5 py-5 overflow-y-auto">
            <div className="space-y-3">
              <p className="text-xs text-gray-600 leading-relaxed">
                <span className="font-semibold text-gray-800">Global Policy Compliance</span> defines organization-wide security standards for Access Policies and Agent Controllers.
              </p>
              <p className="text-xs text-gray-600 leading-relaxed">
                Each attribute can be set to <span className="font-semibold text-gray-800">Required</span> (must be present), <span className="font-semibold text-gray-800">Recommended</span> (flagged if missing), or <span className="font-semibold text-gray-800">Optional</span> (no enforcement).
              </p>
              <p className="text-xs text-gray-600 leading-relaxed">
                The <span className="font-semibold text-gray-800">Access Token Lifetime</span> sets the maximum validity period for API access tokens issued across the tenant.
              </p>
            </div>
          </div>
        </div>
      </div>
    </>
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

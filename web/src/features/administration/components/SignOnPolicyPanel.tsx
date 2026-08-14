import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Pencil } from 'lucide-react'
import { apiClient } from '@/shared/lib/api-client'

interface SignOnPolicyData {
  requireSso: boolean
  requireMfa: boolean
}

export function SignOnPolicyPanel() {
  const qc = useQueryClient()

  const { data } = useQuery<SignOnPolicyData>({
    queryKey: ['sign-on-policy'],
    queryFn: () => apiClient.get<SignOnPolicyData>('/sign-on-policy'),
  })

  if (!data) return <div className="flex-1" />

  return <SignOnPolicyForm data={data} queryClient={qc} />
}

function SignOnPolicyForm({ data, queryClient }: { data: SignOnPolicyData; queryClient: ReturnType<typeof useQueryClient> }) {
  const [requireSSO, setRequireSSO] = useState(data.requireSso)
  const [requireMFA, setRequireMFA] = useState(data.requireMfa)
  const [isEditing, setIsEditing] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const saveMutation = useMutation({
    mutationFn: () =>
      apiClient.put<SignOnPolicyData>('/sign-on-policy', {
        requireSso: requireSSO,
        requireMfa: requireMFA,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['sign-on-policy'] })
      setError(null)
      setIsEditing(false)
    },
    onError: () => {
      setError('Failed to save sign-on policy')
      setRequireSSO(data.requireSso)
      setRequireMFA(data.requireMfa)
      setIsEditing(false)
    },
  })

  const disabled = !isEditing

  return (
    <>
      {/* Toolbar */}
      <div className="flex items-center justify-between gap-3 px-4 py-3 shrink-0">
        <h2 className="text-sm font-medium text-gray-900">Sign-On Policy</h2>
        <div className="flex items-center gap-2">
          {isEditing ? (
            <>
              <button type="button" onClick={() => saveMutation.mutate()} disabled={saveMutation.isPending} className="inline-flex items-center gap-2 h-9 px-4 rounded-md border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors disabled:opacity-50">
                {saveMutation.isPending ? 'Saving…' : 'Save'}
              </button>
              <button type="button" onClick={() => { setRequireSSO(data.requireSso); setRequireMFA(data.requireMfa); setIsEditing(false) }} className="inline-flex items-center gap-2 h-9 px-4 rounded-md border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors">
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
                  <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider mb-4">Require Single Sign-On</h4>
                  <div className="flex items-center justify-between mb-3">
                    <p className="text-xs text-gray-600 leading-relaxed max-w-xxl">
                      Mandates that users authenticate through a Single Sign-On provider. This simplifies the login process and enhances security by centralizing authentication management.
                    </p>
                    <Toggle enabled={requireSSO} onChange={setRequireSSO} disabled={disabled} />
                  </div>
                  <p className="text-[11px] text-gray-400">
                    Note: Users with the SuperAdmin role can always use native sign-in regardless of this setting.
                  </p>
                </div>

                <div>
                  <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider mb-4">Require Multi-Factor Authentication</h4>
                  <div className="flex items-center justify-between mb-3">
                    <p className="text-xs text-gray-600 leading-relaxed max-w-xxl">
                      Enforces the use of multi-factor authentication (MFA) for users logging in through native sign-in. Users must provide an MFA code in addition to their password.
                    </p>
                    <Toggle enabled={requireMFA} onChange={setRequireMFA} disabled={disabled} />
                  </div>
                  <p className="text-[11px] text-gray-400">
                    Note: Users are given a 24-hour grace period to configure MFA after this setting is enabled.
                  </p>
                </div>
              </div>
            </fieldset>
          </div>

          <div className="px-5 py-5 overflow-y-auto">
            <div className="space-y-3">
              <p className="text-xs text-gray-600 leading-relaxed">
                <span className="font-semibold text-gray-800">Sign-On Policy</span> controls how users authenticate into the Dipole tenant.
              </p>
              <p className="text-xs text-gray-600 leading-relaxed">
                <span className="font-semibold text-gray-800">Single Sign-On</span> centralizes authentication through an external identity provider, while <span className="font-semibold text-gray-800">Multi-Factor Authentication</span> adds a second verification step for native sign-ins.
              </p>
              <p className="text-xs text-gray-600 leading-relaxed">
                These settings apply tenant-wide. SuperAdmin accounts retain native sign-in access to prevent lockout.
              </p>
            </div>
          </div>
        </div>
      </div>
    </>
  )
}

function Toggle({ enabled, onChange, disabled = false }: { enabled: boolean; onChange: (v: boolean) => void; disabled?: boolean }) {
  return (
    <button
      type="button"
      onClick={() => !disabled && onChange(!enabled)}
      disabled={disabled}
      className={`relative w-10 h-5 rounded-full transition-colors shrink-0 ${enabled ? 'bg-gray-800' : 'bg-gray-300'} ${disabled ? 'cursor-default' : 'cursor-pointer'}`}
    >
      <span className={`absolute top-0.5 left-0.5 w-4 h-4 bg-white rounded-full shadow transition-transform ${enabled ? 'translate-x-5' : 'translate-x-0'}`} />
    </button>
  )
}

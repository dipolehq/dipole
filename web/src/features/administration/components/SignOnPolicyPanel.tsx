import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
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

  // Don't render until data is loaded — prevents toggle animation flicker
  if (!data) {
    return <div className="flex-1" />
  }

  return <SignOnPolicyForm data={data} queryClient={qc} />
}

function SignOnPolicyForm({ data, queryClient }: { data: SignOnPolicyData; queryClient: ReturnType<typeof useQueryClient> }) {
  const [requireSSO, setRequireSSO] = useState(data.requireSso)
  const [requireMFA, setRequireMFA] = useState(data.requireMfa)
  const [isEditing, setIsEditing] = useState(false)

  const saveMutation = useMutation({
    mutationFn: () =>
      apiClient.put<SignOnPolicyData>('/sign-on-policy', {
        requireSso: requireSSO,
        requireMfa: requireMFA,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['sign-on-policy'] })
      setIsEditing(false)
    },
    onError: () => {
      // Revert to server state on failure
      setRequireSSO(data.requireSso)
      setRequireMFA(data.requireMfa)
      setIsEditing(false)
    },
  })

  function handleSave() {
    saveMutation.mutate()
  }

  function handleEdit() {
    setIsEditing(true)
  }

  const disabled = !isEditing

  return (
    <div className="flex h-full">
      {/* Left 60% — settings */}
      <div className="w-[60%] px-6 py-6 overflow-y-auto">
        {/* Header row */}
        <div className="flex items-center justify-between border-b border-border pb-3 mb-6">
          <h3 className="text-sm font-semibold text-gray-900">Sign-On Policy</h3>
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={isEditing ? handleSave : handleEdit}
              disabled={saveMutation.isPending}
              className="px-4 py-1.5 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors disabled:opacity-60 w-[60px] text-center"
            >
              {saveMutation.isPending ? 'Saving…' : isEditing ? 'Save' : 'Edit'}
            </button>
          </div>
        </div>

        <div className={`space-y-5 ${disabled ? 'opacity-60 pointer-events-none' : ''}`}>
          {/* Require Single Sign-On */}
          <div className="border border-border rounded-lg p-5">
            <div className="flex items-center justify-between mb-2">
              <h4 className="text-sm font-semibold text-gray-800">Require Single Sign-On</h4>
              <Toggle enabled={requireSSO} onChange={setRequireSSO} disabled={disabled} />
            </div>
            <p className="text-xs text-gray-600 leading-relaxed">
              Mandates that users authenticate through a Single Sign-On provider. This simplifies the login process and enhances security by centralizing authentication management.
            </p>
            <p className="text-[11px] text-gray-400 mt-2">
              Note: Users with the SuperAdmin role can always use native sign-in (username and password) regardless of this setting.
            </p>
          </div>

          {/* Require Multi-Factor Authentication */}
          <div className="border border-border rounded-lg p-5">
            <div className="flex items-center justify-between mb-2">
              <h4 className="text-sm font-semibold text-gray-800">Require Multi-Factor Authentication</h4>
              <Toggle enabled={requireMFA} onChange={setRequireMFA} disabled={disabled} />
            </div>
            <p className="text-xs text-gray-600 leading-relaxed">
              Enforces the use of multi-factor authentication (MFA) for users logging in through native sign-in. Users must provide an MFA code in addition to their password.
            </p>
            <p className="text-[11px] text-gray-400 mt-2">
              Note: Users are given a 24-hour grace period to configure MFA after this setting is enabled. After this period, accounts without MFA configured will be locked.
            </p>
          </div>
        </div>
      </div>

      {/* Right 40% — help */}
      <div className="w-[40%] px-6 py-6 overflow-y-auto">
        <div className="border border-border rounded-lg p-5 sticky top-0">
          <p className="text-xs text-gray-700 leading-relaxed mb-3">
            <span className="font-semibold text-orange-600">Sign-On Policy</span> controls how users authenticate into the Dipole tenant.
          </p>
          <p className="text-xs text-gray-700 leading-relaxed mb-3">
            <span className="font-semibold">Single Sign-On</span> centralizes authentication through an external identity provider, while <span className="font-semibold">Multi-Factor Authentication</span> adds a second verification step for native sign-ins.
          </p>
          <p className="text-xs text-gray-700 leading-relaxed">
            These settings apply tenant-wide. SuperAdmin accounts retain native sign-in access to prevent lockout.
          </p>
        </div>
      </div>
    </div>
  )
}

function Toggle({ enabled, onChange, disabled = false }: { enabled: boolean; onChange: (v: boolean) => void; disabled?: boolean }) {
  return (
    <button
      type="button"
      onClick={() => !disabled && onChange(!enabled)}
      disabled={disabled}
      className={`relative w-10 h-5 rounded-full transition-colors ${enabled ? 'bg-orange-500' : 'bg-gray-300'} ${disabled ? 'cursor-default' : 'cursor-pointer'}`}
    >
      <span
        className={`absolute top-0.5 left-0.5 w-4 h-4 bg-white rounded-full shadow transition-transform ${enabled ? 'translate-x-5' : 'translate-x-0'}`}
      />
    </button>
  )
}

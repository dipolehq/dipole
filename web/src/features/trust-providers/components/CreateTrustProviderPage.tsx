import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { PageHeader } from '@/shared/components/PageHeader'
import { FloatingDropdown } from '@/shared/components/FloatingInput'
import { TrustProviderForm } from '@/shared/components/forms/TrustProviderForm'
import { EMPTY_TRUST_PROVIDER } from '@/shared/components/forms/types'
import type { TrustProviderFormValues } from '@/shared/components/forms/types'
import { useCreateTrustProvider } from '@/features/trust-providers/hooks/useTrustProviders'
import { apiErrorMessage } from '@/features/trust-providers/api/trustProvidersApi'
import { trustProviderSchema } from '@/shared/lib/validation'

export function CreateTrustProviderPage() {
  const navigate = useNavigate()
  const createMutation = useCreateTrustProvider()
  const [values, setValues] = useState<TrustProviderFormValues>({ ...EMPTY_TRUST_PROVIDER })
  const [status, setStatus] = useState('active')
  const [error, setError] = useState<string | null>(null)

  async function handleSave() {
    setError(null)
    const result = trustProviderSchema.safeParse(values)
    if (!result.success) {
      setError(result.error.errors[0]?.message ?? 'Please fill in all required fields')
      return
    }
    try {
      await createMutation.mutateAsync({ values, isActive: status !== 'inactive' })
      navigate('/trust-providers')
    } catch (err) {
      setError(apiErrorMessage(err, 'Failed to create trust provider'))
    }
  }

  function handleCancel() {
    navigate('/trust-providers')
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-4 shrink-0 h-[44px] border-b border-border bg-white">
        <PageHeader />
      </div>

      <div className="flex items-center justify-end gap-2 px-4 py-3 shrink-0">
        <button
          type="button"
          onClick={handleSave}
          disabled={createMutation.isPending}
          className="inline-flex items-center gap-2 h-9 px-4 rounded-md border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors disabled:opacity-50"
        >
          {createMutation.isPending ? 'Saving…' : 'Save'}
        </button>
        <button
          type="button"
          onClick={handleCancel}
          className="inline-flex items-center gap-2 h-9 px-4 rounded-md border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors"
        >
          Cancel
        </button>
      </div>

      {error && (
        <div className="mx-4 mb-2 px-4 py-2 bg-red-50 border border-red-200 rounded-md text-sm text-red-700">
          {error}
        </div>
      )}

      <div className="flex-1 min-h-0 mx-4 border border-border bg-white overflow-y-auto">
        <div className="grid grid-cols-[1fr_380px] h-full">
          <div className="px-6 py-5 overflow-y-auto border-r border-border">
            <TrustProviderForm
              values={values}
              onChange={setValues}
              statusSlot={
                <FloatingDropdown
                  label="Status"
                  options={[
                    { value: 'active', label: 'Active' },
                    { value: 'inactive', label: 'Inactive' },
                  ]}
                  value={status}
                  onChange={setStatus}
                />
              }
            />
          </div>

          <div className="px-5 py-5 overflow-y-auto">
            <div className="space-y-3">
              <p className="text-xs text-gray-600 leading-relaxed">
                <span className="font-semibold text-gray-800">Trust Providers</span> verify
                workload identity using cryptographic attestation from the runtime environment. They
                answer: "Can the environment VOUCH for this workload's identity?"
              </p>
              <p className="text-xs text-gray-600 leading-relaxed">
                Instead of a shared secret, the workload's environment provides cryptographic proof.
                Supported types include Kubernetes Service Account tokens, AWS Metadata/Role,
                GitHub/GitLab OIDC tokens, and generic OIDC providers.
              </p>
              <p className="text-xs text-gray-600 leading-relaxed">
                <span className="font-semibold text-gray-800">Match Rules</span> define which claims
                or attributes must match for attestation to succeed. All rules use AND logic — every
                configured rule must pass.
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

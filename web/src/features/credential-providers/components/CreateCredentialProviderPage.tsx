import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { PageHeader } from '@/shared/components/PageHeader'
import { FloatingDropdown } from '@/shared/components/FloatingInput'
import { CredentialProviderForm } from '@/shared/components/forms/CredentialProviderForm'
import { EMPTY_CREDENTIAL_PROVIDER } from '@/shared/components/forms/types'
import type { CredentialProviderFormValues } from '@/shared/components/forms/types'
import { useCreateCredentialProvider } from '@/features/credential-providers/hooks/useCredentialProviders'
import { apiErrorMessage } from '@/features/credential-providers/api/credentialProvidersApi'

export function CreateCredentialProviderPage() {
  const navigate = useNavigate()
  const createMutation = useCreateCredentialProvider()
  const [values, setValues] = useState<CredentialProviderFormValues>({
    ...EMPTY_CREDENTIAL_PROVIDER,
  })
  const [status, setStatus] = useState('active')
  const [error, setError] = useState<string | null>(null)

  async function handleSave() {
    setError(null)
    try {
      await createMutation.mutateAsync({ values, isActive: status !== 'inactive' })
      navigate('/credential-providers')
    } catch (err) {
      setError(apiErrorMessage(err, 'Failed to create credential provider'))
    }
  }

  function handleCancel() {
    navigate('/credential-providers')
  }

  return (
    <div className="flex flex-col h-full">
      {/* Header bar */}
      <div className="flex items-center justify-between px-4 py-3 shrink-0 min-h-[36px] border-b border-border bg-white">
        <PageHeader />
        <div className="flex items-center gap-3">
          <button
            type="button"
            onClick={handleSave}
            disabled={createMutation.isPending}
            className="px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white text-sm font-medium rounded-md transition-colors disabled:opacity-60"
          >
            {createMutation.isPending ? 'Saving…' : 'Save'}
          </button>
          <button
            type="button"
            onClick={handleCancel}
            className="px-4 py-2 border border-gray-300 text-sm font-medium text-gray-600 hover:text-gray-900 hover:border-gray-400 rounded-md transition-colors"
          >
            Cancel
          </button>
        </div>
      </div>

      {error && (
        <div className="px-6 py-2 bg-red-50 border-b border-red-200 text-sm text-red-700 shrink-0">
          {error}
        </div>
      )}

      {/* Content */}
      <div className="flex-1 overflow-y-auto bg-white">
        <div className="flex h-full">
          {/* Form — left 60% */}
          <div className="w-[60%] px-6 py-6 overflow-y-auto">
            <div className="space-y-6">
              <CredentialProviderForm
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
          </div>

          {/* Help panel — right 40% */}
          <div className="w-[40%] px-6 py-6 overflow-y-auto">
            <div className="border border-border rounded-lg p-5 sticky top-0">
              <p className="text-xs text-gray-700 leading-relaxed mb-3">
                <span className="font-semibold text-orange-600">Credential Providers</span> are the
                systems that produce the credential injected into requests. They answer: "What
                credential does the Server Workload expect?"
              </p>
              <p className="text-xs text-gray-700 leading-relaxed mb-3">
                Supported types include <span className="font-semibold">API Key</span>,{' '}
                <span className="font-semibold">Username &amp; Password</span>,{' '}
                <span className="font-semibold">OAuth 2.0 Client Credentials</span>,{' '}
                <span className="font-semibold">AWS STS Federation</span>, and more.
              </p>
              <p className="text-xs text-gray-700 leading-relaxed">
                Credentials are retrieved just-in-time and injected transparently. The Client
                Workload never sees, stores, or manages the credential.
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

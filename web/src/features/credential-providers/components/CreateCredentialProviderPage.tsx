import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { FloatingDropdown } from '@/shared/components/FloatingInput'
import { CredentialProviderForm } from '@/shared/components/forms/CredentialProviderForm'
import { EMPTY_CREDENTIAL_PROVIDER } from '@/shared/components/forms/types'
import type { CredentialProviderFormValues } from '@/shared/components/forms/types'
import { useCreateCredentialProvider } from '@/features/credential-providers/hooks/useCredentialProviders'
import { apiErrorMessage } from '@/features/credential-providers/api/credentialProvidersApi'
import { credentialProviderSchema } from '@/shared/lib/validation'

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
    const result = credentialProviderSchema.safeParse(values)
    if (!result.success) {
      setError(result.error.errors[0]?.message ?? 'Please fill in all required fields')
      return
    }
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

      {/* Toolbar */}
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

      {/* Content box */}
      <div className="flex-1 min-h-0 mx-4 border border-border bg-white overflow-y-auto">
        <div className="grid grid-cols-[1fr_380px] h-full">
          {/* Form area */}
          <div className="px-6 py-5 overflow-y-auto border-r border-border">
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

          {/* Help text */}
          <div className="px-5 py-5 overflow-y-auto">
            <div className="space-y-3">
              <p className="text-xs text-gray-600 leading-relaxed">
                <span className="font-semibold text-gray-800">Credential Providers</span> are the
                systems that produce the credential injected into requests. They answer: "What
                credential does the Server Workload expect?"
              </p>
              <p className="text-xs text-gray-600 leading-relaxed">
                Supported types include <span className="font-semibold text-gray-800">API Key</span>,{' '}
                <span className="font-semibold text-gray-800">Username &amp; Password</span>,{' '}
                <span className="font-semibold text-gray-800">OAuth 2.0 Client Credentials</span>,{' '}
                <span className="font-semibold text-gray-800">AWS STS Federation</span>, and more.
              </p>
              <p className="text-xs text-gray-600 leading-relaxed">
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

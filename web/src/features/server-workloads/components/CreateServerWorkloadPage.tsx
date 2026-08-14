import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { PageHeader } from '@/shared/components/PageHeader'
import { FloatingDropdown } from '@/shared/components/FloatingInput'
import { ServerWorkloadForm } from '@/shared/components/forms/ServerWorkloadForm'
import { EMPTY_SERVER_WORKLOAD } from '@/shared/components/forms/types'
import type { ServerWorkloadFormValues } from '@/shared/components/forms/types'
import { useCreateServerWorkload } from '@/features/server-workloads/hooks/useServerWorkloads'
import { apiErrorMessage } from '@/features/server-workloads/api/serverWorkloadsApi'
import { serverWorkloadSchema } from '@/shared/lib/validation'

export function CreateServerWorkloadPage() {
  const navigate = useNavigate()
  const createMutation = useCreateServerWorkload()
  const [values, setValues] = useState<ServerWorkloadFormValues>({ ...EMPTY_SERVER_WORKLOAD })
  const [status, setStatus] = useState('active')
  const [error, setError] = useState<string | null>(null)

  async function handleSave() {
    setError(null)
    const result = serverWorkloadSchema.safeParse(values)
    if (!result.success) {
      setError(result.error.errors[0]?.message ?? 'Please fill in all required fields')
      return
    }
    try {
      await createMutation.mutateAsync({ values, isActive: status !== 'inactive' })
      navigate('/server-workloads')
    } catch (err) {
      setError(apiErrorMessage(err, 'Failed to create server workload'))
    }
  }

  function handleCancel() {
    navigate('/server-workloads')
  }

  return (
    <div className="flex flex-col h-full">
      {/* Header bar */}
      <div className="flex items-center justify-between px-4 shrink-0 h-[60px] border-b border-border bg-white">
        <PageHeader />
        <div className="flex items-center gap-3">
          <button
            type="button"
            onClick={handleSave}
            disabled={createMutation.isPending}
            className="px-4 py-2 bg-gray-800 hover:bg-gray-900 text-white text-sm font-medium rounded-md transition-colors disabled:opacity-60"
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
          <div className="w-[60%] px-6 py-6 overflow-y-auto">
            <div className="space-y-6">
              <ServerWorkloadForm
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
          <div className="w-[40%] px-6 py-6 overflow-y-auto">
            <div className="border border-border rounded-lg p-5 sticky top-0">
              <p className="text-xs text-gray-700 leading-relaxed mb-3">
                <span className="font-semibold text-gray-700">Server Workloads</span> represent
                the services, APIs, or databases that Client Workloads access. They're identified by
                host, port, and protocol.
              </p>
              <p className="text-xs text-gray-700 leading-relaxed mb-3">
                The <span className="font-semibold">authentication method</span> determines how
                credentials are injected — HTTP Bearer, Basic, API Key header, database password
                auth, AWS SigV4, or mTLS.
              </p>
              <p className="text-xs text-gray-700 leading-relaxed">
                Dipole supports HTTP, PostgreSQL, MySQL, Redis, Snowflake, Redshift, Oracle, MCP,
                and TCP passthrough protocols.
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

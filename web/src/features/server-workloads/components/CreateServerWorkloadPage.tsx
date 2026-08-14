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
      <div className="flex items-center justify-between px-4 shrink-0 h-[44px] border-b border-border bg-white">
        <PageHeader />
      </div>

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

          {/* Help text */}
          <div className="px-5 py-5 overflow-y-auto">
            <div className="space-y-3">
              <p className="text-xs text-gray-600 leading-relaxed">
                <span className="font-semibold text-gray-800">Server Workloads</span> represent
                the services, APIs, or databases that Client Workloads access. They're identified by
                host, port, and protocol.
              </p>
              <p className="text-xs text-gray-600 leading-relaxed">
                The <span className="font-semibold text-gray-800">authentication method</span> determines how
                credentials are injected — HTTP Bearer, Basic, API Key header, database password
                auth, AWS SigV4, or mTLS.
              </p>
              <p className="text-xs text-gray-600 leading-relaxed">
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

import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { PageHeader } from '@/shared/components/PageHeader'
import { FloatingDropdown } from '@/shared/components/FloatingInput'
import { ClientWorkloadForm } from '@/shared/components/forms/ClientWorkloadForm'
import { EMPTY_CLIENT_WORKLOAD } from '@/shared/components/forms/types'
import type { ClientWorkloadFormValues } from '@/shared/components/forms/types'
import { useCreateClientWorkload } from '@/features/client-workloads/hooks/useClientWorkloads'
import { apiErrorMessage } from '@/features/client-workloads/api/clientWorkloadsApi'
import { clientWorkloadSchema } from '@/shared/lib/validation'

export function CreateClientWorkloadPage() {
  const navigate = useNavigate()
  const createMutation = useCreateClientWorkload()
  const [values, setValues] = useState<ClientWorkloadFormValues>({ ...EMPTY_CLIENT_WORKLOAD })
  const [status, setStatus] = useState('active')
  const [error, setError] = useState<string | null>(null)

  async function handleSave() {
    setError(null)
    const result = clientWorkloadSchema.safeParse(values)
    if (!result.success) {
      setError(result.error.errors[0]?.message ?? 'Please fill in all required fields')
      return
    }
    try {
      await createMutation.mutateAsync({ values, isActive: status !== 'inactive' })
      navigate('/client-workloads')
    } catch (err) {
      setError(apiErrorMessage(err, 'Failed to create client workload'))
    }
  }

  function handleCancel() {
    navigate('/client-workloads')
  }

  return (
    <div className="flex flex-col h-full">
      {/* Header bar — same as table/detail pages */}
      <div className="flex items-center justify-between px-4 shrink-0 h-[44px] border-b border-border bg-white">
        <PageHeader />
      </div>

      {/* Toolbar — Save + Cancel on the right */}
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

      {/* Content box — same border treatment as detail page */}
      <div className="flex-1 min-h-0 mx-4 border border-border bg-white overflow-y-auto">
        <div className="grid grid-cols-[1fr_380px] h-full">
          {/* Form area */}
          <div className="px-6 py-5 overflow-y-auto border-r border-border">
            <ClientWorkloadForm
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

          {/* Help text — right panel */}
          <div className="px-5 py-5 overflow-y-auto">
            <div className="space-y-3">
              <p className="text-xs text-gray-600 leading-relaxed">
                <span className="font-semibold text-gray-800">Client Workloads</span> are software
                applications that access services provided by Server Workloads. Examples include
                custom applications, daemons, ETL processors, one-off or batch jobs, and deployment
                scripts. Typically, Client Workloads run in the background and must access services
                on their own behalf, without user intervention.
              </p>
              <p className="text-xs text-gray-600 leading-relaxed">
                A <span className="font-semibold text-gray-800">Client Identifier</span> is a bit
                like the Client Workload's username. Dipole uses it to identify the workload but not
                to authenticate it. If you deploy your workload in Kubernetes, we recommend using its
                Kubernetes Pod Name Prefix as its identifier.
              </p>
              <p className="text-xs text-gray-600 leading-relaxed">
                Dipole supports identifying{' '}
                <span className="font-semibold text-gray-800">Client Workloads</span> by using
                multiple Client Identifiers and typically requires all of the values be matched.
                Multiple values within a Client Identifier type can be used in an either OR
                combination, though at least one of the values for each Client Identifier type will
                be required.
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

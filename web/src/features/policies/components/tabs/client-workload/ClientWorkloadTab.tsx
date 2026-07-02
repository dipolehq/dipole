import { useState } from 'react'
import { ExistingWorkloadSelector } from '@/features/policies/components/tabs/client-workload/ExistingWorkloadSelector'
import { ClientWorkloadForm } from '@/shared/components/forms/ClientWorkloadForm'
import { buildClientWorkloadValues } from '@/data/policies/policyFormMappers'
import type { ClientWorkloadFormValues } from '@/shared/components/forms/types'

interface ClientWorkloadTabProps {
  selectedId: string
  onSelect: (id: string) => void
  values: ClientWorkloadFormValues
  onChange: (values: ClientWorkloadFormValues) => void
  disabled?: boolean
}

export function ClientWorkloadTab({
  selectedId,
  onSelect,
  values,
  onChange,
  disabled = false,
}: ClientWorkloadTabProps) {
  const [mode, setMode] = useState<'new' | 'existing'>('new')

  function handleSelectExisting(id: string) {
    onSelect(id)
    onChange(buildClientWorkloadValues(id))
    setMode('new')
  }

  return (
    <div className="space-y-6">
      <h3 className="text-sm font-semibold text-gray-900">Client Workload</h3>

      {!disabled && (
        <div className="flex items-center gap-4 border-b border-border pb-3">
          <button
            type="button"
            onClick={() => setMode('new')}
            className={`text-xs font-medium transition-colors ${mode === 'new' ? 'text-orange-600' : 'text-gray-500 hover:text-gray-700'}`}
          >
            Add New
          </button>
          <button
            type="button"
            onClick={() => setMode('existing')}
            className={`text-xs font-medium transition-colors ${mode === 'existing' ? 'text-orange-600' : 'text-gray-500 hover:text-gray-700'}`}
          >
            Select Existing
          </button>
        </div>
      )}

      {!disabled && mode === 'existing' ? (
        <ExistingWorkloadSelector
          selectedId={selectedId}
          onSelect={handleSelectExisting}
          onCancel={() => setMode('new')}
        />
      ) : (
        <ClientWorkloadForm values={values} onChange={onChange} disabled={disabled} />
      )}
    </div>
  )
}

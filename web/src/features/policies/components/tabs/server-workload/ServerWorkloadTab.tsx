import { useState } from 'react'
import { ExistingServerWorkloadSelector } from '@/features/policies/components/tabs/server-workload/ExistingServerWorkloadSelector'
import { ServerWorkloadForm } from '@/shared/components/forms/ServerWorkloadForm'
import { buildServerWorkloadValues } from '@/data/policies/policyFormMappers'
import type { ServerWorkloadFormValues } from '@/shared/components/forms/types'

interface ServerWorkloadTabProps {
  selectedId: string
  onSelect: (id: string) => void
  values: ServerWorkloadFormValues
  onChange: (values: ServerWorkloadFormValues) => void
  disabled?: boolean
}

export function ServerWorkloadTab({
  selectedId,
  onSelect,
  values,
  onChange,
  disabled = false,
}: ServerWorkloadTabProps) {
  const [mode, setMode] = useState<'new' | 'existing'>('new')

  function handleSelectExisting(id: string) {
    onSelect(id)
    onChange(buildServerWorkloadValues(id))
    setMode('new')
  }

  return (
    <div className="space-y-6">
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
        <ExistingServerWorkloadSelector
          selectedId={selectedId}
          onSelect={handleSelectExisting}
          onCancel={() => setMode('new')}
        />
      ) : (
        <ServerWorkloadForm values={values} onChange={onChange} disabled={disabled} />
      )}
    </div>
  )
}

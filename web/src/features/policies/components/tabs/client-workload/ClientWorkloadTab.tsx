import { useState } from 'react'
import { ExistingWorkloadSelector } from '@/features/policies/components/tabs/client-workload/ExistingWorkloadSelector'
import { ClientWorkloadForm } from '@/shared/components/forms/ClientWorkloadForm'
import { fetchClientWorkload } from '@/features/policies/api/entityApi'
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

  async function handleSelectExisting(id: string) {
    onSelect(id)
    try {
      const cw = await fetchClientWorkload(id)
      onChange({
        name: cw.name,
        description: cw.description,
        identifiers: (cw.identifiers ?? []).map((ident) => ({
          type: ident.type,
          values: [ident.value || ''],
        })),
        tags: [],
      })
    } catch {
      // If fetch fails, just set the ID without populating form
    }
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

import { useState } from 'react'
import { ExistingWorkloadSelector } from '@/features/policies/components/tabs/client-workload/ExistingWorkloadSelector'
import { ClientWorkloadForm } from '@/shared/components/forms/ClientWorkloadForm'

interface ClientWorkloadTabProps {
  selectedId: string
  onSelect: (id: string) => void
}

export function ClientWorkloadTab({ selectedId, onSelect }: ClientWorkloadTabProps) {
  const [mode, setMode] = useState<'new' | 'existing'>('new')

  return (
    <div className="space-y-6">
      <h3 className="text-sm font-semibold text-gray-900">Client Workload</h3>

      <div className="flex items-center gap-4 border-b border-border pb-3">
        <button
          onClick={() => setMode('new')}
          className={`text-xs font-medium transition-colors ${mode === 'new' ? 'text-orange-600' : 'text-gray-500 hover:text-gray-700'}`}
        >
          Add New
        </button>
        <button
          onClick={() => setMode('existing')}
          className={`text-xs font-medium transition-colors ${mode === 'existing' ? 'text-orange-600' : 'text-gray-500 hover:text-gray-700'}`}
        >
          Select Existing
        </button>
      </div>

      {mode === 'existing' ? (
        <ExistingWorkloadSelector
          selectedId={selectedId}
          onSelect={onSelect}
          onCancel={() => setMode('new')}
        />
      ) : (
        <ClientWorkloadForm />
      )}
    </div>
  )
}

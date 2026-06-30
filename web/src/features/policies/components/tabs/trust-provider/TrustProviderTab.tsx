import { useState } from 'react'
import { ExistingTrustProviderSelector } from '@/features/policies/components/tabs/trust-provider/ExistingTrustProviderSelector'
import { TrustProviderForm } from '@/shared/components/forms/TrustProviderForm'

interface TrustProviderTabProps {
  selectedIds: string[]
  onToggle: (id: string) => void
}

export function TrustProviderTab({ selectedIds, onToggle }: TrustProviderTabProps) {
  const [mode, setMode] = useState<'new' | 'existing'>('new')

  return (
    <div className="space-y-6">
      <h3 className="text-sm font-semibold text-gray-900">Trust Providers</h3>

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
        <ExistingTrustProviderSelector
          selectedIds={selectedIds}
          onToggle={onToggle}
          onCancel={() => setMode('new')}
        />
      ) : (
        <TrustProviderForm />
      )}
    </div>
  )
}

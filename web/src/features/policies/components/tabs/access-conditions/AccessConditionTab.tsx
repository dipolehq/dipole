import { useState } from 'react'
import { ExistingAccessConditionSelector } from '@/features/policies/components/tabs/access-conditions/ExistingAccessConditionSelector'
import { AccessConditionForm } from '@/shared/components/forms/AccessConditionForm'

interface AccessConditionTabProps {
  selectedIds: string[]
  onToggle: (id: string) => void
}

interface ConditionEntry {
  id: number
  isActive: boolean
  expanded: boolean
}

export function AccessConditionTab({ selectedIds, onToggle }: AccessConditionTabProps) {
  const [mode, setMode] = useState<'new' | 'existing'>('new')
  const [conditions, setConditions] = useState<ConditionEntry[]>([
    { id: 1, isActive: true, expanded: true },
  ])
  const [nextId, setNextId] = useState(2)

  function addCondition() {
    setConditions([...conditions, { id: nextId, isActive: true, expanded: true }])
    setNextId(nextId + 1)
  }

  function updateCondition(id: number, field: keyof ConditionEntry, value: boolean) {
    setConditions(conditions.map((c) => c.id === id ? { ...c, [field]: value } : c))
  }

  function removeCondition(id: number) {
    setConditions(conditions.filter((c) => c.id !== id))
  }

  function toggleExpand(id: number) {
    setConditions(conditions.map((c) => c.id === id ? { ...c, expanded: !c.expanded } : c))
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold text-gray-900">Access Conditions</h3>
      </div>

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
        <ExistingAccessConditionSelector
          selectedIds={selectedIds}
          onToggle={onToggle}
          onCancel={() => setMode('new')}
        />
      ) : (
        <div className="space-y-4">
          {conditions.map((condition) => (
            <div key={condition.id} className="border border-border rounded-lg overflow-hidden">
              {/* Card header */}
              <div className="flex items-center justify-between px-4 py-3 bg-gray-50/50">
                <div className="flex items-center gap-3">
                  <label className="flex items-center gap-1.5 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={condition.isActive}
                      onChange={(e) => updateCondition(condition.id, 'isActive', e.target.checked)}
                      className="w-3.5 h-3.5 rounded border-gray-300 text-green-500 focus:ring-green-200"
                    />
                    <span className="text-[10px] font-medium text-green-600">{condition.isActive ? 'Active' : 'Inactive'}</span>
                  </label>
                </div>
                <div className="flex items-center gap-3">
                  <button
                    onClick={() => toggleExpand(condition.id)}
                    className="text-xs text-gray-500 hover:text-gray-700 font-medium"
                  >
                    {condition.expanded ? 'Hide Details' : 'Show Details'}
                  </button>
                  {conditions.length > 1 && (
                    <button
                      onClick={() => removeCondition(condition.id)}
                      className="text-gray-400 hover:text-red-500"
                      title="Remove"
                    >
                      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" /></svg>
                    </button>
                  )}
                </div>
              </div>

              {/* Card body */}
              {condition.expanded && (
                <AccessConditionForm />
              )}
            </div>
          ))}

          <button onClick={addCondition} className="w-full py-2 border border-dashed border-orange-300 text-orange-500 hover:bg-orange-50 text-xs font-medium rounded-md transition-colors">
            + Add Another Condition
          </button>
        </div>
      )}
    </div>
  )
}

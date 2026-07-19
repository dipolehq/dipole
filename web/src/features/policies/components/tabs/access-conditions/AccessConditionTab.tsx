import { useState } from 'react'
import { ExistingAccessConditionSelector } from '@/features/policies/components/tabs/access-conditions/ExistingAccessConditionSelector'
import { AccessConditionForm } from '@/shared/components/forms/AccessConditionForm'
import { fetchAccessCondition } from '@/features/policies/api/entityApi'
import { EMPTY_ACCESS_CONDITION } from '@/shared/components/forms/types'
import type { AccessConditionFormValues } from '@/shared/components/forms/types'

interface AccessConditionTabProps {
  selectedIds: string[]
  onToggleId: (id: string) => void
  values: AccessConditionFormValues[]
  onChange: (values: AccessConditionFormValues[]) => void
  disabled?: boolean
}

export function AccessConditionTab({
  selectedIds,
  onToggleId,
  values,
  onChange,
  disabled = false,
}: AccessConditionTabProps) {
  const [mode, setMode] = useState<'new' | 'existing'>('new')
  const [expandedIndex, setExpandedIndex] = useState<Set<number>>(new Set(values.map((_, i) => i)))

  const displayValues = values.length === 0 && !disabled ? [{ ...EMPTY_ACCESS_CONDITION }] : values

  function updateAt(index: number, next: AccessConditionFormValues) {
    const clone = [...displayValues]
    clone[index] = next
    onChange(clone)
  }

  function addCondition() {
    onChange([...displayValues, { ...EMPTY_ACCESS_CONDITION }])
    setExpandedIndex((prev) => new Set([...prev, displayValues.length]))
  }

  function removeCondition(index: number) {
    onChange(displayValues.filter((_, i) => i !== index))
  }

  function toggleExpand(index: number) {
    setExpandedIndex((prev) => {
      const next = new Set(prev)
      if (next.has(index)) next.delete(index)
      else next.add(index)
      return next
    })
  }

  async function handleAddExisting(id: string) {
    if (!selectedIds.includes(id)) onToggleId(id)
    try {
      const ac = await fetchAccessCondition(id)
      const config = ac.config as Record<string, unknown> | undefined
      onChange([
        ...values,
        {
          name: ac.name,
          description: ac.description,
          integration: ac.type,
          timezone: (config?.timezone as string) ?? '',
          days: (config?.days as AccessConditionFormValues['days']) ?? [],
          countries: (config?.countries as AccessConditionFormValues['countries']) ?? [],
          tags: [],
        },
      ])
    } catch {
      onChange([...values, { ...EMPTY_ACCESS_CONDITION }])
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
        <ExistingAccessConditionSelector
          selectedIds={selectedIds}
          onToggle={handleAddExisting}
          onCancel={() => setMode('new')}
        />
      ) : (
        <div className="space-y-4">
          {displayValues.length === 0 && disabled && (
            <div className="text-xs text-gray-400 border border-dashed border-border rounded-md py-6 text-center">
              No access conditions configured for this policy.
            </div>
          )}

          {displayValues.map((condition, index) => (
            <div key={index} className="border border-border rounded-lg overflow-hidden">
              <div className="flex items-center justify-between px-4 py-3 bg-gray-50/50">
                <span className="text-xs font-medium text-gray-700 truncate">
                  {condition.name || 'Unnamed Condition'}
                </span>
                <div className="flex items-center gap-3">
                  <button
                    type="button"
                    onClick={() => toggleExpand(index)}
                    className="text-xs text-gray-500 hover:text-gray-700 font-medium"
                  >
                    {expandedIndex.has(index) ? 'Hide Details' : 'Show Details'}
                  </button>
                  {!disabled && (
                    <button
                      type="button"
                      onClick={() => removeCondition(index)}
                      className="text-gray-400 hover:text-red-500"
                      title="Remove condition"
                    >
                      <svg
                        className="w-4 h-4"
                        fill="none"
                        stroke="currentColor"
                        viewBox="0 0 24 24"
                      >
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          strokeWidth="2"
                          d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                        />
                      </svg>
                    </button>
                  )}
                </div>
              </div>
              {expandedIndex.has(index) && (
                <AccessConditionForm
                  values={condition}
                  onChange={(v) => updateAt(index, v)}
                  disabled={disabled}
                />
              )}
            </div>
          ))}

          {!disabled && (
            <button
              type="button"
              onClick={addCondition}
              className="w-full py-2 border border-dashed border-orange-300 text-orange-500 hover:bg-orange-50 text-xs font-medium rounded-md transition-colors"
            >
              + Add Another Condition
            </button>
          )}
        </div>
      )}
    </div>
  )
}

import { useState } from 'react'
import { ExistingTrustProviderSelector } from '@/features/policies/components/tabs/trust-provider/ExistingTrustProviderSelector'
import { TrustProviderForm } from '@/shared/components/forms/TrustProviderForm'
import { fetchTrustProvider } from '@/features/policies/api/entityApi'
import { EMPTY_TRUST_PROVIDER } from '@/shared/components/forms/types'
import type { TrustProviderFormValues } from '@/shared/components/forms/types'

interface TrustProviderTabProps {
  selectedIds: string[]
  onToggleId: (id: string) => void
  values: TrustProviderFormValues[]
  onChange: (values: TrustProviderFormValues[]) => void
  disabled?: boolean
}

export function TrustProviderTab({
  selectedIds,
  onToggleId,
  values,
  onChange,
  disabled = false,
}: TrustProviderTabProps) {
  const [mode, setMode] = useState<'new' | 'existing'>('new')
  const [expandedIndex, setExpandedIndex] = useState<Set<number>>(new Set(values.map((_, i) => i)))

  // Ensure there is at least one form when editing.
  const displayValues = values.length === 0 && !disabled ? [{ ...EMPTY_TRUST_PROVIDER }] : values

  function updateAt(index: number, next: TrustProviderFormValues) {
    const clone = [...displayValues]
    clone[index] = next
    onChange(clone)
  }

  function addProvider() {
    onChange([...displayValues, { ...EMPTY_TRUST_PROVIDER }])
    setExpandedIndex((prev) => new Set([...prev, displayValues.length]))
  }

  function removeProvider(index: number) {
    if (displayValues.length <= 1) return
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
      const tp = await fetchTrustProvider(id)
      const matchRules = (tp.matchRules ?? []).length > 0
        ? tp.matchRules
        : [{ attribute: '', value: '' }]
      onChange([
        ...values,
        {
          name: tp.name,
          description: tp.description,
          trustProviderType: tp.type,
          matchRules,
          certificate: '',
          thumbprint: '',
          subject: '',
          expiresAt: '',
          oidcEndpoint: '',
          attestationMethod: 'oidc_discovery',
          samlMetadataUrl: '',
          samlMetadataXml: '',
          symmetricKey: '',
          jwks: '',
          publicKey: '',
          tags: [],
        },
      ])
    } catch {
      onChange([...values, { ...EMPTY_TRUST_PROVIDER }])
    }
    setMode('new')
  }

  const hasMultiple = displayValues.length > 1

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
        <ExistingTrustProviderSelector
          selectedIds={selectedIds}
          onToggle={handleAddExisting}
          onCancel={() => setMode('new')}
        />
      ) : displayValues.length === 0 ? (
        <div className="text-xs text-gray-400 border border-dashed border-border rounded-md py-6 text-center">
          No trust providers configured for this policy.
        </div>
      ) : hasMultiple ? (
        <div className="space-y-4">
          {displayValues.map((provider, index) => (
            <div key={index} className="border border-border rounded-lg overflow-hidden">
              <div className="flex items-center justify-between px-4 py-3 bg-gray-50/50">
                <span className="text-xs font-medium text-gray-700 truncate">
                  {provider.name || 'Unnamed Trust Provider'}
                </span>
                <div className="flex items-center gap-3">
                  <button
                    type="button"
                    onClick={() => toggleExpand(index)}
                    className="text-xs text-gray-500 hover:text-gray-700 font-medium"
                  >
                    {expandedIndex.has(index) ? 'Hide Details' : 'Show Details'}
                  </button>
                  {!disabled && displayValues.length > 1 && (
                    <button
                      type="button"
                      onClick={() => removeProvider(index)}
                      className="text-gray-400 hover:text-red-500"
                      title="Remove trust provider"
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
                <div className="p-4">
                  <TrustProviderForm
                    values={provider}
                    onChange={(v) => updateAt(index, v)}
                    disabled={disabled}
                  />
                </div>
              )}
            </div>
          ))}
          {!disabled && (
            <button
              type="button"
              onClick={addProvider}
              className="w-full py-2 border border-dashed border-orange-300 text-orange-500 hover:bg-orange-50 text-xs font-medium rounded-md transition-colors"
            >
              + Add Another Trust Provider
            </button>
          )}
        </div>
      ) : (
        (() => {
          const only = displayValues[0]
          if (!only) return null
          return (
            <>
              <TrustProviderForm
                values={only}
                onChange={(v) => updateAt(0, v)}
                disabled={disabled}
              />
              {!disabled && (
                <button
                  type="button"
                  onClick={addProvider}
                  className="w-full py-2 border border-dashed border-orange-300 text-orange-500 hover:bg-orange-50 text-xs font-medium rounded-md transition-colors"
                >
                  + Add Another Trust Provider
                </button>
              )}
            </>
          )
        })()
      )}
    </div>
  )
}

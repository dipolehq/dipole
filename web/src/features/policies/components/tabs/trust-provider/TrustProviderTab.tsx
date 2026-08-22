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
    // If this entry was hydrated from an existing provider (tagged with
    // existingId when added — see handleAddExisting), also drop its id
    // from trustProviderIds. Without this, the trash icon only trimmed the
    // display array and left the id attached, so it silently reappeared
    // after save (fetchPolicy re-hydrates from trustProviderIds).
    const existingId = displayValues[index]?.existingId
    if (existingId) onToggleId(existingId)
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
    // The selector calls this for both additions and removals (a real
    // toggle). If already selected, this is a removal.
    if (selectedIds.includes(id)) {
      const index = displayValues.findIndex((v) => v.existingId === id)
      onToggleId(id)
      if (index !== -1) onChange(displayValues.filter((_, i) => i !== index))
      return
    }
    onToggleId(id)
    try {
      const tp = await fetchTrustProvider(id)
      const matchRules = (tp.matchRules ?? []).length > 0
        ? tp.matchRules
        : [{ attribute: '', value: '' }]
      const cfg = (tp.config ?? {}) as Record<string, string>
      onChange([
        ...values,
        {
          // Tagging this entry as existing tells createPolicy() to reuse
          // trustProviderIds rather than POST a new /trust-providers record
          // for it — see the comment on TrustProviderFormValues.existingId.
          existingId: id,
          name: tp.name,
          description: tp.description,
          trustProviderType: tp.type,
          matchRules,
          certificate: cfg.certificate ?? '',
          thumbprint: cfg.thumbprint ?? '',
          subject: cfg.subject ?? '',
          expiresAt: cfg.expiresAt ?? '',
          oidcEndpoint: cfg.oidcEndpoint ?? '',
          attestationMethod: cfg.attestationMethod ?? 'oidc_discovery',
          samlMetadataUrl: cfg.samlMetadataUrl ?? '',
          samlMetadataXml: cfg.samlMetadataXml ?? '',
          symmetricKey: cfg.symmetricKey ?? '',
          jwks: cfg.jwks ?? '',
          publicKey: cfg.publicKey ?? '',
          tags: tp.tags ?? [],
        },
      ])
    } catch {
      // Keep the id selected (already toggled on above) even though we
      // couldn't fetch its details — push a placeholder tagged with
      // existingId so it still displays as a card, stays removable via
      // removeProvider, and is never mistaken for a brand-new entry to
      // create. Reverting the toggle here would silently un-select the
      // provider on a transient fetch failure.
      onChange([...values, { ...EMPTY_TRUST_PROVIDER, existingId: id }])
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
            className={`text-xs font-medium transition-colors ${mode === 'new' ? 'text-gray-700' : 'text-gray-500 hover:text-gray-700'}`}
          >
            Add New
          </button>
          <button
            type="button"
            onClick={() => setMode('existing')}
            className={`text-xs font-medium transition-colors ${mode === 'existing' ? 'text-gray-700' : 'text-gray-500 hover:text-gray-700'}`}
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
              className="w-full py-2 border border-dashed border-gray-300 text-gray-600 hover:bg-gray-50 text-xs font-medium rounded-md transition-colors"
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
                  className="w-full py-2 border border-dashed border-gray-300 text-gray-600 hover:bg-gray-50 text-xs font-medium rounded-md transition-colors"
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

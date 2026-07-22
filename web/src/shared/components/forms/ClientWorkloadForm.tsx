import { type ReactNode } from 'react'
import { TagsField } from '@/shared/components/TagsField'
import {
  FloatingInput,
  FloatingTextarea,
  FloatingDropdown,
} from '@/shared/components/FloatingInput'
import { CLIENT_IDENTIFIER_TYPES } from '@/shared/constants/policies'
import type { ClientWorkloadFormValues } from '@/shared/components/forms/types'

interface ClientWorkloadFormProps {
  values: ClientWorkloadFormValues
  onChange: (values: ClientWorkloadFormValues) => void
  disabled?: boolean
  statusSlot?: ReactNode
}

export function ClientWorkloadForm({
  values,
  onChange,
  disabled = false,
  statusSlot,
}: ClientWorkloadFormProps) {
  function update(partial: Partial<ClientWorkloadFormValues>) {
    onChange({ ...values, ...partial })
  }

  function updateIdentifierType(idx: number, type: string) {
    const identifiers = values.identifiers.map((ident, i) =>
      i === idx ? { ...ident, type } : ident,
    )
    update({ identifiers })
  }

  function updateIdentifierValue(idx: number, valIdx: number, val: string) {
    const identifiers = values.identifiers.map((ident, i) => {
      if (i !== idx) return ident
      const vals = [...ident.values]
      vals[valIdx] = val
      return { ...ident, values: vals }
    })
    update({ identifiers })
  }

  function addOrValue(idx: number) {
    const identifiers = values.identifiers.map((ident, i) => {
      if (i !== idx) return ident
      return { ...ident, values: [...ident.values, ''] }
    })
    update({ identifiers })
  }

  function removeOrValue(idx: number, valIdx: number) {
    const identifiers = values.identifiers.map((ident, i) => {
      if (i !== idx) return ident
      const vals = ident.values.filter((_, vi) => vi !== valIdx)
      return { ...ident, values: vals.length === 0 ? [''] : vals }
    })
    update({ identifiers })
  }

  function addIdentifier() {
    update({ identifiers: [...values.identifiers, { type: '', values: [''] }] })
  }

  function removeIdentifier(idx: number) {
    if (values.identifiers.length > 1) {
      update({ identifiers: values.identifiers.filter((_, i) => i !== idx) })
    }
  }

  const disabledClass = disabled ? 'opacity-60 pointer-events-none' : ''

  return (
    <fieldset disabled={disabled} className={disabledClass}>
      <div className="space-y-5">
        <div className="border border-border rounded-lg p-5">
          <h4 className="text-sm font-semibold text-gray-800 mb-4">Details</h4>
          <div className="space-y-4">
            {statusSlot}
            <FloatingInput
              label="Name"
              value={values.name}
              onChange={(v) => update({ name: v })}
              disabled={disabled}
              required
            />
            <FloatingTextarea
              label="Description (Optional)"
              value={values.description}
              onChange={(v) => update({ description: v })}
              disabled={disabled}
              rows={2}
            />
          </div>
        </div>

        <div className="border border-border rounded-lg p-5">
          <h4 className="text-sm font-semibold text-gray-800 mb-4">Client Identification</h4>
          <div className="space-y-3">
            {values.identifiers.map((ident, idx) => (
              <div key={idx}>
                {idx > 0 && (
                  <span className="inline-block px-2 py-0.5 bg-gray-100 text-gray-600 text-[10px] font-semibold rounded mb-2">
                    AND
                  </span>
                )}
                <div className="space-y-2">
                  {ident.values.map((val, valIdx) => (
                    <div key={valIdx} className="flex items-center gap-2">
                      {valIdx === 0 ? (
                        <FloatingDropdown
                          label="Identifier Type"
                          options={CLIENT_IDENTIFIER_TYPES.map((t) => ({
                            value: t.value,
                            label: t.label,
                          }))}
                          value={ident.type}
                          onChange={(v) => updateIdentifierType(idx, v)}
                          className="w-48"
                        />
                      ) : (
                        <span className="w-48 flex items-center justify-center">
                          <span className="px-2 py-0.5 bg-blue-50 text-blue-600 text-[10px] font-semibold rounded">
                            OR
                          </span>
                        </span>
                      )}
                      <div className="flex-1">
                        <FloatingInput
                          label="Value"
                          value={val}
                          onChange={(v) => updateIdentifierValue(idx, valIdx, v)}
                          disabled={disabled}
                        />
                      </div>
                      <button
                        onClick={() => addOrValue(idx)}
                        className="text-orange-500 hover:text-orange-600 text-sm font-bold"
                        title="Add OR value"
                      >
                        +
                      </button>
                      {(values.identifiers.length > 1 || ident.values.length > 1) && (
                        <button
                          onClick={() =>
                            ident.values.length > 1
                              ? removeOrValue(idx, valIdx)
                              : removeIdentifier(idx)
                          }
                          className="text-orange-400 hover:text-red-500 text-sm"
                        >
                          ×
                        </button>
                      )}
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </div>
          <button
            onClick={addIdentifier}
            className="mt-3 text-xs text-orange-500 hover:text-orange-600 font-medium"
          >
            + Additional Client Identifier
          </button>
        </div>

        <div className="border border-border rounded-lg p-5">
          <h4 className="text-sm font-semibold text-gray-800 mb-4">Tags</h4>
          <TagsField
            value={values.tags}
            onChange={(tags) => update({ tags })}
            disabled={disabled}
          />
        </div>
      </div>
    </fieldset>
  )
}

import { useState } from 'react'
import { Dropdown } from '@/shared/components/Dropdown'
import { TagsField } from '@/shared/components/TagsField'
import { CLIENT_IDENTIFIER_TYPES } from '@/features/policies/config'
import { policyFormOptions } from '@/data/policies/options'

export function ClientWorkloadForm() {
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [identifiers, setIdentifiers] = useState<{ type: string; values: string[] }[]>([
    { type: '', values: [''] },
  ])

  function addIdentifier() {
    setIdentifiers([...identifiers, { type: '', values: [''] }])
  }

  function updateIdentifierType(idx: number, type: string) {
    setIdentifiers(identifiers.map((ident, i) => i === idx ? { ...ident, type } : ident))
  }

  function updateIdentifierValue(idx: number, valIdx: number, val: string) {
    setIdentifiers(identifiers.map((ident, i) => {
      if (i !== idx) return ident
      const values = [...ident.values]
      values[valIdx] = val
      return { ...ident, values }
    }))
  }

  function addOrValue(idx: number) {
    setIdentifiers(identifiers.map((ident, i) => {
      if (i !== idx) return ident
      return { ...ident, values: [...ident.values, ''] }
    }))
  }

  function removeOrValue(idx: number, valIdx: number) {
    setIdentifiers(identifiers.map((ident, i) => {
      if (i !== idx) return ident
      const values = ident.values.filter((_, vi) => vi !== valIdx)
      return { ...ident, values: values.length === 0 ? [''] : values }
    }))
  }

  function removeIdentifier(idx: number) {
    if (identifiers.length > 1) {
      setIdentifiers(identifiers.filter((_, i) => i !== idx))
    }
  }

  return (
    <div className="space-y-5">
      <div>
        <label className="block text-xs font-medium text-gray-700 mb-1.5">Name</label>
        <input
          type="text"
          placeholder="e.g. Payments Service"
          value={name}
          onChange={(e) => setName(e.target.value)}
          className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-gray-300 focus:border-gray-400"
        />
      </div>

      <div>
        <label className="block text-xs font-medium text-gray-700 mb-1.5">Description</label>
        <textarea
          placeholder="Handles payment processing for production traffic"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          rows={2}
          className="w-full px-3 py-2 border border-border rounded-md text-sm resize-y focus:outline-none focus:ring-1 focus:ring-gray-300 focus:border-gray-400"
        />
      </div>

      <div>
        <h4 className="text-xs font-medium text-gray-800 mb-3">Client Identification</h4>
        <div className="space-y-3">
          {identifiers.map((ident, idx) => (
            <div key={idx}>
              {idx > 0 && (
                <span className="inline-block px-2 py-0.5 bg-gray-100 text-gray-600 text-[10px] font-semibold rounded mb-2">AND</span>
              )}
              <div className="space-y-2">
                {ident.values.map((val, valIdx) => (
                  <div key={valIdx} className="flex items-center gap-2">
                    {valIdx === 0 ? (
                      <Dropdown
                        options={CLIENT_IDENTIFIER_TYPES.map((t) => ({ value: t.value, label: t.label }))}
                        value={ident.type}
                        onChange={(v) => updateIdentifierType(idx, v)}
                        placeholder="Please select one"
                        className="w-48"
                      />
                    ) : (
                      <span className="w-48 flex items-center justify-center">
                        <span className="px-2 py-0.5 bg-blue-50 text-blue-600 text-[10px] font-semibold rounded">OR</span>
                      </span>
                    )}
                    <input
                      type="text"
                      placeholder="Value"
                      value={val}
                      onChange={(e) => updateIdentifierValue(idx, valIdx, e.target.value)}
                      className="flex-1 px-3 py-2 border border-border rounded-md text-xs focus:outline-none focus:ring-1 focus:ring-gray-300"
                    />
                    <button onClick={() => addOrValue(idx)} className="text-orange-500 hover:text-orange-600 text-sm font-bold" title="Add OR value">+</button>
                    {(identifiers.length > 1 || ident.values.length > 1) && (
                      <button onClick={() => ident.values.length > 1 ? removeOrValue(idx, valIdx) : removeIdentifier(idx)} className="text-orange-400 hover:text-red-500 text-sm">×</button>
                    )}
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
        <button onClick={addIdentifier} className="mt-3 text-xs text-orange-500 hover:text-orange-600 font-medium">
          + Additional Client Identifier
        </button>
      </div>

      <div>
        <h4 className="text-xs font-medium text-gray-800 mb-3">Standalone Certificate Authority</h4>
        <div className="flex gap-4">
          <Dropdown
            options={policyFormOptions.standaloneCAs.map((ca) => ({ value: ca.id, label: ca.name }))}
            value=""
            onChange={() => {}}
            placeholder="Select Standalone CA"
            className="w-56"
          />
          <p className="text-[11px] text-gray-500 flex-1">
            If a Standalone Certificate Authority (CA) is selected, Dipole will use this CA for TLS Decrypt functionality.
          </p>
        </div>
      </div>

      <div>
        <label className="block text-xs font-medium text-gray-700 mb-1.5">Tags</label>
        <TagsField />
      </div>
    </div>
  )
}

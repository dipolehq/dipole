import { useState } from 'react'
import { Dropdown } from '@/shared/components/Dropdown'
import { TagsField } from '@/shared/components/TagsField'
import { TRUST_PROVIDER_TYPES } from '@/features/policies/config'
import { TrustProviderConfig } from '@/features/policies/components/tabs/trust-provider/TrustProviderConfig'

export function TrustProviderForm() {
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [trustProviderType, setTrustProviderType] = useState('')

  return (
    <div className="space-y-5">
      <div>
        <label className="block text-xs font-medium text-gray-700 mb-1.5">Name</label>
        <input
          type="text"
          placeholder="Name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-gray-300"
        />
      </div>

      <div>
        <label className="block text-xs font-medium text-gray-700 mb-1.5">Description</label>
        <textarea
          placeholder="Description (Optional)"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          rows={2}
          className="w-full px-3 py-2 border border-border rounded-md text-sm resize-y focus:outline-none focus:ring-1 focus:ring-gray-300"
        />
      </div>

      <div>
        <h4 className="text-xs font-semibold text-gray-800 uppercase tracking-wide mb-2">Trust Provider</h4>
        <Dropdown
          options={TRUST_PROVIDER_TYPES.map((t) => ({ value: t.value, label: t.label }))}
          value={trustProviderType}
          onChange={setTrustProviderType}
          placeholder=""
        />
      </div>

      {trustProviderType && (
        <TrustProviderConfig type={trustProviderType} />
      )}

      <div>
        <label className="block text-xs font-medium text-gray-700 mb-1.5">Tags</label>
        <TagsField />
      </div>
    </div>
  )
}

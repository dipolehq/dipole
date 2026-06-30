import { TagsField } from '@/shared/components/TagsField'
import { Dropdown } from '@/shared/components/Dropdown'

interface DetailsTabProps {
  name: string
  description: string
  isActive: boolean
  onNameChange: (value: string) => void
  onDescriptionChange: (value: string) => void
  onActiveChange: (value: boolean) => void
}

export function DetailsTab({ name, description, isActive, onNameChange, onDescriptionChange, onActiveChange }: DetailsTabProps) {
  const statusValue = isActive ? 'active' : 'inactive'

  function handleStatusChange(val: string) {
    onActiveChange(val === 'active')
  }

  return (
    <div className="space-y-6">
      <h3 className="text-sm font-semibold text-gray-900">Access Policy Details</h3>

      <div>
        <label className="block text-xs font-medium text-gray-700 mb-1.5">Status</label>
        <Dropdown
          options={[
            { value: 'active', label: 'Active' },
            { value: 'inactive', label: 'Inactive' },
            { value: 'deprecated', label: 'Deprecated' },
          ]}
          value={statusValue}
          onChange={handleStatusChange}
          placeholder="Select status..."
        />
      </div>

      <div>
        <label className="block text-xs font-medium text-gray-700 mb-1.5">Access Policy Name</label>
        <input
          type="text"
          placeholder="e.g. Production EC2 to Stripe API"
          value={name}
          onChange={(e) => onNameChange(e.target.value)}
          className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-gray-300 focus:border-gray-400"
        />
      </div>

      <div>
        <label className="block text-xs font-medium text-gray-700 mb-1.5">Description</label>
        <textarea
          placeholder="Allows Production EC2 instances to access the Stripe API using OAuth credentials"
          value={description}
          onChange={(e) => onDescriptionChange(e.target.value)}
          rows={3}
          className="w-full px-3 py-2 border border-border rounded-md text-sm resize-y focus:outline-none focus:ring-1 focus:ring-gray-300 focus:border-gray-400"
        />
      </div>

      <div>
        <label className="block text-xs font-medium text-gray-700 mb-1.5">Tags</label>
        <TagsField />
      </div>
    </div>
  )
}

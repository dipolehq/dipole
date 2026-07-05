import { TagsField } from '@/shared/components/TagsField'
import { FloatingInput, FloatingTextarea, FloatingDropdown } from '@/shared/components/FloatingInput'

interface DetailsTabProps {
  name: string
  description: string
  isActive: boolean
  onNameChange: (value: string) => void
  onDescriptionChange: (value: string) => void
  onActiveChange: (value: boolean) => void
}

export function DetailsTab({ name, description, isActive, onNameChange, onDescriptionChange, onActiveChange }: DetailsTabProps) {
  // Map boolean to string for display, but empty string means "not yet selected"
  const statusValue = isActive ? 'active' : ''

  function handleStatusChange(val: string) {
    onActiveChange(val === 'active')
  }

  return (
    <div className="space-y-5">
      <div className="border border-border rounded-lg p-5">
        <h4 className="text-sm font-semibold text-gray-800 mb-4">Details</h4>
        <div className="space-y-4">
          <FloatingDropdown
            label="Status"
            options={[
              { value: 'active', label: 'Active' },
              { value: 'inactive', label: 'Inactive' },
            ]}
            value={statusValue}
            onChange={handleStatusChange}
          />
          <FloatingInput
            label="Access Policy Name"
            value={name}
            onChange={onNameChange}
            required
          />
          <FloatingTextarea
            label="Description (Optional)"
            value={description}
            onChange={onDescriptionChange}
            rows={3}
          />
        </div>
      </div>

      <div className="border border-border rounded-lg p-5">
        <h4 className="text-sm font-semibold text-gray-800 mb-4">Tags</h4>
        <TagsField />
      </div>
    </div>
  )
}

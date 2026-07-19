import { TagsField } from '@/shared/components/TagsField'
import {
  FloatingInput,
  FloatingTextarea,
  FloatingDropdown,
} from '@/shared/components/FloatingInput'
import type { FormTag } from '@/shared/components/forms/types'

interface DetailsTabProps {
  name: string
  description: string
  isActive: boolean
  tags: FormTag[]
  onNameChange: (value: string) => void
  onDescriptionChange: (value: string) => void
  onActiveChange: (value: boolean) => void
  onTagsChange: (tags: FormTag[]) => void
  disabled?: boolean
}

export function DetailsTab({
  name,
  description,
  isActive,
  tags,
  onNameChange,
  onDescriptionChange,
  onActiveChange,
  onTagsChange,
  disabled = false,
}: DetailsTabProps) {
  const statusValue = isActive ? 'active' : ''

  function handleStatusChange(val: string) {
    onActiveChange(val === 'active')
  }

  const disabledClass = disabled ? 'opacity-60 pointer-events-none' : ''

  return (
    <fieldset disabled={disabled} className={disabledClass}>
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
              disabled={disabled}
            />
            <FloatingInput label="Access Policy Name" value={name} onChange={onNameChange} required disabled={disabled} />
            <FloatingTextarea
              label="Description (Optional)"
              value={description}
              onChange={onDescriptionChange}
              rows={3}
              disabled={disabled}
            />
          </div>
        </div>

        <div className="border border-border rounded-lg p-5">
          <h4 className="text-sm font-semibold text-gray-800 mb-4">Tags</h4>
          <TagsField value={tags} onChange={onTagsChange} disabled={disabled} />
        </div>
      </div>
    </fieldset>
  )
}

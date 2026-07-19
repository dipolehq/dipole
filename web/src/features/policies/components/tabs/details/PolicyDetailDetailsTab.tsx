import { TagsField } from '@/shared/components/TagsField'
import {
  FloatingInput,
  FloatingTextarea,
  FloatingDropdown,
} from '@/shared/components/FloatingInput'
import type { FormTag } from '@/shared/components/forms/types'

interface PolicyDetailDetailsTabProps {
  name: string
  description: string
  status: string
  tags: FormTag[]
  disabled: boolean
  onNameChange: (value: string) => void
  onDescriptionChange: (value: string) => void
  onStatusChange: (value: string) => void
  onTagsChange: (tags: FormTag[]) => void
}

export function PolicyDetailDetailsTab({
  name,
  description,
  status,
  tags,
  disabled,
  onNameChange,
  onDescriptionChange,
  onStatusChange,
  onTagsChange,
}: PolicyDetailDetailsTabProps) {
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
              { value: 'deprecated', label: 'Deprecated' },
            ]}
            value={status}
            onChange={onStatusChange}
            disabled={disabled}
          />
          <FloatingInput
            label="Access Policy Name"
            value={name}
            onChange={onNameChange}
            disabled={disabled}
            required
          />
          <FloatingTextarea
            label="Description (Optional)"
            value={description}
            onChange={onDescriptionChange}
            disabled={disabled}
            rows={3}
          />
        </div>
      </div>

      <div className="border border-border rounded-lg p-5">
        <h4 className="text-sm font-semibold text-gray-800 mb-4">Tags</h4>
        {disabled && tags.length === 0 ? (
          <div className="px-3 py-2 border border-border rounded-md text-sm text-gray-500 bg-gray-50">
            No tags
          </div>
        ) : (
          <TagsField value={tags} onChange={onTagsChange} disabled={disabled} />
        )}
      </div>
    </div>
  )
}

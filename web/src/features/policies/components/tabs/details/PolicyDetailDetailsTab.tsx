import { TagsField } from '@/shared/components/TagsField'
import { FloatingInput, FloatingTextarea, FloatingDropdown } from '@/shared/components/FloatingInput'

interface PolicyDetailDetailsTabProps {
  name: string
  description: string
  status: string
  policyId: string
  createdAt: string
  createdBy: string
  lastModifiedAt: string
  lastModifiedBy: string
  disabled: boolean
  onNameChange: (value: string) => void
  onDescriptionChange: (value: string) => void
  onStatusChange: (value: string) => void
}

export function PolicyDetailDetailsTab({
  name,
  description,
  status,
  policyId,
  createdAt,
  createdBy,
  lastModifiedAt,
  lastModifiedBy,
  disabled,
  onNameChange,
  onDescriptionChange,
  onStatusChange,
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
        <h4 className="text-sm font-semibold text-gray-800 mb-4">Metadata</h4>
        <div className="space-y-4">
          <FloatingInput label="Policy ID" value={policyId} disabled />
          <FloatingInput label="Created At" value={`${new Date(createdAt).toLocaleString()} by ${createdBy}`} disabled />
          <FloatingInput label="Last Modified" value={`${new Date(lastModifiedAt).toLocaleString()} by ${lastModifiedBy}`} disabled />
        </div>
      </div>

      <div className="border border-border rounded-lg p-5">
        <h4 className="text-sm font-semibold text-gray-800 mb-4">Tags</h4>
        {disabled ? (
          <div className="px-3 py-2 border border-border rounded-md text-sm text-gray-500 bg-gray-50">No tags</div>
        ) : (
          <TagsField />
        )}
      </div>
    </div>
  )
}

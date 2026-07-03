import { TagsField } from '@/shared/components/TagsField'
import { Dropdown } from '@/shared/components/Dropdown'

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

function formatTimestamp(iso: string): string {
  const date = new Date(iso)
  return date.toLocaleString('en-IN', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    hour12: true,
  })
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
    <div className="space-y-6">
      {/* Metadata section */}
      <div className="grid grid-cols-2 gap-4 p-4 bg-gray-50 rounded-md border border-border">
        <div>
          <label className="block text-[10px] font-semibold text-gray-500 uppercase tracking-wide mb-0.5">Policy ID</label>
          <div className="flex items-center gap-2">
            <span className="text-xs text-gray-700 font-mono">{policyId}</span>
            <button
              onClick={() => navigator.clipboard.writeText(policyId)}
              className="text-gray-400 hover:text-gray-600"
              title="Copy ID"
            >
              <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><rect x="9" y="9" width="13" height="13" rx="2" ry="2" strokeWidth="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" strokeWidth="2"/></svg>
            </button>
          </div>
        </div>
        <div>
          <label className="block text-[10px] font-semibold text-gray-500 uppercase tracking-wide mb-0.5">Created At</label>
          <span className="text-xs text-gray-700">{formatTimestamp(createdAt)}</span>
          <span className="text-[10px] text-gray-400 ml-1">by {createdBy}</span>
        </div>
        <div>
          <label className="block text-[10px] font-semibold text-gray-500 uppercase tracking-wide mb-0.5">Last Modified</label>
          <span className="text-xs text-gray-700">{formatTimestamp(lastModifiedAt)}</span>
          <span className="text-[10px] text-gray-400 ml-1">by {lastModifiedBy}</span>
        </div>
      </div>

      {/* Status */}
      <div>
        <label className="block text-xs font-medium text-gray-700 mb-1.5">Status</label>
        {disabled ? (
          <div className="px-3 py-2 border border-border rounded-md text-sm text-gray-600 bg-gray-50 capitalize">{status}</div>
        ) : (
          <Dropdown
            options={[
              { value: 'active', label: 'Active' },
              { value: 'inactive', label: 'Inactive' },
              { value: 'deprecated', label: 'Deprecated' },
            ]}
            value={status}
            onChange={onStatusChange}
            placeholder="Select status..."
          />
        )}
      </div>

      {/* Name */}
      <div>
        <label className="block text-xs font-medium text-gray-700 mb-1.5">Access Policy Name</label>
        {disabled ? (
          <div className="px-3 py-2 border border-border rounded-md text-sm text-gray-700 bg-gray-50">{name}</div>
        ) : (
          <input
            type="text"
            placeholder="e.g. Production EC2 to Stripe API"
            value={name}
            onChange={(e) => onNameChange(e.target.value)}
            className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-gray-300 focus:border-gray-400"
          />
        )}
      </div>

      {/* Description */}
      <div>
        <label className="block text-xs font-medium text-gray-700 mb-1.5">Description</label>
        {disabled ? (
          <div className="px-3 py-2 border border-border rounded-md text-sm text-gray-700 bg-gray-50 min-h-[60px]">{description || '—'}</div>
        ) : (
          <textarea
            placeholder="Allows Production EC2 instances to access the Stripe API using OAuth credentials"
            value={description}
            onChange={(e) => onDescriptionChange(e.target.value)}
            rows={3}
            className="w-full px-3 py-2 border border-border rounded-md text-sm resize-y focus:outline-none focus:ring-1 focus:ring-gray-300 focus:border-gray-400"
          />
        )}
      </div>

      {/* Tags */}
      <div>
        <label className="block text-xs font-medium text-gray-700 mb-1.5">Tags</label>
        {disabled ? (
          <div className="px-3 py-2 border border-border rounded-md text-sm text-gray-500 bg-gray-50">No tags</div>
        ) : (
          <TagsField />
        )}
      </div>
    </div>
  )
}

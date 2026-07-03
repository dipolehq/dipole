interface EntityMetadataProps {
  entityId: string
  entityLabel: string
  createdAt: string
  createdBy: string
  lastModifiedAt?: string
  lastModifiedBy?: string
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

export function EntityMetadata({
  entityId,
  entityLabel,
  createdAt,
  createdBy,
  lastModifiedAt,
  lastModifiedBy,
}: EntityMetadataProps) {
  return (
    <div className="grid grid-cols-2 gap-4 p-4 bg-gray-50 rounded-md border border-border">
      <div>
        <label className="block text-[10px] font-semibold text-gray-500 uppercase tracking-wide mb-0.5">
          {entityLabel} ID
        </label>
        <div className="flex items-center gap-2">
          <span className="text-xs text-gray-700 font-mono">{entityId}</span>
          <button
            type="button"
            onClick={() => navigator.clipboard.writeText(entityId)}
            className="text-gray-400 hover:text-gray-600"
            title="Copy ID"
          >
            <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <rect x="9" y="9" width="13" height="13" rx="2" ry="2" strokeWidth="2" />
              <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" strokeWidth="2" />
            </svg>
          </button>
        </div>
      </div>
      <div>
        <label className="block text-[10px] font-semibold text-gray-500 uppercase tracking-wide mb-0.5">
          Created At
        </label>
        <span className="text-xs text-gray-700">{formatTimestamp(createdAt)}</span>
        <span className="text-[10px] text-gray-400 ml-1">by {createdBy}</span>
      </div>
      {lastModifiedAt && (
        <div>
          <label className="block text-[10px] font-semibold text-gray-500 uppercase tracking-wide mb-0.5">
            Last Modified
          </label>
          <span className="text-xs text-gray-700">{formatTimestamp(lastModifiedAt)}</span>
          {lastModifiedBy && <span className="text-[10px] text-gray-400 ml-1">by {lastModifiedBy}</span>}
        </div>
      )}
    </div>
  )
}

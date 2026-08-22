import { useState, type ReactNode } from 'react'
import { Pencil, Trash2 } from 'lucide-react'
import { FloatingInput } from '@/shared/components/FloatingInput'
import { DeleteConfirmDialog } from '@/shared/components/ui/DeleteConfirmDialog'

interface DetailPageMetadata {
  id: string
  createdAt: string
  lastModifiedAt?: string
}

interface DetailPageProps {
  /** Resource display name (shown in toolbar and delete dialog) */
  name: string
  /** Resource status — 'active' or 'inactive' */
  status: string
  /** Metadata for the right sidebar */
  metadata: DetailPageMetadata
  /** Resource type label for the delete dialog (e.g., "client workload") */
  resourceType: string
  /** The form/content for the main area */
  children: ReactNode
  /** Whether the page is in edit mode */
  isEditing: boolean
  /** Called when the Edit button is clicked */
  onEdit: () => void
  /** Called when Save is clicked in edit mode */
  onSave: () => void
  /** Called when Cancel is clicked in edit mode */
  onCancel: () => void
  /** Called when delete is confirmed */
  onDelete: () => void
  /** Whether a save operation is in progress */
  isSaving?: boolean
  /** Whether a delete operation is in progress */
  isDeleting?: boolean
  /** Error message to display below toolbar */
  error?: string | null
  /** Additional metadata fields to render below the standard ones */
  extraMetadata?: ReactNode
}

/**
 * Reusable detail page layout — consistent with DataTable list pages.
 *
 * Structure:
 * - h-[44px] header bar (PageHeader with breadcrumb + switcher)
 * - Toolbar: resource name + status left, Edit/Delete buttons right
 * - Bordered content box: children (form) left, metadata sidebar right
 * - DeleteConfirmDialog for safe deletion
 */
export function DetailPage({
  name,
  status,
  metadata,
  resourceType,
  children,
  isEditing,
  onEdit,
  onSave,
  onCancel,
  onDelete,
  isSaving = false,
  isDeleting = false,
  error = null,
  extraMetadata,
}: DetailPageProps) {
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)

  function handleDeleteConfirm() {
    setDeleteDialogOpen(false)
    onDelete()
  }

  return (
    <div className="flex flex-col h-full">

      {/* Toolbar — matches DataTable toolbar positioning */}
      <div className="flex items-center justify-between gap-3 px-4 py-3 shrink-0">
        <div className="flex items-center gap-3">
          <h2 className="text-sm font-medium text-gray-900">{name || 'Untitled'}</h2>
          <span className={`text-xs font-medium px-2 py-0.5 rounded ${status === 'active' ? 'bg-green-50 text-green-700' : 'bg-gray-100 text-gray-500'}`}>
            {status === 'active' ? 'Active' : 'Inactive'}
          </span>
        </div>
        <div className="flex items-center gap-2">
          {isEditing ? (
            <>
              <button
                type="button"
                onClick={onSave}
                disabled={isSaving}
                className="inline-flex items-center gap-2 h-9 px-4 rounded-md border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors disabled:opacity-50"
              >
                {isSaving ? 'Saving…' : 'Save'}
              </button>
              <button
                type="button"
                onClick={onCancel}
                className="inline-flex items-center gap-2 h-9 px-4 rounded-md border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors"
              >
                Cancel
              </button>
            </>
          ) : (
            <>
              <button
                type="button"
                onClick={onEdit}
                className="inline-flex items-center gap-2 h-9 px-4 rounded-md border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors"
              >
                <Pencil size={14} strokeWidth={2} className="opacity-60" />
                Edit
              </button>
              <button
                type="button"
                onClick={() => setDeleteDialogOpen(true)}
                disabled={isDeleting}
                className="inline-flex items-center gap-2 h-9 px-4 rounded-md border border-red-200 bg-white text-sm font-medium text-red-600 hover:bg-red-50 transition-colors disabled:opacity-50"
              >
                <Trash2 size={14} strokeWidth={2} className="opacity-60" />
                Delete
              </button>
            </>
          )}
        </div>
      </div>

      {error && (
        <div className="mx-4 mb-2 px-4 py-2 bg-red-50 border border-red-200 rounded-md text-sm text-red-700">
          {error}
        </div>
      )}

      {/* Content box — same border treatment as DataTable's table container */}
      <div className="flex-1 min-h-0 mx-4 border border-border bg-white overflow-y-auto">
        <div className="grid grid-cols-[1fr_380px] h-full">
          {/* Main content area */}
          <div className="px-6 py-5 overflow-y-auto border-r border-border">
            {children}
          </div>

          {/* Metadata sidebar */}
          <div className="px-5 py-5 overflow-y-auto">
            <div className="space-y-4">
              <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider">Metadata</h4>
              <FloatingInput label="ID" value={metadata.id} disabled />
              <FloatingInput
                label="Created"
                value={new Date(metadata.createdAt).toLocaleString('en-IN', { dateStyle: 'medium', timeStyle: 'short' })}
                disabled
              />
              <FloatingInput
                label="Last Modified"
                value={metadata.lastModifiedAt ? new Date(metadata.lastModifiedAt).toLocaleString('en-IN', { dateStyle: 'medium', timeStyle: 'short' }) : '—'}
                disabled
              />
              {extraMetadata}
            </div>
          </div>
        </div>
      </div>

      <DeleteConfirmDialog
        open={deleteDialogOpen}
        onClose={() => setDeleteDialogOpen(false)}
        onConfirm={handleDeleteConfirm}
        resourceName={name || 'Untitled'}
        resourceId={metadata.id}
        resourceType={resourceType}
        loading={isDeleting}
      />
    </div>
  )
}

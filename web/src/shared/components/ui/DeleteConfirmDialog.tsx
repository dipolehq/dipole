import { useState, useRef, useEffect } from 'react'
import { AlertTriangle, X } from 'lucide-react'

interface DeleteConfirmDialogProps {
  open: boolean
  onClose: () => void
  onConfirm: () => void
  /** The resource name shown in the dialog (e.g., "CI/CD Pipeline") */
  resourceName: string
  /** The resource ID the user must type to confirm deletion */
  resourceId: string
  /** Optional resource type label (e.g., "client workload") */
  resourceType?: string
  /** Whether the delete action is in progress */
  loading?: boolean
}

/**
 * Reusable destructive-action confirmation dialog.
 * Requires the user to type the resource ID to confirm deletion.
 * Used across all detail pages for consistent delete UX.
 */
export function DeleteConfirmDialog({
  open,
  onClose,
  onConfirm,
  resourceName,
  resourceId,
  resourceType = 'resource',
  loading = false,
}: DeleteConfirmDialogProps) {
  const [confirmInput, setConfirmInput] = useState('')
  const inputRef = useRef<HTMLInputElement>(null)

  const canConfirm = confirmInput === resourceId

  useEffect(() => {
    if (open) {
      setConfirmInput('')
      setTimeout(() => inputRef.current?.focus(), 100)
    }
  }, [open])

  if (!open) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div className="absolute inset-0 bg-black/40" onClick={onClose} />

      {/* Dialog */}
      <div className="relative bg-white rounded-lg shadow-xl border border-border w-full max-w-md mx-4">
        {/* Header */}
        <div className="flex items-center justify-between px-5 py-4 border-b border-border">
          <div className="flex items-center gap-2.5">
            <div className="w-8 h-8 rounded-full bg-red-50 flex items-center justify-center">
              <AlertTriangle size={16} className="text-red-500" />
            </div>
            <h3 className="text-sm font-semibold text-gray-900">Delete {resourceType}</h3>
          </div>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600 transition-colors"
          >
            <X size={16} />
          </button>
        </div>

        {/* Body */}
        <div className="px-5 py-4 space-y-4">
          <p className="text-sm text-gray-600">
            This action is <span className="font-medium text-gray-900">permanent and irreversible</span>.
            Deleting <span className="font-medium text-gray-900">{resourceName}</span> will
            remove it and all associated data immediately. This cannot be undone.
          </p>

          <div className="bg-gray-50 border border-border rounded-md px-3 py-2.5">
            <p className="text-xs text-gray-500 mb-1">To confirm, type the resource ID below:</p>
            <code className="text-xs font-mono text-gray-700 select-all">{resourceId}</code>
          </div>

          <input
            ref={inputRef}
            type="text"
            value={confirmInput}
            onChange={(e) => setConfirmInput(e.target.value)}
            placeholder="Paste resource ID here"
            className="w-full h-9 px-3 text-sm border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-red-300 focus:border-red-400 placeholder:text-gray-400"
          />
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end gap-2 px-5 py-3 border-t border-border bg-gray-50 rounded-b-lg">
          <button
            type="button"
            onClick={onClose}
            className="h-9 px-4 text-sm font-medium text-gray-700 border border-border rounded-md hover:bg-white transition-colors"
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={onConfirm}
            disabled={!canConfirm || loading}
            className="h-9 px-4 text-sm font-medium text-white bg-red-600 rounded-md hover:bg-red-700 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          >
            {loading ? 'Deleting…' : 'Delete permanently'}
          </button>
        </div>
      </div>
    </div>
  )
}

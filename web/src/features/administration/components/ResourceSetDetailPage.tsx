import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { Pencil, Trash2 } from 'lucide-react'
import { PageHeader } from '@/shared/components/PageHeader'
import { FloatingInput, FloatingTextarea } from '@/shared/components/FloatingInput'
import { DeleteConfirmDialog } from '@/shared/components/ui/DeleteConfirmDialog'
import { useResourceSet, useSaveResourceSet, useDeleteResourceSet } from '@/features/administration/hooks/useResourceSets'
import { apiErrorMessage } from '@/features/administration/api/resourceSetsApi'

export function ResourceSetDetailPage() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()

  const { data: resourceSet, isLoading } = useResourceSet(id)
  const saveMutation = useSaveResourceSet(id ?? '')
  const deleteMutation = useDeleteResourceSet()

  const [isEditing, setIsEditing] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)

  const [displayName, setDisplayName] = useState('')
  const [description, setDescription] = useState('')
  const [initialized, setInitialized] = useState(false)

  if (resourceSet && !initialized) {
    setDisplayName(resourceSet.name)
    setDescription(resourceSet.description)
    setInitialized(true)
  }

  if (initialized && resourceSet && resourceSet.id !== id) {
    setInitialized(false)
  }

  if (!id || (!isLoading && resourceSet === null)) {
    return (
      <div className="flex flex-col h-full">
        <div className="flex items-center justify-between px-4 shrink-0 h-[44px] border-b border-border bg-white">
          <PageHeader />
        </div>
        <div className="flex-1 flex items-center justify-center">
          <p className="text-sm text-gray-400">Resource Set not found.</p>
        </div>
      </div>
    )
  }

  const disabled = !isEditing

  function handleEdit() {
    setError(null)
    setIsEditing(true)
  }

  async function handleSave() {
    setError(null)
    try {
      await saveMutation.mutateAsync({ name: displayName, description })
      setIsEditing(false)
    } catch (err) {
      setError(apiErrorMessage(err, 'Failed to save resource set'))
    }
  }

  function handleCancel() {
    setError(null)
    if (resourceSet) {
      setDisplayName(resourceSet.name)
      setDescription(resourceSet.description)
    }
    setIsEditing(false)
  }

  async function handleDelete() {
    if (!id) return
    try {
      await deleteMutation.mutateAsync(id)
      navigate('/administration/resource-sets')
    } catch (err) {
      setError(apiErrorMessage(err, 'Failed to delete resource set'))
    }
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-4 shrink-0 h-[44px] border-b border-border bg-white">
        <PageHeader />
      </div>

      <div className="flex items-center justify-between gap-3 px-4 py-3 shrink-0">
        <div className="flex items-center gap-3">
          <h2 className="text-sm font-medium text-gray-900">{displayName || 'Untitled'}</h2>
        </div>
        <div className="flex items-center gap-2">
          {isEditing ? (
            <>
              <button type="button" onClick={handleSave} disabled={saveMutation.isPending} className="inline-flex items-center gap-2 h-9 px-4 rounded-md border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors disabled:opacity-50">
                {saveMutation.isPending ? 'Saving…' : 'Save'}
              </button>
              <button type="button" onClick={handleCancel} className="inline-flex items-center gap-2 h-9 px-4 rounded-md border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors">
                Cancel
              </button>
            </>
          ) : (
            <>
              <button type="button" onClick={handleEdit} className="inline-flex items-center gap-2 h-9 px-4 rounded-md border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors">
                <Pencil size={14} strokeWidth={2} className="opacity-60" /> Edit
              </button>
              <button type="button" onClick={() => setDeleteDialogOpen(true)} className="inline-flex items-center gap-2 h-9 px-4 rounded-md border border-red-200 bg-white text-sm font-medium text-red-600 hover:bg-red-50 transition-colors">
                <Trash2 size={14} strokeWidth={2} className="opacity-60" /> Delete
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

      <div className="flex-1 min-h-0 mx-4 border border-border bg-white overflow-y-auto">
        <div className="grid grid-cols-[1fr_380px] h-full">
          <div className="px-6 py-5 overflow-y-auto border-r border-border">
            <fieldset disabled={disabled} className={disabled ? 'opacity-60 pointer-events-none' : ''}>
              <div className="space-y-6">
                <div>
                  <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider mb-4">Resource Set Details</h4>
                  <div className="space-y-4">
                    <FloatingInput label="Display Name" value={displayName} onChange={setDisplayName} disabled={disabled} required />
                    <FloatingTextarea label="Description (Optional)" value={description} onChange={setDescription} disabled={disabled} rows={3} />
                  </div>
                </div>
              </div>
            </fieldset>
          </div>

          <div className="px-5 py-5 overflow-y-auto">
            <div className="space-y-4">
              <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider">Metadata</h4>
              {resourceSet && (
                <>
                  <FloatingInput label="ID" value={resourceSet.id} disabled />
                  <FloatingInput label="Created" value={new Date(resourceSet.createdAt).toLocaleString('en-IN', { dateStyle: 'medium', timeStyle: 'short' })} disabled />
                  <FloatingInput label="Last Modified" value={new Date(resourceSet.updatedAt).toLocaleString('en-IN', { dateStyle: 'medium', timeStyle: 'short' })} disabled />
                </>
              )}
            </div>
          </div>
        </div>
      </div>

      <DeleteConfirmDialog
        open={deleteDialogOpen}
        onClose={() => setDeleteDialogOpen(false)}
        onConfirm={() => { setDeleteDialogOpen(false); handleDelete() }}
        resourceName={displayName || 'Untitled'}
        resourceId={id ?? ''}
        resourceType="resource set"
        loading={deleteMutation.isPending}
      />
    </div>
  )
}

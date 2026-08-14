import { useState, useEffect } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { Pencil, Trash2 } from 'lucide-react'
import { useQuery } from '@tanstack/react-query'
import { PageHeader } from '@/shared/components/PageHeader'
import { FloatingInput, FloatingTextarea, FloatingDropdown } from '@/shared/components/FloatingInput'
import { TagsField } from '@/shared/components/TagsField'
import { DeleteConfirmDialog } from '@/shared/components/ui/DeleteConfirmDialog'
import { useAgentController, useSaveAgentController, useDeleteAgentController } from '@/features/edge-components/hooks/useAgentControllers'
import { apiErrorMessage } from '@/features/edge-components/api/agentControllersApi'
import { fetchTrustProviders } from '@/features/policies/api/entityApi'
import type { FormTag } from '@/shared/components/forms/types'

export function AgentControllerDetailPage() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()

  const { data: controller, isLoading } = useAgentController(id)
  const saveMutation = useSaveAgentController(id ?? '')
  const deleteMutation = useDeleteAgentController()

  const [isEditing, setIsEditing] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)

  const [status, setStatus] = useState('')
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [trustProviderId, setTrustProviderId] = useState('')
  const [allowedTlsHostname, setAllowedTlsHostname] = useState('')
  const [tags, setTags] = useState<FormTag[]>([])

  // eslint-disable-next-line react-hooks/set-state-in-effect
  useEffect(() => {
    if (controller) {
      setStatus(controller.isActive ? 'active' : 'inactive')
      setName(controller.name)
      setDescription(controller.description ?? '')
      setTrustProviderId(controller.trustProviderId ?? '')
      setAllowedTlsHostname(controller.tlsHostname ?? '')
      setTags((controller.tags ?? []).map((t) => ({ key: t.key, value: t.value })))
    }
  }, [controller])

  const { data: tpData } = useQuery({
    queryKey: ['trust-providers', 'dropdown'],
    queryFn: () => fetchTrustProviders({ perPage: 100 }),
    staleTime: 60_000,
  })
  const trustProviderOptions = (tpData?.items ?? []).map((tp) => ({ value: tp.id, label: tp.name }))

  if (!id || (!isLoading && controller === null)) {
    return (
      <div className="flex flex-col h-full">
        <div className="flex items-center justify-between px-4 shrink-0 h-[44px] border-b border-border bg-white">
          <PageHeader />
        </div>
        <div className="flex-1 flex items-center justify-center">
          <p className="text-sm text-gray-400">Agent Controller not found.</p>
        </div>
      </div>
    )
  }

  const disabled = !isEditing
  const displayName = name || 'Untitled'

  function handleEdit() {
    setError(null)
    setIsEditing(true)
  }

  async function handleSave() {
    setError(null)
    try {
      await saveMutation.mutateAsync({
        name,
        description,
        isActive: status === 'active',
        trustProviderId: trustProviderId || undefined,
        tlsHostname: allowedTlsHostname || undefined,
        tags,
      })
      setIsEditing(false)
    } catch (err) {
      setError(apiErrorMessage(err, 'Failed to save agent controller'))
    }
  }

  function handleCancel() {
    setError(null)
    if (controller) {
      setStatus(controller.isActive ? 'active' : 'inactive')
      setName(controller.name)
      setDescription(controller.description ?? '')
      setTrustProviderId(controller.trustProviderId ?? '')
      setAllowedTlsHostname(controller.tlsHostname ?? '')
      setTags((controller.tags ?? []).map((t) => ({ key: t.key, value: t.value })))
    }
    setIsEditing(false)
  }

  async function handleDelete() {
    if (!id) return
    try {
      await deleteMutation.mutateAsync(id)
      navigate('/edge-components')
    } catch (err) {
      setError(apiErrorMessage(err, 'Failed to delete agent controller'))
    }
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-4 shrink-0 h-[44px] border-b border-border bg-white">
        <PageHeader />
      </div>

      <div className="flex items-center justify-between gap-3 px-4 py-3 shrink-0">
        <div className="flex items-center gap-3">
          <h2 className="text-sm font-medium text-gray-900">{displayName}</h2>
          <span className={`text-xs font-medium px-2 py-0.5 rounded ${status === 'active' ? 'bg-green-50 text-green-700' : 'bg-gray-100 text-gray-500'}`}>
            {status === 'active' ? 'Active' : 'Inactive'}
          </span>
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
                  <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider mb-4">Details</h4>
                  <div className="space-y-4">
                    <FloatingDropdown label="Status" options={[{ value: 'active', label: 'Active' }, { value: 'inactive', label: 'Inactive' }]} value={status} onChange={setStatus} disabled={disabled} />
                    <FloatingInput label="Name" value={name} onChange={setName} disabled={disabled} required />
                    <FloatingTextarea label="Description (Optional)" value={description} onChange={setDescription} disabled={disabled} rows={2} />
                  </div>
                </div>

                <div>
                  <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider mb-4">Configuration</h4>
                  <div className="space-y-4">
                    <FloatingDropdown label="Trust Provider" options={trustProviderOptions} value={trustProviderId} onChange={setTrustProviderId} disabled={disabled} />
                    <FloatingInput label="Allowed TLS Hostname (Optional)" value={allowedTlsHostname} onChange={setAllowedTlsHostname} disabled={disabled} />
                  </div>
                </div>

                <div>
                  <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider mb-4">Tags</h4>
                  {disabled && tags.length === 0 ? (
                    <div className="px-3 py-2 border border-border rounded-md text-sm text-gray-500 bg-gray-50">No tags</div>
                  ) : (
                    <TagsField value={tags} onChange={setTags} disabled={disabled} />
                  )}
                </div>
              </div>
            </fieldset>
          </div>

          <div className="px-5 py-5 overflow-y-auto">
            <div className="space-y-4">
              <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider">Metadata</h4>
              {controller && (
                <>
                  <FloatingInput label="ID" value={controller.id} disabled />
                  <FloatingInput label="Created" value={new Date(controller.createdAt).toLocaleString('en-IN', { dateStyle: 'medium', timeStyle: 'short' })} disabled />
                  {controller.deviceCode && (
                    <FloatingInput label="Device Code" value={controller.deviceCode} disabled />
                  )}
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
        resourceName={displayName}
        resourceId={id ?? ''}
        resourceType="agent controller"
        loading={deleteMutation.isPending}
      />
    </div>
  )
}

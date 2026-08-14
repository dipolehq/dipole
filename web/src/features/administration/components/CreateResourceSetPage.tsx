import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { PageHeader } from '@/shared/components/PageHeader'
import { FloatingInput, FloatingTextarea } from '@/shared/components/FloatingInput'
import { useCreateResourceSet } from '@/features/administration/hooks/useResourceSets'
import { apiErrorMessage } from '@/features/administration/api/resourceSetsApi'

export function CreateResourceSetPage() {
  const navigate = useNavigate()
  const createMutation = useCreateResourceSet()

  const [error, setError] = useState<string | null>(null)
  const [displayName, setDisplayName] = useState('')
  const [description, setDescription] = useState('')

  async function handleSave() {
    setError(null)
    try {
      await createMutation.mutateAsync({ name: displayName, description })
      navigate('/administration/resource-sets')
    } catch (err) {
      setError(apiErrorMessage(err, 'Failed to create resource set'))
    }
  }

  function handleCancel() {
    navigate('/administration/resource-sets')
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-4 shrink-0 h-[44px] border-b border-border bg-white">
        <PageHeader />
      </div>

      <div className="flex items-center justify-end gap-2 px-4 py-3 shrink-0">
        <button
          type="button"
          onClick={handleSave}
          disabled={createMutation.isPending}
          className="inline-flex items-center gap-2 h-9 px-4 rounded-md border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors disabled:opacity-50"
        >
          {createMutation.isPending ? 'Saving…' : 'Save'}
        </button>
        <button
          type="button"
          onClick={handleCancel}
          className="inline-flex items-center gap-2 h-9 px-4 rounded-md border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors"
        >
          Cancel
        </button>
      </div>

      {error && (
        <div className="mx-4 mb-2 px-4 py-2 bg-red-50 border border-red-200 rounded-md text-sm text-red-700">
          {error}
        </div>
      )}

      <div className="flex-1 min-h-0 mx-4 border border-border bg-white overflow-y-auto">
        <div className="grid grid-cols-[1fr_380px] h-full">
          <div className="px-6 py-5 overflow-y-auto border-r border-border">
            <div className="space-y-6">
              <div>
                <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider mb-4">Resource Set Details</h4>
                <div className="space-y-4">
                  <FloatingInput label="Display Name" value={displayName} onChange={setDisplayName} required />
                  <FloatingTextarea label="Description (Optional)" value={description} onChange={setDescription} rows={3} />
                </div>
              </div>
            </div>
          </div>

          <div className="px-5 py-5 overflow-y-auto">
            <div className="space-y-3">
              <p className="text-xs text-gray-600 leading-relaxed">
                <span className="font-semibold text-gray-800">Resource Sets</span> provide multi-tenancy within your organization. Each Resource Set is an isolated group of resources (policies, workloads, credentials) that can be managed independently.
              </p>
              <p className="text-xs text-gray-600 leading-relaxed">
                Assign <span className="font-semibold text-gray-800">Roles</span> to control which users can manage resources within this Resource Set.
              </p>
              <p className="text-xs text-gray-600 leading-relaxed">
                The default Resource Set contains all resources that are not explicitly assigned to another set.
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

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
      {error && (
        <div className="px-6 py-2 bg-red-50 border-b border-red-200 text-sm text-red-700 shrink-0">
          {error}
        </div>
      )}

      <div className="flex items-center justify-between px-4 shrink-0 h-[60px] border-b border-border bg-white">
        <PageHeader />
        <div className="flex items-center gap-3">
          <button type="button" onClick={handleSave} disabled={createMutation.isPending} className="px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white text-sm font-medium rounded-md transition-colors disabled:opacity-60">
            {createMutation.isPending ? 'Saving…' : 'Save'}
          </button>
          <button type="button" onClick={handleCancel} className="px-4 py-2 border border-gray-300 text-sm font-medium text-gray-600 hover:text-gray-900 hover:border-gray-400 rounded-md transition-colors">Cancel</button>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto bg-white">
        <div className="flex h-full">
          <div className="w-[60%] px-6 py-6 overflow-y-auto">
            <div className="space-y-5">
              <div className="border border-border rounded-lg p-5">
                <h4 className="text-sm font-semibold text-gray-800 mb-4">Resource Set Details</h4>
                <div className="space-y-4">
                  <FloatingInput label="Display Name" value={displayName} onChange={setDisplayName} required />
                  <FloatingTextarea label="Description (Optional)" value={description} onChange={setDescription} rows={3} />
                </div>
              </div>
            </div>
          </div>

          <div className="w-[40%] px-6 py-6 overflow-y-auto">
            <div className="border border-border rounded-lg p-5 sticky top-0">
              <p className="text-xs text-gray-700 leading-relaxed mb-3">
                <span className="font-semibold text-orange-600">Resource Sets</span> provide multi-tenancy within your organization. Each Resource Set is an isolated group of resources (policies, workloads, credentials) that can be managed independently.
              </p>
              <p className="text-xs text-gray-700 leading-relaxed mb-3">
                Assign <span className="font-semibold">Roles</span> to control which users can manage resources within this Resource Set.
              </p>
              <p className="text-xs text-gray-700 leading-relaxed">
                The default Resource Set contains all resources that are not explicitly assigned to another set.
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { PageHeader } from '@/shared/components/PageHeader'
import { PolicyForm } from '@/features/policies/components/PolicyForm'
import { usePolicy, useSavePolicy, useDeletePolicy } from '@/features/policies/hooks/usePolicy'
import type { PolicyFormMode, PolicyFormValues } from '@/features/policies/types/formValues'

export function PolicyDetailPage() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()

  const { data, isLoading, isError } = usePolicy(id)
  const saveMutation = useSavePolicy(id ?? '')
  const deleteMutation = useDeletePolicy()

  const [mode, setMode] = useState<PolicyFormMode>('view')
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)

  if (!id || isError || (!isLoading && data === null)) {
    return (
      <div className="flex flex-col h-full">
        <div className="flex items-center justify-between px-4 shrink-0 h-[60px] border-b border-border bg-white">
          <PageHeader />
        </div>
        <div className="flex-1 flex items-center justify-center flex-col">
          <p className="text-sm text-gray-500">Policy not found.</p>
          <button
            onClick={() => navigate('/access-policies')}
            className="mt-4 text-xs text-orange-500 hover:text-orange-600 font-medium"
          >
            ← Back to Access Policies
          </button>
        </div>
      </div>
    )
  }

  async function handleSubmit(values: PolicyFormValues) {
    await saveMutation.mutateAsync(values)
  }

  function handleCancel() {
    setMode('view')
  }

  async function handleDelete() {
    if (!id) return
    await deleteMutation.mutateAsync(id)
    navigate('/access-policies')
  }

  return (
    <>
      <PolicyForm
        mode={mode}
        isLoading={isLoading || !data}
        initialValues={data?.values}
        metadata={data?.metadata}
        onSubmit={handleSubmit}
        onCancel={handleCancel}
        onSwitchToEdit={() => setMode('edit')}
        onSwitchToView={() => setMode('view')}
        onDelete={() => setShowDeleteConfirm(true)}
      />

      {showDeleteConfirm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
          <div className="bg-white rounded-lg shadow-xl p-6 w-[400px]">
            <h3 className="text-sm font-semibold text-gray-900 mb-2">Delete Access Policy</h3>
            <p className="text-xs text-gray-600 mb-5">
              Are you sure you want to delete this access policy? This action cannot be undone.
            </p>
            <div className="flex items-center justify-end gap-3">
              <button
                onClick={() => setShowDeleteConfirm(false)}
                className="px-4 py-2 text-xs font-medium text-gray-600 border border-gray-300 rounded-md hover:bg-gray-50 transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleDelete}
                disabled={deleteMutation.isPending}
                className="px-4 py-2 text-xs font-medium text-white bg-red-600 hover:bg-red-700 rounded-md transition-colors disabled:opacity-50"
              >
                {deleteMutation.isPending ? 'Deleting…' : 'Delete'}
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  )
}

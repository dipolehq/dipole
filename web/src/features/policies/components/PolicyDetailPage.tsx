import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { PageHeader } from '@/shared/components/PageHeader'
import { PolicyForm } from '@/features/policies/components/PolicyForm'
import { usePolicy, useSavePolicy } from '@/features/policies/hooks/usePolicy'
import type { PolicyFormMode, PolicyFormValues } from '@/features/policies/types/formValues'

export function PolicyDetailPage() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()

  const { data, isLoading, isError } = usePolicy(id)
  const saveMutation = useSavePolicy(id ?? '')

  const [mode, setMode] = useState<PolicyFormMode>('view')

  if (!id || isError || (!isLoading && data === null)) {
    return (
      <div className="flex flex-col h-full">
        <div className="flex items-center justify-between px-4 py-3 shrink-0 min-h-[36px] border-b border-border bg-white">
          <PageHeader />
        </div>
        <div className="flex-1 flex items-center justify-center flex-col">
          <p className="text-sm text-gray-500">Policy not found.</p>
          <button onClick={() => navigate('/access-policies')} className="mt-4 text-xs text-orange-500 hover:text-orange-600 font-medium">
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

  return (
    <PolicyForm
      mode={mode}
      isLoading={isLoading || !data}
      initialValues={data?.values}
      metadata={data?.metadata}
      onSubmit={handleSubmit}
      onCancel={handleCancel}
      onSwitchToEdit={() => setMode('edit')}
      onSwitchToView={() => setMode('view')}
    />
  )
}

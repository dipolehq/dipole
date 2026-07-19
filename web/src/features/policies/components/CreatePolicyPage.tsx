import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { PolicyForm } from '@/features/policies/components/PolicyForm'
import { useCreatePolicy } from '@/features/policies/hooks/usePolicy'
import { apiErrorMessage } from '@/features/policies/api/policiesApi'
import type { PolicyFormValues } from '@/features/policies/types/formValues'

export function CreatePolicyPage() {
  const navigate = useNavigate()
  const createMutation = useCreatePolicy()
  const [error, setError] = useState<string | null>(null)

  async function handleSubmit(values: PolicyFormValues) {
    setError(null)
    try {
      await createMutation.mutateAsync(values)
      navigate('/access-policies')
    } catch (err) {
      setError(apiErrorMessage(err, 'Failed to create access policy'))
    }
  }

  function handleCancel() {
    navigate('/access-policies')
  }

  return (
    <>
      {error && (
        <div className="px-6 py-2 bg-red-50 border-b border-red-200 text-sm text-red-700 shrink-0">
          {error}
        </div>
      )}
      <PolicyForm mode="create" onSubmit={handleSubmit} onCancel={handleCancel} />
    </>
  )
}

import { useNavigate } from 'react-router-dom'
import { PolicyForm } from '@/features/policies/components/PolicyForm'
import type { PolicyFormValues } from '@/features/policies/types/formValues'

export function CreatePolicyPage() {
  const navigate = useNavigate()

  function handleSubmit(values: PolicyFormValues) {
    // TODO: wire to a `createPolicy` mutation once a real backend exists.
    // For now, log the payload for developer visibility and return to the list.
    console.info('[create-policy] submitted', values)
    navigate('/access-policies')
  }

  function handleCancel() {
    navigate('/access-policies')
  }

  return <PolicyForm mode="create" onSubmit={handleSubmit} onCancel={handleCancel} />
}

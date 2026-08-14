import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { PageHeader } from '@/shared/components/PageHeader'
import { FloatingDropdown } from '@/shared/components/FloatingInput'
import { DetailPage } from '@/shared/components/ui/DetailPage'
import { AccessConditionForm } from '@/shared/components/forms/AccessConditionForm'
import { apiErrorMessage } from '@/features/access-conditions/api/accessConditionsApi'
import {
  useAccessCondition,
  useSaveAccessCondition,
  useDeleteAccessCondition,
} from '@/features/access-conditions/hooks/useAccessConditions'
import type { AccessConditionFormValues } from '@/shared/components/forms/types'

export function AccessConditionDetailPage() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const { data, isLoading, isError } = useAccessCondition(id)
  const saveMutation = useSaveAccessCondition(id ?? '')
  const deleteMutation = useDeleteAccessCondition()

  const [isEditing, setIsEditing] = useState(false)
  const [localValues, setLocalValues] = useState<AccessConditionFormValues | null>(null)
  const [localStatus, setLocalStatus] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  if (!id || isError || (!isLoading && data === null)) {
    return (
      <div className="flex flex-col h-full">
        <div className="flex items-center justify-between px-4 shrink-0 h-[44px] border-b border-border bg-white">
          <PageHeader />
        </div>
        <div className="flex-1 flex items-center justify-center">
          <p className="text-sm text-gray-400">Access condition not found.</p>
        </div>
      </div>
    )
  }

  if (isLoading || !data) return null

  const currentValues = localValues ?? data.values
  const status = localStatus ?? (data.metadata.isActive ? 'active' : 'inactive')

  function handleEdit() {
    setError(null)
    if (data) setLocalValues({ ...data.values })
    setLocalStatus(data?.metadata.isActive ? 'active' : 'inactive')
    setIsEditing(true)
  }

  async function handleSave() {
    setError(null)
    try {
      await saveMutation.mutateAsync({ values: currentValues, isActive: status !== 'inactive' })
      setIsEditing(false)
      setLocalValues(null)
      setLocalStatus(null)
    } catch (err) {
      setError(apiErrorMessage(err, 'Failed to save access condition'))
    }
  }

  function handleCancel() {
    setError(null)
    setLocalValues(null)
    setLocalStatus(null)
    setIsEditing(false)
  }

  async function handleDelete() {
    try {
      await deleteMutation.mutateAsync(id!)
      navigate('/access-conditions')
    } catch (err) {
      setError(apiErrorMessage(err, 'Failed to delete access condition'))
    }
  }

  return (
    <DetailPage
      name={currentValues.name}
      status={status}
      metadata={{
        id: data.metadata.entityId,
        createdAt: data.metadata.createdAt,
        lastModifiedAt: data.metadata.lastModifiedAt,
      }}
      resourceType="access condition"
      isEditing={isEditing}
      onEdit={handleEdit}
      onSave={handleSave}
      onCancel={handleCancel}
      onDelete={handleDelete}
      isSaving={saveMutation.isPending}
      isDeleting={deleteMutation.isPending}
      error={error}
    >
      <AccessConditionForm
        values={currentValues}
        onChange={(next) => setLocalValues(next)}
        disabled={!isEditing}
        statusSlot={
          <FloatingDropdown
            label="Status"
            options={[
              { value: 'active', label: 'Active' },
              { value: 'inactive', label: 'Inactive' },
            ]}
            value={status}
            onChange={setLocalStatus}
            disabled={!isEditing}
          />
        }
      />
    </DetailPage>
  )
}

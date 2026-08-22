import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { FloatingDropdown } from '@/shared/components/FloatingInput'
import { DetailPage } from '@/shared/components/ui/DetailPage'
import { ClientWorkloadForm } from '@/shared/components/forms/ClientWorkloadForm'
import { apiErrorMessage } from '@/features/client-workloads/api/clientWorkloadsApi'
import {
  useClientWorkload,
  useSaveClientWorkload,
  useDeleteClientWorkload,
} from '@/features/client-workloads/hooks/useClientWorkloads'
import type { ClientWorkloadFormValues } from '@/shared/components/forms/types'

export function ClientWorkloadDetailPage() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const { data, isLoading, isError } = useClientWorkload(id)
  const saveMutation = useSaveClientWorkload(id ?? '')
  const deleteMutation = useDeleteClientWorkload()

  const [isEditing, setIsEditing] = useState(false)
  const [localValues, setLocalValues] = useState<ClientWorkloadFormValues | null>(null)
  const [localStatus, setLocalStatus] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  if (!id || isError || (!isLoading && data === null)) {
    return (
      <div className="flex flex-col h-full">
        <div className="flex-1 flex items-center justify-center">
          <p className="text-sm text-gray-400">Client workload not found.</p>
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
      setError(apiErrorMessage(err, 'Failed to save client workload'))
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
      navigate('/client-workloads')
    } catch (err) {
      setError(apiErrorMessage(err, 'Failed to delete client workload'))
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
      resourceType="client workload"
      isEditing={isEditing}
      onEdit={handleEdit}
      onSave={handleSave}
      onCancel={handleCancel}
      onDelete={handleDelete}
      isSaving={saveMutation.isPending}
      isDeleting={deleteMutation.isPending}
      error={error}
    >
      <ClientWorkloadForm
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

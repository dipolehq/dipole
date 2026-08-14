import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { PageHeader } from '@/shared/components/PageHeader'
import { FloatingDropdown } from '@/shared/components/FloatingInput'
import { DetailPage } from '@/shared/components/ui/DetailPage'
import { TrustProviderForm } from '@/shared/components/forms/TrustProviderForm'
import { apiErrorMessage } from '@/features/trust-providers/api/trustProvidersApi'
import {
  useTrustProvider,
  useSaveTrustProvider,
  useDeleteTrustProvider,
} from '@/features/trust-providers/hooks/useTrustProviders'
import type { TrustProviderFormValues } from '@/shared/components/forms/types'

export function TrustProviderDetailPage() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const { data, isLoading, isError } = useTrustProvider(id)
  const saveMutation = useSaveTrustProvider(id ?? '')
  const deleteMutation = useDeleteTrustProvider()

  const [isEditing, setIsEditing] = useState(false)
  const [localValues, setLocalValues] = useState<TrustProviderFormValues | null>(null)
  const [localStatus, setLocalStatus] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  if (!id || isError || (!isLoading && data === null)) {
    return (
      <div className="flex flex-col h-full">
        <div className="flex items-center justify-between px-4 shrink-0 h-[44px] border-b border-border bg-white">
          <PageHeader />
        </div>
        <div className="flex-1 flex items-center justify-center">
          <p className="text-sm text-gray-400">Trust provider not found.</p>
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
      setError(apiErrorMessage(err, 'Failed to save trust provider'))
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
      navigate('/trust-providers')
    } catch (err) {
      setError(apiErrorMessage(err, 'Failed to delete trust provider'))
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
      resourceType="trust provider"
      isEditing={isEditing}
      onEdit={handleEdit}
      onSave={handleSave}
      onCancel={handleCancel}
      onDelete={handleDelete}
      isSaving={saveMutation.isPending}
      isDeleting={deleteMutation.isPending}
      error={error}
    >
      <TrustProviderForm
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

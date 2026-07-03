import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { PageHeader } from '@/shared/components/PageHeader'
import { EntityMetadata } from '@/shared/components/EntityMetadata'
import { Dropdown } from '@/shared/components/Dropdown'
import { ServerWorkloadForm } from '@/shared/components/forms/ServerWorkloadForm'
import { useServerWorkload, useSaveServerWorkload } from '@/features/server-workloads/hooks/useServerWorkloads'
import type { ServerWorkloadFormValues } from '@/shared/components/forms/types'

const TABS = [
  { key: 'details', label: 'Server Workload' },
] as const

type TabKey = (typeof TABS)[number]['key']

export function ServerWorkloadDetailPage() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const { data, isLoading, isError } = useServerWorkload(id)
  const saveMutation = useSaveServerWorkload(id ?? '')

  const [activeTab, setActiveTab] = useState<TabKey>('details')
  const [isEditing, setIsEditing] = useState(false)
  const [localValues, setLocalValues] = useState<ServerWorkloadFormValues | null>(null)
  const [localStatus, setLocalStatus] = useState<string | null>(null)

  if (!id || isError || (!isLoading && data === null)) {
    return (
      <div className="p-6">
        <PageHeader />
        <p className="mt-6 text-sm text-gray-500">Server workload not found.</p>
        <button
          onClick={() => navigate('/server-workloads')}
          className="mt-4 text-xs text-orange-500 hover:text-orange-600 font-medium"
        >
          ← Back to Server Workloads
        </button>
      </div>
    )
  }

  if (isLoading || !data) {
    return null
  }

  const currentValues = localValues ?? data.values
  const status = localStatus ?? (data.metadata.isActive ? 'active' : 'inactive')
  const disabled = !isEditing

  function handleEditToggle() {
    if (isEditing) {
      void saveMutation.mutateAsync(currentValues).then(() => {
        setIsEditing(false)
        setLocalValues(null)
        setLocalStatus(null)
      })
    } else {
      if (data) setLocalValues({ ...data.values })
      setLocalStatus(data?.metadata.isActive ? 'active' : 'inactive')
      setIsEditing(true)
    }
  }

  function handleCancel() {
    if (isEditing) {
      setLocalValues(null)
      setLocalStatus(null)
      setIsEditing(false)
    } else {
      navigate('/server-workloads')
    }
  }

  function handleValuesChange(next: ServerWorkloadFormValues) {
    setLocalValues(next)
  }

  return (
    <div className="p-6 flex flex-col h-full">
      <div className="flex items-center justify-between mb-6 shrink-0 min-h-[36px]">
        <PageHeader />
      </div>

      <div className="bg-white rounded-lg border border-border flex-1 flex min-h-0">
        <nav className="w-48 border-r border-border py-3 shrink-0 flex flex-col">
          <div className="flex-1">
            {TABS.map((tab) => (
              <button
                key={tab.key}
                type="button"
                onClick={() => setActiveTab(tab.key)}
                className={`w-full text-left px-5 py-2.5 text-xs font-medium transition-colors relative ${
                  activeTab === tab.key
                    ? 'text-orange-600 bg-orange-50'
                    : 'text-gray-500 hover:text-gray-800 hover:bg-gray-50'
                }`}
              >
                {activeTab === tab.key && (
                  <span className="absolute left-0 top-0 bottom-0 w-0.5 bg-orange-500 rounded-r" />
                )}
                <span>{tab.label}</span>
              </button>
            ))}
          </div>
          <div className="px-4 py-3 border-t border-border space-y-2">
            <button
              type="button"
              onClick={handleEditToggle}
              className="w-full px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors"
            >
              {isEditing ? 'Save Server Workload' : 'Edit Server Workload'}
            </button>
            <button
              type="button"
              onClick={handleCancel}
              className="w-full px-4 py-2 border border-gray-300 text-xs font-medium text-gray-600 hover:text-gray-900 hover:border-gray-400 rounded-md transition-colors"
            >
              {isEditing ? 'Cancel' : 'Back'}
            </button>
          </div>
        </nav>

        <div className="flex-1 overflow-y-auto p-6">
          <div className="pb-12">
            {activeTab === 'details' && (
              <>
                <div className="flex items-center justify-between mb-4">
                  <h3 className="text-sm font-semibold text-gray-900">Server Workload</h3>
                  {isEditing && (
                    <button
                      type="button"
                      onClick={handleEditToggle}
                      className="px-4 py-1.5 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors"
                    >
                      Save
                    </button>
                  )}
                </div>
                <div className="border-b border-border mb-6" />

                <div className="space-y-6">
                  <EntityMetadata
                    entityId={data.metadata.entityId}
                    entityLabel="Server Workload"
                    createdAt={data.metadata.createdAt}
                    createdBy={data.metadata.createdBy}
                    lastModifiedAt={data.metadata.lastModifiedAt}
                    lastModifiedBy={data.metadata.lastModifiedBy}
                  />

                  <div>
                    <label className="block text-xs font-medium text-gray-700 mb-1.5">Status</label>
                    {disabled ? (
                      <div className="px-3 py-2 border border-border rounded-md text-sm text-gray-600 bg-gray-50 capitalize">{status}</div>
                    ) : (
                      <Dropdown
                        options={[
                          { value: 'active', label: 'Active' },
                          { value: 'inactive', label: 'Inactive' },
                        ]}
                        value={status}
                        onChange={setLocalStatus}
                        placeholder="Select status..."
                      />
                    )}
                  </div>

                  <ServerWorkloadForm
                    values={currentValues}
                    onChange={handleValuesChange}
                    disabled={disabled}
                  />
                </div>
              </>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

import { useState, useEffect } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { PageHeader } from '@/shared/components/PageHeader'
import {
  FloatingInput,
  FloatingTextarea,
  FloatingDropdown,
} from '@/shared/components/FloatingInput'
import { TagsField } from '@/shared/components/TagsField'
import { DeleteConfirmDialog } from '@/shared/components/ui/DeleteConfirmDialog'
import { ClientWorkloadTab } from '@/features/policies/components/tabs/client-workload/ClientWorkloadTab'
import { TrustProviderTab } from '@/features/policies/components/tabs/trust-provider/TrustProviderTab'
import { AccessConditionTab } from '@/features/policies/components/tabs/access-conditions/AccessConditionTab'
import { CredentialProviderTab } from '@/features/policies/components/tabs/credential-provider/CredentialProviderTab'
import { ServerWorkloadTab } from '@/features/policies/components/tabs/server-workload/ServerWorkloadTab'
import { usePolicy, useSavePolicy, useDeletePolicy } from '@/features/policies/hooks/usePolicy'
import { apiErrorMessage } from '@/features/policies/api/policiesApi'
import type { PolicyFormValues } from '@/features/policies/types/formValues'

const TABS = [
  { key: 'details', label: 'Policy Details' },
  { key: 'client', label: 'Client Workload' },
  { key: 'trust', label: 'Trust Provider' },
  { key: 'conditions', label: 'Access Conditions' },
  { key: 'credential', label: 'Credential Provider' },
  { key: 'server', label: 'Server Workload' },
] as const

type TabKey = (typeof TABS)[number]['key']

export function PolicyDetailPage() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()

  const { data, isLoading, isError } = usePolicy(id)
  const saveMutation = useSavePolicy(id ?? '')
  const deleteMutation = useDeletePolicy()

  const [activeTab, setActiveTab] = useState<TabKey>('details')
  const [isEditing, setIsEditing] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)

  // Local edit state — null means "use data from server", non-null means "user is editing"
  const [localValues, setLocalValues] = useState<PolicyFormValues | null>(null)

  // Unsaved changes protection
  const hasUnsavedChanges = localValues !== null

  // Block browser close/refresh when editing
  useEffect(() => {
    if (!hasUnsavedChanges) return
    function handleBeforeUnload(e: BeforeUnloadEvent) {
      e.preventDefault()
    }
    window.addEventListener('beforeunload', handleBeforeUnload)
    return () => window.removeEventListener('beforeunload', handleBeforeUnload)
  }, [hasUnsavedChanges])

  if (!id || isError || (!isLoading && data === null)) {
    return (
      <div className="flex flex-col h-full">
        <div className="flex items-center justify-between px-4 shrink-0 h-[44px] border-b border-border bg-white">
          <PageHeader />
        </div>
        <div className="flex-1 flex items-center justify-center flex-col">
          <p className="text-sm text-gray-500">Policy not found.</p>
          <button
            onClick={() => navigate('/access-policies')}
            className="mt-4 text-xs text-gray-600 hover:text-gray-700 font-medium"
          >
            &larr; Back to Access Policies
          </button>
        </div>
      </div>
    )
  }

  if (isLoading || !data) return null

  const metadata = data.metadata
  const currentValues = localValues ?? data.values

  // Helpers to update individual fields within localValues
  function updateField<K extends keyof PolicyFormValues>(key: K, value: PolicyFormValues[K]) {
    setLocalValues((prev) => ({ ...(prev ?? data.values), [key]: value }))
  }

  function handleEdit() {
    setError(null)
    setLocalValues({ ...data.values })
    setIsEditing(true)
  }

  async function handleSave() {
    setError(null)
    try {
      await saveMutation.mutateAsync(currentValues)
      setIsEditing(false)
      setLocalValues(null)
    } catch (err) {
      setError(apiErrorMessage(err, 'Failed to save access policy'))
    }
  }

  function handleCancel() {
    setError(null)
    setLocalValues(null)
    setIsEditing(false)
  }

  async function handleDelete() {
    if (!id) return
    try {
      await deleteMutation.mutateAsync(id)
      navigate('/access-policies')
    } catch (err) {
      setError(apiErrorMessage(err, 'Failed to delete access policy'))
    }
  }

  function handleToggleTrustProviderId(tpId: string) {
    const prev = currentValues.trustProviderIds
    if (prev.includes(tpId)) {
      // Removing — also clear inline form data since it may be stale
      updateField('trustProviderIds', prev.filter((v) => v !== tpId))
      updateField('trustProviders', [])
    } else {
      updateField('trustProviderIds', [...prev, tpId])
    }
  }

  function handleToggleAccessConditionId(acId: string) {
    const prev = currentValues.accessConditionIds
    if (prev.includes(acId)) {
      updateField('accessConditionIds', prev.filter((v) => v !== acId))
      updateField('accessConditions', [])
    } else {
      updateField('accessConditionIds', [...prev, acId])
    }
  }

  const disabled = !isEditing

  return (
    <div className="flex flex-col h-full">
      {/* Header bar */}
      <div className="flex items-center justify-between px-4 shrink-0 h-[44px] border-b border-border bg-white">
        <PageHeader />
      </div>

      {/* Toolbar — tabs on left, actions on right */}
      <div className="flex items-center justify-between px-4 py-3 shrink-0">
        {/* Tab tracker (clickable) */}
        <div className="flex items-center gap-1">
          {TABS.map((tab, idx) => (
            <div key={tab.key} className="flex items-center">
              <button
                type="button"
                onClick={() => setActiveTab(tab.key)}
                className={`px-2 py-1 text-xs font-medium transition-colors ${
                  tab.key === activeTab ? 'text-gray-900' : 'text-gray-400 hover:text-gray-700'
                }`}
              >
                {tab.label}
              </button>
              {idx < TABS.length - 1 && (
                <span className="text-gray-300 mx-0.5 text-xs select-none">&rarr;</span>
              )}
            </div>
          ))}
        </div>

        {/* Actions */}
        <div className="flex items-center gap-2">
          {isEditing ? (
            <>
              <button
                type="button"
                onClick={handleCancel}
                className="inline-flex items-center gap-2 h-9 px-4 rounded-md border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={handleSave}
                disabled={saveMutation.isPending}
                className="inline-flex items-center gap-2 h-9 px-4 rounded-md border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors disabled:opacity-50"
              >
                {saveMutation.isPending ? 'Saving…' : 'Save'}
              </button>
            </>
          ) : (
            <>
              <button
                type="button"
                onClick={handleEdit}
                className="inline-flex items-center gap-2 h-9 px-4 rounded-md border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors"
              >
                Edit
              </button>
              <button
                type="button"
                onClick={() => setShowDeleteConfirm(true)}
                disabled={deleteMutation.isPending}
                className="inline-flex items-center gap-2 h-9 px-4 rounded-md border border-red-200 bg-white text-sm font-medium text-red-600 hover:bg-red-50 transition-colors disabled:opacity-50"
              >
                Delete
              </button>
            </>
          )}
        </div>
      </div>

      {error && (
        <div className="mx-4 mb-2 px-4 py-2 bg-red-50 border border-red-200 rounded-md text-sm text-red-700">
          {error}
        </div>
      )}

      {/* Content box */}
      <div className="flex-1 min-h-0 mx-4 border border-border bg-white overflow-y-auto">
        <div className="grid grid-cols-[1fr_380px] h-full">
          {/* Form area */}
          <div className="px-6 py-5 overflow-y-auto border-r border-border">
            {/* Policy Details */}
            {activeTab === 'details' && (
              <div className="space-y-6">
                <div>
                  <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider mb-4">Details</h4>
                  <div className="space-y-4">
                    <FloatingDropdown
                      label="Status"
                      options={[
                        { value: 'active', label: 'Active' },
                        { value: 'inactive', label: 'Inactive' },
                      ]}
                      value={currentValues.status}
                      onChange={(v) => updateField('status', v)}
                      disabled={disabled}
                    />
                    <FloatingInput
                      label="Access Policy Name"
                      value={currentValues.name}
                      onChange={(v) => updateField('name', v)}
                      required
                      disabled={disabled}
                    />
                    <FloatingTextarea
                      label="Description (Optional)"
                      value={currentValues.description}
                      onChange={(v) => updateField('description', v)}
                      rows={3}
                      disabled={disabled}
                    />
                  </div>
                </div>

                <div>
                  <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider mb-4">Tags</h4>
                  <TagsField
                    value={currentValues.tags}
                    onChange={(v) => updateField('tags', v)}
                    disabled={disabled}
                  />
                </div>
              </div>
            )}

            {/* Client Workload */}
            {activeTab === 'client' && (
              <ClientWorkloadTab
                selectedId={currentValues.clientWorkloadId}
                onSelect={(v) => updateField('clientWorkloadId', v)}
                values={currentValues.client}
                onChange={(v) => updateField('client', v)}
                disabled={disabled}
              />
            )}

            {/* Trust Provider */}
            {activeTab === 'trust' && (
              <TrustProviderTab
                selectedIds={currentValues.trustProviderIds}
                onToggleId={handleToggleTrustProviderId}
                values={currentValues.trustProviders}
                onChange={(v) => updateField('trustProviders', v)}
                disabled={disabled}
              />
            )}

            {/* Access Conditions */}
            {activeTab === 'conditions' && (
              <AccessConditionTab
                selectedIds={currentValues.accessConditionIds}
                onToggleId={handleToggleAccessConditionId}
                values={currentValues.accessConditions}
                onChange={(v) => updateField('accessConditions', v)}
                disabled={disabled}
              />
            )}

            {/* Credential Provider */}
            {activeTab === 'credential' && (
              <CredentialProviderTab
                selectedId={currentValues.credentialProviderId}
                onSelect={(v) => updateField('credentialProviderId', v)}
                values={currentValues.credential}
                onChange={(v) => updateField('credential', v)}
                disabled={disabled}
              />
            )}

            {/* Server Workload */}
            {activeTab === 'server' && (
              <ServerWorkloadTab
                selectedId={currentValues.serverWorkloadId}
                onSelect={(v) => updateField('serverWorkloadId', v)}
                values={currentValues.server}
                onChange={(v) => updateField('server', v)}
                disabled={disabled}
              />
            )}
          </div>

          {/* Right panel — metadata */}
          <div className="px-5 py-5 overflow-y-auto">
            <div className="space-y-4">
              <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider">Metadata</h4>
              <FloatingInput label="ID" value={metadata.policyId} disabled />
              <FloatingInput
                label="Created"
                value={new Date(metadata.createdAt).toLocaleString('en-IN', { dateStyle: 'medium', timeStyle: 'short' })}
                disabled
              />
              <FloatingInput
                label="Last Modified"
                value={new Date(metadata.lastModifiedAt).toLocaleString('en-IN', { dateStyle: 'medium', timeStyle: 'short' })}
                disabled
              />
            </div>
          </div>
        </div>
      </div>

      {/* Delete confirmation dialog */}
      <DeleteConfirmDialog
        open={showDeleteConfirm}
        onClose={() => setShowDeleteConfirm(false)}
        onConfirm={handleDelete}
        resourceName={currentValues.name || 'Untitled'}
        resourceId={metadata.policyId}
        resourceType="access policy"
        loading={deleteMutation.isPending}
      />
    </div>
  )
}

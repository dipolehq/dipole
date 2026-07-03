import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Controller, FormProvider, useForm } from 'react-hook-form'
import { PageHeader } from '@/shared/components/PageHeader'
import { PolicyDetailDetailsTab } from '@/features/policies/components/tabs/details/PolicyDetailDetailsTab'
import { DetailsTab } from '@/features/policies/components/tabs/details/DetailsTab'
import { ClientWorkloadTab } from '@/features/policies/components/tabs/client-workload/ClientWorkloadTab'
import { ServerWorkloadTab } from '@/features/policies/components/tabs/server-workload/ServerWorkloadTab'
import { TrustProviderTab } from '@/features/policies/components/tabs/trust-provider/TrustProviderTab'
import { AccessConditionTab } from '@/features/policies/components/tabs/access-conditions/AccessConditionTab'
import { CredentialProviderTab } from '@/features/policies/components/tabs/credential-provider/CredentialProviderTab'
import { EMPTY_POLICY_FORM_VALUES } from '@/features/policies/utils/defaults'
import type {
  PolicyFormMode,
  PolicyFormValues,
  PolicyMetadata,
} from '@/features/policies/types/formValues'

const TABS = [
  { key: 'details', label: 'Policy Details' },
  { key: 'client', label: 'Client Workload' },
  { key: 'trust', label: 'Trust Provider' },
  { key: 'conditions', label: 'Access Conditions' },
  { key: 'credential', label: 'Credential Providers' },
  { key: 'server', label: 'Server Workload' },
] as const

type TabKey = (typeof TABS)[number]['key']

interface PolicyFormProps {
  mode: PolicyFormMode
  /** Pre-loaded values (edit/view); omitted for create. */
  initialValues?: PolicyFormValues
  /** Read-only policy metadata (edit/view only). */
  metadata?: PolicyMetadata
  /** True while `initialValues` is still loading. */
  isLoading?: boolean
  onSubmit: (values: PolicyFormValues) => void | Promise<void>
  onCancel: () => void
  /** Only shown in edit mode; toggles the mode to 'view' externally. */
  onSwitchToView?: () => void
  /** Only shown in view mode; toggles the mode to 'edit' externally. */
  onSwitchToEdit?: () => void
}

export function PolicyForm({
  mode,
  initialValues,
  metadata,
  isLoading = false,
  onSubmit,
  onCancel,
  onSwitchToView,
  onSwitchToEdit,
}: PolicyFormProps) {
  const navigate = useNavigate()
  const disabled = mode === 'view'

  const form = useForm<PolicyFormValues>({
    defaultValues: initialValues ?? EMPTY_POLICY_FORM_VALUES,
  })

  const { control, handleSubmit, reset } = form

  // When async data arrives (or when returning to view mode) re-seed the form.
  useEffect(() => {
    if (initialValues) {
      reset(initialValues)
    }
  }, [initialValues, reset])

  const [activeTab, setActiveTab] = useState<TabKey>('details')
  const [savedTabs, setSavedTabs] = useState<Set<TabKey>>(new Set())

  function markTabSaved() {
    setSavedTabs((prev) => new Set([...prev, activeTab]))
  }

  const primaryLabel =
    mode === 'view' ? 'Edit Policy' : mode === 'create' ? 'Save Policy' : 'Save Policy'
  const secondaryLabel = mode === 'view' ? 'Back' : 'Cancel'

  function handlePrimary() {
    if (mode === 'view') {
      onSwitchToEdit?.()
      return
    }
    void handleSubmit(async (values) => {
      await onSubmit(values)
      if (mode === 'edit') onSwitchToView?.()
    })()
  }

  function handleSecondary() {
    if (mode === 'view') {
      navigate('/access-policies')
      return
    }
    // Reset any in-progress edits then bubble the cancel up.
    if (initialValues) reset(initialValues)
    else reset(EMPTY_POLICY_FORM_VALUES)
    setSavedTabs(new Set())
    onCancel()
  }

  if (isLoading) {
    return null
  }

  return (
    <FormProvider {...form}>
      <div
        className="px-4 py-3 flex flex-col h-full"
      >
        <div className="flex items-center justify-between mb-4 shrink-0 min-h-[36px]">
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
                  className={`w-full text-left px-5 py-2.5 text-xs font-medium transition-colors relative flex items-center justify-between ${
                    activeTab === tab.key
                      ? 'text-orange-600 bg-orange-50'
                      : 'text-gray-500 hover:text-gray-800 hover:bg-gray-50'
                  }`}
                >
                  {activeTab === tab.key && (
                    <span className="absolute left-0 top-0 bottom-0 w-0.5 bg-orange-500 rounded-r" />
                  )}
                  <span>{tab.label}</span>
                  {savedTabs.has(tab.key) && (
                    <span className="w-4 h-4 bg-green-500 rounded-full flex items-center justify-center shrink-0">
                      <svg className="w-2.5 h-2.5 text-white" fill="none" stroke="currentColor" strokeWidth="3" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                      </svg>
                    </span>
                  )}
                </button>
              ))}
            </div>
            <div className="px-4 py-3 border-t border-border space-y-2">
              <button
                type="button"
                onClick={handlePrimary}
                className="w-full px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors"
              >
                {primaryLabel}
              </button>
              <button
                type="button"
                onClick={handleSecondary}
                className="w-full px-4 py-2 border border-gray-300 text-xs font-medium text-gray-600 hover:text-gray-900 hover:border-gray-400 rounded-md transition-colors"
              >
                {secondaryLabel}
              </button>
            </div>
          </nav>

          <div className="flex-1 overflow-y-auto p-6">
            <div className="pb-12">
              {activeTab !== 'details' && (
                <>
                  <div className="flex items-center justify-between mb-4">
                    <h3 className="text-sm font-semibold text-gray-900">
                      {TABS.find((t) => t.key === activeTab)?.label}
                    </h3>
                    {!disabled && (
                      <button
                        type="button"
                        onClick={markTabSaved}
                        className="px-4 py-1.5 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors"
                      >
                        Save
                      </button>
                    )}
                  </div>
                  <div className="border-b border-border mb-6" />
                </>
              )}

              {activeTab === 'details' && (
                <DetailsSection mode={mode} disabled={disabled} metadata={metadata} onSave={markTabSaved} />
              )}

              {activeTab === 'client' && (
                <Controller
                  control={control}
                  name="client"
                  render={({ field }) => (
                    <Controller
                      control={control}
                      name="clientWorkloadId"
                      render={({ field: idField }) => (
                        <ClientWorkloadTab
                          selectedId={idField.value}
                          onSelect={idField.onChange}
                          values={field.value}
                          onChange={field.onChange}
                          disabled={disabled}
                        />
                      )}
                    />
                  )}
                />
              )}

              {activeTab === 'server' && (
                <Controller
                  control={control}
                  name="server"
                  render={({ field }) => (
                    <Controller
                      control={control}
                      name="serverWorkloadId"
                      render={({ field: idField }) => (
                        <ServerWorkloadTab
                          selectedId={idField.value}
                          onSelect={idField.onChange}
                          values={field.value}
                          onChange={field.onChange}
                          disabled={disabled}
                        />
                      )}
                    />
                  )}
                />
              )}

              {activeTab === 'trust' && (
                <Controller
                  control={control}
                  name="trustProviders"
                  render={({ field }) => (
                    <Controller
                      control={control}
                      name="trustProviderIds"
                      render={({ field: idField }) => (
                        <TrustProviderTab
                          selectedIds={idField.value}
                          onToggleId={(id) =>
                            idField.onChange(
                              idField.value.includes(id)
                                ? idField.value.filter((v) => v !== id)
                                : [...idField.value, id],
                            )
                          }
                          values={field.value}
                          onChange={field.onChange}
                          disabled={disabled}
                        />
                      )}
                    />
                  )}
                />
              )}

              {activeTab === 'conditions' && (
                <Controller
                  control={control}
                  name="accessConditions"
                  render={({ field }) => (
                    <Controller
                      control={control}
                      name="accessConditionIds"
                      render={({ field: idField }) => (
                        <AccessConditionTab
                          selectedIds={idField.value}
                          onToggleId={(id) =>
                            idField.onChange(
                              idField.value.includes(id)
                                ? idField.value.filter((v) => v !== id)
                                : [...idField.value, id],
                            )
                          }
                          values={field.value}
                          onChange={field.onChange}
                          disabled={disabled}
                        />
                      )}
                    />
                  )}
                />
              )}

              {activeTab === 'credential' && (
                <Controller
                  control={control}
                  name="credential"
                  render={({ field }) => (
                    <Controller
                      control={control}
                      name="credentialProviderId"
                      render={({ field: idField }) => (
                        <CredentialProviderTab
                          selectedId={idField.value}
                          onSelect={idField.onChange}
                          values={field.value}
                          onChange={field.onChange}
                          disabled={disabled}
                        />
                      )}
                    />
                  )}
                />
              )}
            </div>
          </div>
        </div>
      </div>
    </FormProvider>
  )
}

interface DetailsSectionProps {
  mode: PolicyFormMode
  disabled: boolean
  metadata?: PolicyMetadata
  onSave: () => void
}

function DetailsSection({ mode, disabled, metadata, onSave }: DetailsSectionProps) {
  const header = (
    <div className="flex items-center justify-between mb-4">
      <h3 className="text-sm font-semibold text-gray-900">Access Policy Details</h3>
      {!disabled && (
        <button
          type="button"
          onClick={onSave}
          className="px-4 py-1.5 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors"
        >
          Save
        </button>
      )}
    </div>
  )

  if (mode === 'create') {
    return (
      <>
        {header}
        <div className="border-b border-border mb-6" />
        <Controller
          name="name"
          render={({ field: nameField }) => (
            <Controller
              name="description"
              render={({ field: descField }) => (
                <Controller
                  name="status"
                  render={({ field: statusField }) => (
                    <DetailsTab
                      name={nameField.value}
                      description={descField.value}
                      isActive={statusField.value === 'active'}
                      onNameChange={nameField.onChange}
                      onDescriptionChange={descField.onChange}
                      onActiveChange={(v) => statusField.onChange(v ? 'active' : 'inactive')}
                    />
                  )}
                />
              )}
            />
          )}
        />
      </>
    )
  }

  return (
    <>
      {header}
      <div className="border-b border-border mb-6" />
      <Controller
        name="name"
        render={({ field: nameField }) => (
          <Controller
            name="description"
            render={({ field: descField }) => (
              <Controller
                name="status"
                render={({ field: statusField }) => (
                  <PolicyDetailDetailsTab
                    name={nameField.value}
                    description={descField.value}
                    status={statusField.value}
                    policyId={metadata?.policyId ?? ''}
                    createdAt={metadata?.createdAt ?? ''}
                    createdBy={metadata?.createdBy ?? ''}
                    lastModifiedAt={metadata?.lastModifiedAt ?? ''}
                    lastModifiedBy={metadata?.lastModifiedBy ?? ''}
                    disabled={disabled}
                    onNameChange={nameField.onChange}
                    onDescriptionChange={descField.onChange}
                    onStatusChange={statusField.onChange}
                  />
                )}
              />
            )}
          />
        )}
      />
    </>
  )
}

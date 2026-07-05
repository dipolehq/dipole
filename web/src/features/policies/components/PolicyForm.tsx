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
  { key: 'credential', label: 'Credential Provider' },
  { key: 'server', label: 'Server Workload' },
] as const

type TabKey = (typeof TABS)[number]['key']

interface PolicyFormProps {
  mode: PolicyFormMode
  initialValues?: PolicyFormValues
  metadata?: PolicyMetadata
  isLoading?: boolean
  onSubmit: (values: PolicyFormValues) => void | Promise<void>
  onCancel: () => void
  onSwitchToView?: () => void
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

  useEffect(() => {
    if (initialValues) reset(initialValues)
  }, [initialValues, reset])

  const [activeTab, setActiveTab] = useState<TabKey>('details')

  const primaryLabel = mode === 'view' ? 'Edit' : 'Save'
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
    if (initialValues) reset(initialValues)
    else reset(EMPTY_POLICY_FORM_VALUES)
    onCancel()
  }

  if (isLoading) return null

  return (
    <FormProvider {...form}>
      <div className="flex flex-col h-full">
        {/* Header bar */}
        <div className="flex items-center justify-between px-4 py-3 shrink-0 min-h-[36px] border-b border-border bg-white">
          <PageHeader />
          <div className="flex items-center gap-3">
            <button type="button" onClick={handlePrimary} className="px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white text-sm font-medium rounded-md transition-colors">
              {primaryLabel}
            </button>
            <button type="button" onClick={handleSecondary} className="px-4 py-2 border border-gray-300 text-sm font-medium text-gray-600 hover:text-gray-900 hover:border-gray-400 rounded-md transition-colors">
              {secondaryLabel}
            </button>
          </div>
        </div>

        {/* Horizontal tabs */}
        <div className="flex items-center bg-white px-6 pt-4 shrink-0">
          {TABS.map((tab, idx) => (
            <button
              key={tab.key}
              type="button"
              onClick={() => setActiveTab(tab.key)}
              className={`${idx === 0 ? 'pl-0 pr-3' : 'px-3'} py-2.5 text-xs font-medium transition-colors relative ${
                activeTab === tab.key
                  ? 'text-orange-600'
                  : 'text-gray-500 hover:text-gray-800'
              }`}
            >
              {tab.label}
              {activeTab === tab.key && (
                <span className="absolute bottom-0 left-0 right-0 h-0.5 bg-orange-500 rounded-t" />
              )}
            </button>
          ))}
        </div>

        {/* Tab content */}
        <div className="flex-1 overflow-y-auto bg-white">
          <div className="flex h-full">
            {/* Form — left 60% */}
            <div className="w-[60%] px-6 py-6 overflow-y-auto">
              {activeTab === 'details' && (
                <div className="relative">
                  {!disabled && <div className="absolute -top-1 right-0"><button type="button" onClick={handlePrimary} className="px-4 py-1.5 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors">Save</button></div>}
                  <div className="space-y-6">
                    {!disabled && <div className="flex items-center border-b border-border pb-3"><span className="text-xs font-medium text-transparent select-none">.</span></div>}
                    <DetailsSection mode={mode} disabled={disabled} metadata={metadata} />
                  </div>
                </div>
              )}

              {activeTab === 'client' && (
                <div className="relative">
                  {!disabled && <div className="absolute -top-1 right-0"><button type="button" onClick={handlePrimary} className="px-4 py-1.5 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors">Save</button></div>}
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
                </div>
              )}

              {activeTab === 'server' && (
                <div className="relative">
                  {!disabled && <div className="absolute -top-1 right-0"><button type="button" onClick={handlePrimary} className="px-4 py-1.5 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors">Save</button></div>}
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
                </div>
              )}

              {activeTab === 'trust' && (
                <div className="relative">
                  {!disabled && <div className="absolute -top-1 right-0"><button type="button" onClick={handlePrimary} className="px-4 py-1.5 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors">Save</button></div>}
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
                </div>
              )}

              {activeTab === 'conditions' && (
                <div className="relative">
                  {!disabled && <div className="absolute -top-1 right-0"><button type="button" onClick={handlePrimary} className="px-4 py-1.5 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors">Save</button></div>}
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
                </div>
              )}

              {activeTab === 'credential' && (
                <div className="relative">
                  {!disabled && <div className="absolute -top-1 right-0"><button type="button" onClick={handlePrimary} className="px-4 py-1.5 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors">Save</button></div>}
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
                </div>
              )}
            </div>

            {/* Help panel — right 40% */}
            <div className="w-[40%] px-6 py-6 overflow-y-auto">
              <div className="border border-border rounded-lg p-5 sticky top-0">
                {activeTab === 'details' && (
                  <>
                    <p className="text-xs text-gray-700 leading-relaxed mb-3">
                      <span className="font-semibold text-orange-600">Access Policies</span> are the fundamental unit of authorization. They define who (Client Workload) can access what (Server Workload), under what conditions, and with what credential.
                    </p>
                    <p className="text-xs text-gray-700 leading-relaxed">
                      A policy links together a Client Workload, Trust Provider, Access Conditions, Credential Provider, and Server Workload into a single authorization decision.
                    </p>
                  </>
                )}
                {activeTab === 'client' && (
                  <>
                    <p className="text-xs text-gray-700 leading-relaxed mb-3">
                      <span className="font-semibold text-orange-600">Client Workloads</span> represent the software that initiates access requests. They are identified by attributes from their runtime environment.
                    </p>
                    <p className="text-xs text-gray-700 leading-relaxed">
                      Select an existing Client Workload or create a new one inline. The workload's identifiers determine how it will be matched during policy evaluation.
                    </p>
                  </>
                )}
                {activeTab === 'trust' && (
                  <>
                    <p className="text-xs text-gray-700 leading-relaxed mb-3">
                      <span className="font-semibold text-orange-600">Trust Providers</span> verify workload identity using cryptographic attestation. Multiple providers use AND logic — all must pass.
                    </p>
                    <p className="text-xs text-gray-700 leading-relaxed">
                      Add one or more Trust Providers to verify the Client Workload's identity before granting access.
                    </p>
                  </>
                )}
                {activeTab === 'conditions' && (
                  <>
                    <p className="text-xs text-gray-700 leading-relaxed mb-3">
                      <span className="font-semibold text-orange-600">Access Conditions</span> add contextual security checks beyond identity. All conditions use AND logic.
                    </p>
                    <p className="text-xs text-gray-700 leading-relaxed">
                      Conditions are optional. If configured, they must all pass for access to be granted. Supports time-based and GeoIP restrictions.
                    </p>
                  </>
                )}
                {activeTab === 'credential' && (
                  <>
                    <p className="text-xs text-gray-700 leading-relaxed mb-3">
                      <span className="font-semibold text-orange-600">Credential Providers</span> produce the credential that gets injected into the request to the Server Workload.
                    </p>
                    <p className="text-xs text-gray-700 leading-relaxed">
                      Select the type of credential the Server Workload expects — API Key, OAuth token, AWS STS credentials, database password, etc.
                    </p>
                  </>
                )}
                {activeTab === 'server' && (
                  <>
                    <p className="text-xs text-gray-700 leading-relaxed mb-3">
                      <span className="font-semibold text-orange-600">Server Workloads</span> represent the services, APIs, or databases being accessed. They're identified by host, port, and protocol.
                    </p>
                    <p className="text-xs text-gray-700 leading-relaxed">
                      The authentication method configured here determines how the credential gets injected into the request.
                    </p>
                  </>
                )}
              </div>
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
}

function DetailsSection({ mode, disabled, metadata }: DetailsSectionProps) {
  if (mode === 'create') {
    return (
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
    )
  }

  return (
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
  )
}

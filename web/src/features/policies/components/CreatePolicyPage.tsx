import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  FloatingInput,
  FloatingTextarea,
  FloatingDropdown,
} from '@/shared/components/FloatingInput'
import { TagsField } from '@/shared/components/TagsField'
import { ClientWorkloadTab } from '@/features/policies/components/tabs/client-workload/ClientWorkloadTab'
import { TrustProviderTab } from '@/features/policies/components/tabs/trust-provider/TrustProviderTab'
import { AccessConditionTab } from '@/features/policies/components/tabs/access-conditions/AccessConditionTab'
import { CredentialProviderTab } from '@/features/policies/components/tabs/credential-provider/CredentialProviderTab'
import { ServerWorkloadTab } from '@/features/policies/components/tabs/server-workload/ServerWorkloadTab'
import { useCreatePolicy } from '@/features/policies/hooks/usePolicy'
import { apiErrorMessage } from '@/features/policies/api/policiesApi'
import { policyFormSchema } from '@/shared/lib/validation'
import type { PolicyFormValues } from '@/features/policies/types/formValues'
import type {
  FormTag,
  ClientWorkloadFormValues,
  ServerWorkloadFormValues,
  TrustProviderFormValues,
  AccessConditionFormValues,
  CredentialProviderFormValues,
} from '@/shared/components/forms/types'
import {
  EMPTY_CLIENT_WORKLOAD,
  EMPTY_SERVER_WORKLOAD,
  EMPTY_CREDENTIAL_PROVIDER,
} from '@/shared/components/forms/types'

const STEPS = [
  { key: 'details', label: 'Policy Details' },
  { key: 'client', label: 'Client Workload' },
  { key: 'trust', label: 'Trust Provider' },
  { key: 'conditions', label: 'Access Conditions' },
  { key: 'credential', label: 'Credential Provider' },
  { key: 'server', label: 'Server Workload' },
] as const

type StepKey = (typeof STEPS)[number]['key']

export function CreatePolicyPage() {
  const navigate = useNavigate()
  const createMutation = useCreatePolicy()
  const [activeStep, setActiveStep] = useState<StepKey>('details')
  const [error, setError] = useState<string | null>(null)

  // Policy details
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [status, setStatus] = useState('active')
  const [tags, setTags] = useState<FormTag[]>([])

  // Client workload
  const [clientWorkloadId, setClientWorkloadId] = useState('')
  const [client, setClient] = useState<ClientWorkloadFormValues>({ ...EMPTY_CLIENT_WORKLOAD })

  // Trust providers (multi)
  const [trustProviderIds, setTrustProviderIds] = useState<string[]>([])
  const [trustProviders, setTrustProviders] = useState<TrustProviderFormValues[]>([])

  // Access conditions (multi)
  const [accessConditionIds, setAccessConditionIds] = useState<string[]>([])
  const [accessConditions, setAccessConditions] = useState<AccessConditionFormValues[]>([])

  // Credential provider
  const [credentialProviderId, setCredentialProviderId] = useState('')
  const [credential, setCredential] = useState<CredentialProviderFormValues>({ ...EMPTY_CREDENTIAL_PROVIDER })

  // Server workload
  const [serverWorkloadId, setServerWorkloadId] = useState('')
  const [server, setServer] = useState<ServerWorkloadFormValues>({ ...EMPTY_SERVER_WORKLOAD })

  const activeIndex = STEPS.findIndex((s) => s.key === activeStep)

  function handleNext() {
    if (activeIndex < STEPS.length - 1) {
      setActiveStep(STEPS[activeIndex + 1]!.key)
    }
  }

  function handleBack() {
    if (activeIndex > 0) {
      setActiveStep(STEPS[activeIndex - 1]!.key)
    }
  }

  function buildFormValues(): PolicyFormValues {
    return {
      name,
      description,
      status,
      tags,
      clientWorkloadId,
      client,
      serverWorkloadId,
      server,
      trustProviderIds,
      trustProviders,
      accessConditionIds,
      accessConditions,
      credentialProviderId,
      credential,
    }
  }

  async function handleSave() {
    setError(null)
    const values = buildFormValues()
    const result = policyFormSchema.safeParse(values)
    if (!result.success) {
      setError(result.error.errors[0]?.message ?? 'Please fill in all required fields')
      return
    }
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

  function handleToggleTrustProviderId(id: string) {
    setTrustProviderIds((prev) =>
      prev.includes(id) ? prev.filter((v) => v !== id) : [...prev, id],
    )
  }

  function handleToggleAccessConditionId(id: string) {
    setAccessConditionIds((prev) =>
      prev.includes(id) ? prev.filter((v) => v !== id) : [...prev, id],
    )
  }

  return (
    <div className="flex flex-col h-full">

      {/* Toolbar — step tracker on left, Save + Cancel on right */}
      <div className="flex items-center justify-between px-4 py-3 shrink-0">
        {/* Step tracker (read-only, controlled by form navigation) */}
        <div className="flex items-center gap-1">
          {STEPS.map((step, idx) => (
            <div key={step.key} className="flex items-center">
              <span
                className={`px-2 py-1 text-xs font-medium ${
                  idx === activeIndex
                    ? 'text-gray-900'
                    : idx < activeIndex
                      ? 'text-gray-700'
                      : 'text-gray-400'
                }`}
              >
                {step.label}
              </span>
              {idx < STEPS.length - 1 && (
                <span className="text-gray-300 mx-0.5 text-xs select-none">&rarr;</span>
              )}
            </div>
          ))}
        </div>

        {/* Actions */}
        <div className="flex items-center gap-2">
          {activeIndex > 0 && (
            <button
              type="button"
              onClick={handleBack}
              className="inline-flex items-center gap-2 h-9 px-4 rounded-md border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors"
            >
              Back
            </button>
          )}
          {activeIndex === 0 && (
            <button
              type="button"
              onClick={handleCancel}
              className="inline-flex items-center gap-2 h-9 px-4 rounded-md border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors"
            >
              Cancel
            </button>
          )}
          {activeIndex < STEPS.length - 1 ? (
            <button
              type="button"
              onClick={handleNext}
              className="inline-flex items-center gap-2 h-9 px-4 rounded-md border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors"
            >
              Save &amp; Continue
            </button>
          ) : (
            <button
              type="button"
              onClick={handleSave}
              disabled={createMutation.isPending}
              className="inline-flex items-center gap-2 h-9 px-4 rounded-md border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors disabled:opacity-50"
            >
              {createMutation.isPending ? 'Saving…' : 'Save & Create Policy'}
            </button>
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
            {/* Step 1: Policy Details */}
            {activeStep === 'details' && (
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
                      value={status}
                      onChange={setStatus}
                    />
                    <FloatingInput
                      label="Access Policy Name"
                      value={name}
                      onChange={setName}
                      required
                    />
                    <FloatingTextarea
                      label="Description (Optional)"
                      value={description}
                      onChange={setDescription}
                      rows={3}
                    />
                  </div>
                </div>

                <div>
                  <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider mb-4">Tags</h4>
                  <TagsField value={tags} onChange={setTags} />
                </div>
              </div>
            )}

            {/* Step 2: Client Workload */}
            {activeStep === 'client' && (
              <ClientWorkloadTab
                selectedId={clientWorkloadId}
                onSelect={setClientWorkloadId}
                values={client}
                onChange={setClient}
              />
            )}

            {/* Step 3: Trust Provider */}
            {activeStep === 'trust' && (
              <TrustProviderTab
                selectedIds={trustProviderIds}
                onToggleId={handleToggleTrustProviderId}
                values={trustProviders}
                onChange={setTrustProviders}
              />
            )}

            {/* Step 4: Access Conditions */}
            {activeStep === 'conditions' && (
              <AccessConditionTab
                selectedIds={accessConditionIds}
                onToggleId={handleToggleAccessConditionId}
                values={accessConditions}
                onChange={setAccessConditions}
              />
            )}

            {/* Step 5: Credential Provider */}
            {activeStep === 'credential' && (
              <CredentialProviderTab
                selectedId={credentialProviderId}
                onSelect={setCredentialProviderId}
                values={credential}
                onChange={setCredential}
              />
            )}

            {/* Step 6: Server Workload (final step) */}
            {activeStep === 'server' && (
              <ServerWorkloadTab
                selectedId={serverWorkloadId}
                onSelect={setServerWorkloadId}
                values={server}
                onChange={setServer}
              />
            )}
          </div>

          {/* Help text — right panel */}
          <div className="px-5 py-5 overflow-y-auto">
            {activeStep === 'details' && (
              <div className="space-y-3">
                <p className="text-xs text-gray-600 leading-relaxed">
                  <span className="font-semibold text-gray-800">Access Policies</span> are the
                  fundamental unit of authorization. They define who (Client Workload) can access
                  what (Server Workload), under what conditions, and with what credential.
                </p>
                <p className="text-xs text-gray-600 leading-relaxed">
                  A policy links together a Client Workload, Trust Provider, Access Conditions,
                  Credential Provider, and Server Workload into a single authorization decision.
                </p>
              </div>
            )}

            {activeStep === 'client' && (
              <div className="space-y-3">
                <p className="text-xs text-gray-600 leading-relaxed">
                  <span className="font-semibold text-gray-800">Client Workloads</span> are software
                  applications that access services provided by Server Workloads. Examples include
                  custom applications, daemons, ETL processors, one-off or batch jobs, and deployment
                  scripts. Typically, Client Workloads run in the background and must access services
                  on their own behalf, without user intervention.
                </p>
                <p className="text-xs text-gray-600 leading-relaxed">
                  A <span className="font-semibold text-gray-800">Client Identifier</span> is a bit
                  like the Client Workload's username. Dipole uses it to identify the workload but not
                  to authenticate it. If you deploy your workload in Kubernetes, we recommend using its
                  Kubernetes Pod Name Prefix as its identifier.
                </p>
                <p className="text-xs text-gray-600 leading-relaxed">
                  Dipole supports identifying{' '}
                  <span className="font-semibold text-gray-800">Client Workloads</span> by using
                  multiple Client Identifiers and typically requires all of the values be matched.
                  Multiple values within a Client Identifier type can be used in an either OR
                  combination, though at least one of the values for each Client Identifier type will
                  be required.
                </p>
              </div>
            )}

            {activeStep === 'trust' && (
              <div className="space-y-3">
                <p className="text-xs text-gray-600 leading-relaxed">
                  <span className="font-semibold text-gray-800">Trust Providers</span> verify
                  workload identity using cryptographic attestation from the runtime environment. They
                  answer: "Can the environment VOUCH for this workload's identity?"
                </p>
                <p className="text-xs text-gray-600 leading-relaxed">
                  Instead of a shared secret, the workload's environment provides cryptographic proof.
                  Supported types include Kubernetes Service Account tokens, AWS Metadata/Role,
                  GitHub/GitLab OIDC tokens, and generic OIDC providers.
                </p>
                <p className="text-xs text-gray-600 leading-relaxed">
                  <span className="font-semibold text-gray-800">Match Rules</span> define which claims
                  or attributes must match for attestation to succeed. All rules use AND logic — every
                  configured rule must pass.
                </p>
              </div>
            )}

            {activeStep === 'conditions' && (
              <div className="space-y-3">
                <p className="text-xs text-gray-600 leading-relaxed">
                  <span className="font-semibold text-gray-800">Access Conditions</span> add
                  contextual security checks beyond identity verification. Even if a workload's
                  identity is verified, additional conditions must be satisfied before access is
                  granted.
                </p>
                <p className="text-xs text-gray-600 leading-relaxed">
                  Supported conditions include <span className="font-semibold text-gray-800">Time-based</span>{' '}
                  restrictions (specific days and hours) and{' '}
                  <span className="font-semibold text-gray-800">GeoIP</span> restrictions (country and
                  subdivision).
                </p>
                <p className="text-xs text-gray-600 leading-relaxed">
                  All conditions use <span className="font-semibold text-gray-800">AND logic</span> — every
                  configured condition must pass for authorization to proceed.
                </p>
              </div>
            )}

            {activeStep === 'credential' && (
              <div className="space-y-3">
                <p className="text-xs text-gray-600 leading-relaxed">
                  <span className="font-semibold text-gray-800">Credential Providers</span> are the
                  systems that produce the credential injected into requests. They answer: "What
                  credential does the Server Workload expect?"
                </p>
                <p className="text-xs text-gray-600 leading-relaxed">
                  Supported types include <span className="font-semibold text-gray-800">API Key</span>,{' '}
                  <span className="font-semibold text-gray-800">Username &amp; Password</span>,{' '}
                  <span className="font-semibold text-gray-800">OAuth 2.0 Client Credentials</span>,{' '}
                  <span className="font-semibold text-gray-800">AWS STS Federation</span>, and more.
                </p>
                <p className="text-xs text-gray-600 leading-relaxed">
                  Credentials are retrieved just-in-time and injected transparently. The Client
                  Workload never sees, stores, or manages the credential.
                </p>
              </div>
            )}

            {activeStep === 'server' && (
              <div className="space-y-3">
                <p className="text-xs text-gray-600 leading-relaxed">
                  <span className="font-semibold text-gray-800">Server Workloads</span> represent
                  the services, APIs, or databases that Client Workloads access. They're identified by
                  host, port, and protocol.
                </p>
                <p className="text-xs text-gray-600 leading-relaxed">
                  The <span className="font-semibold text-gray-800">authentication method</span> determines how
                  credentials are injected — HTTP Bearer, Basic, API Key header, database password
                  auth, AWS SigV4, or mTLS.
                </p>
                <p className="text-xs text-gray-600 leading-relaxed">
                  Dipole supports HTTP, PostgreSQL, MySQL, Redis, Snowflake, Redshift, Oracle, MCP,
                  and TCP passthrough protocols.
                </p>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

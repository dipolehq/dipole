import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { FileText } from 'lucide-react'
import { PageHeader } from '@/shared/components/PageHeader'
import { FloatingDropdown, FloatingInput } from '@/shared/components/FloatingInput'
import { CredentialProviderForm } from '@/shared/components/forms/CredentialProviderForm'
import {
  useCredentialProvider,
  useSaveCredentialProvider,
} from '@/features/credential-providers/hooks/useCredentialProviders'
import { apiErrorMessage } from '@/features/credential-providers/api/credentialProvidersApi'
import type { CredentialProviderFormValues } from '@/shared/components/forms/types'

const DETAIL_SECTIONS = [
  { value: 'details', label: 'Details' },
  { value: 'metrics', label: 'Metrics' },
  { value: 'logs', label: 'Logs' },
] as const

type DetailSection = (typeof DETAIL_SECTIONS)[number]['value']

export function CredentialProviderDetailPage() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const { data, isLoading, isError } = useCredentialProvider(id)
  const saveMutation = useSaveCredentialProvider(id ?? '')

  const [selectedSection, setSelectedSection] = useState<DetailSection>('details')
  const [sectionOpen, setSectionOpen] = useState(false)
  const [isEditing, setIsEditing] = useState(false)
  const [localValues, setLocalValues] = useState<CredentialProviderFormValues | null>(null)
  const [localStatus, setLocalStatus] = useState<string | null>(null)
  const [saveError, setSaveError] = useState<string | null>(null)

  if (!id || isError || (!isLoading && data === null)) {
    return (
      <div className="flex flex-col h-full">
        <div className="flex items-center justify-between px-4 shrink-0 h-[60px] border-b border-border bg-white">
          <PageHeader />
        </div>
        <div className="flex-1 flex items-center justify-center flex-col">
          <p className="text-sm text-gray-500">Credential provider not found.</p>
          <button
            onClick={() => navigate('/credential-providers')}
            className="mt-4 text-xs text-gray-600 hover:text-gray-700 font-medium"
          >
            ← Back to Credential Providers
          </button>
        </div>
      </div>
    )
  }

  if (isLoading || !data) return null

  const currentValues = localValues ?? data.values
  const status = localStatus ?? (data.metadata.isActive ? 'active' : 'inactive')
  const disabled = !isEditing
  const selectedLabel = DETAIL_SECTIONS.find((s) => s.value === selectedSection)?.label ?? ''

  async function handleEditToggle() {
    if (isEditing) {
      setSaveError(null)
      try {
        await saveMutation.mutateAsync({ values: currentValues, isActive: status !== 'inactive' })
        setIsEditing(false)
        setLocalValues(null)
        setLocalStatus(null)
      } catch (err) {
        setSaveError(apiErrorMessage(err, 'Failed to save credential provider'))
      }
    } else {
      setSaveError(null)
      if (data) setLocalValues({ ...data.values })
      setLocalStatus(data?.metadata.isActive ? 'active' : 'inactive')
      setIsEditing(true)
    }
  }

  function handleCancel() {
    if (isEditing) {
      setSaveError(null)
      setLocalValues(null)
      setLocalStatus(null)
      setIsEditing(false)
    } else {
      navigate('/credential-providers')
    }
  }

  function handleValuesChange(next: CredentialProviderFormValues) {
    setLocalValues(next)
  }

  return (
    <div className="flex flex-col h-full">
      {/* Header bar */}
      <div className="flex items-center justify-between px-4 shrink-0 h-[60px] border-b border-border bg-white">
        <PageHeader />
        <div className="flex items-center gap-3">
          <button
            type="button"
            onClick={handleEditToggle}
            className="px-4 py-2 bg-gray-800 hover:bg-gray-900 text-white text-sm font-medium rounded-md transition-colors"
          >
            {isEditing ? 'Save' : 'Edit'}
          </button>
          <button
            type="button"
            onClick={handleCancel}
            className="px-4 py-2 border border-gray-300 text-sm font-medium text-gray-600 hover:text-gray-900 hover:border-gray-400 rounded-md transition-colors"
          >
            {isEditing ? 'Cancel' : 'Back'}
          </button>
          {/* Section dropdown */}
          <div className="relative">
            <button
              type="button"
              onClick={() => setSectionOpen(!sectionOpen)}
              className="flex items-center gap-1.5 w-[160px] px-4 py-2 bg-[#28282B] hover:bg-[#333336] text-white text-sm font-medium rounded-md transition-colors"
            >
              <FileText size={14} className="shrink-0" />
              <span className="truncate">{selectedLabel}</span>
              <svg
                className={`w-3.5 h-3.5 ml-auto shrink-0 transition-transform ${sectionOpen ? 'rotate-180' : ''}`}
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                viewBox="0 0 24 24"
              >
                <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
              </svg>
            </button>
            {sectionOpen && (
              <div className="absolute top-full right-0 mt-1 bg-white border border-border rounded-md shadow-lg z-50 w-[180px] overflow-hidden">
                {DETAIL_SECTIONS.map((opt) => (
                  <button
                    key={opt.value}
                    type="button"
                    onClick={() => {
                      setSelectedSection(opt.value)
                      setSectionOpen(false)
                    }}
                    className={`w-full text-left px-4 py-2.5 text-sm transition-colors ${selectedSection === opt.value ? 'bg-gray-50 text-gray-700 font-medium' : 'text-gray-600 hover:bg-gray-50'}`}
                  >
                    {opt.label}
                  </button>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>

      {saveError && (
        <div className="px-6 py-2 bg-red-50 border-b border-red-200 text-sm text-red-700 shrink-0">
          {saveError}
        </div>
      )}

      {/* Content */}
      <div className="flex-1 overflow-y-auto bg-white">
        {selectedSection === 'details' && (
          <div className="flex h-full">
            {/* Form — left 60% */}
            <div className="w-[60%] px-6 py-6 overflow-y-auto">
              <div className="space-y-6">
                <CredentialProviderForm
                  values={currentValues}
                  onChange={handleValuesChange}
                  disabled={disabled}
                  statusSlot={
                    <FloatingDropdown
                      label="Status"
                      options={[
                        { value: 'active', label: 'Active' },
                        { value: 'inactive', label: 'Inactive' },
                      ]}
                      value={status}
                      onChange={setLocalStatus}
                      disabled={disabled}
                    />
                  }
                />
              </div>
            </div>

            {/* Help panel — right 40% */}
            <div className="w-[40%] px-6 py-6 overflow-y-auto">
              <div className="border border-border rounded-lg p-5 mb-5">
                <h4 className="text-sm font-semibold text-gray-800 mb-4">Metadata</h4>
                <div className="space-y-4">
                  <FloatingInput label="ID" value={data.metadata.entityId} disabled />
                  <FloatingInput
                    label="Created At"
                    value={new Date(data.metadata.createdAt).toLocaleString()}
                    disabled
                  />
                  <FloatingInput
                    label="Last Modified"
                    value={
                      data.metadata.lastModifiedAt
                        ? new Date(data.metadata.lastModifiedAt).toLocaleString()
                        : '—'
                    }
                    disabled
                  />
                </div>
              </div>

              <div className="border border-border rounded-lg p-5">
                <p className="text-xs text-gray-700 leading-relaxed mb-3">
                  <span className="font-semibold text-gray-700">Credential Providers</span> are
                  the systems that produce the credential injected into requests. They answer: "What
                  credential does the Server Workload expect?"
                </p>
                <p className="text-xs text-gray-700 leading-relaxed mb-3">
                  Supported types include <span className="font-semibold">API Key</span>,{' '}
                  <span className="font-semibold">Username &amp; Password</span>,{' '}
                  <span className="font-semibold">OAuth 2.0 Client Credentials</span>,{' '}
                  <span className="font-semibold">AWS STS Federation</span>, and more.
                </p>
                <p className="text-xs text-gray-700 leading-relaxed">
                  Credentials are retrieved just-in-time and injected transparently. The Client
                  Workload never sees, stores, or manages the credential.
                </p>
              </div>
            </div>
          </div>
        )}

        {selectedSection === 'metrics' && (
          <div className="flex items-center justify-center h-full">
            <p className="text-sm text-gray-400">Metrics — coming soon</p>
          </div>
        )}
        {selectedSection === 'logs' && (
          <div className="flex items-center justify-center h-full">
            <p className="text-sm text-gray-400">Logs — coming soon</p>
          </div>
        )}
      </div>
    </div>
  )
}

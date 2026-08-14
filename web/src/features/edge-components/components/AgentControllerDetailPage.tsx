import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { FileText } from 'lucide-react'
import { useQuery } from '@tanstack/react-query'
import { PageHeader } from '@/shared/components/PageHeader'
import { FloatingInput, FloatingTextarea, FloatingDropdown } from '@/shared/components/FloatingInput'
import { TagsField } from '@/shared/components/TagsField'
import { useAgentController, useSaveAgentController, useDeleteAgentController } from '@/features/edge-components/hooks/useAgentControllers'
import { apiErrorMessage } from '@/features/edge-components/api/agentControllersApi'
import { fetchTrustProviders } from '@/features/policies/api/entityApi'
import type { FormTag } from '@/shared/components/forms/types'

const DETAIL_SECTIONS = [
  { value: 'details', label: 'Details' },
  { value: 'metrics', label: 'Metrics' },
  { value: 'logs', label: 'Logs' },
] as const

type DetailSection = (typeof DETAIL_SECTIONS)[number]['value']

export function AgentControllerDetailPage() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()

  const { data: controller, isLoading } = useAgentController(id)
  const saveMutation = useSaveAgentController(id ?? '')
  const deleteMutation = useDeleteAgentController()

  const [selectedSection, setSelectedSection] = useState<DetailSection>('details')
  const [sectionOpen, setSectionOpen] = useState(false)
  const [isEditing, setIsEditing] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)

  // Form state
  const [status, setStatus] = useState('')
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [trustProviderId, setTrustProviderId] = useState('')
  const [allowedTlsHostname, setAllowedTlsHostname] = useState('')
  const [tags, setTags] = useState<FormTag[]>([])
  const [initialized, setInitialized] = useState(false)

  // Populate form when data loads
  if (controller && !initialized) {
    setStatus(controller.isActive ? 'active' : 'inactive')
    setName(controller.name)
    setDescription(controller.description ?? '')
    setTrustProviderId(controller.trustProviderId ?? '')
    setAllowedTlsHostname(controller.tlsHostname ?? '')
    setTags((controller.tags ?? []).map((t) => ({ key: t.key, value: t.value })))
    setInitialized(true)
  }

  const { data: tpData } = useQuery({
    queryKey: ['trust-providers', 'dropdown'],
    queryFn: () => fetchTrustProviders({ perPage: 100 }),
    staleTime: 60_000,
  })
  const trustProviderOptions = (tpData?.items ?? []).map((tp) => ({
    value: tp.id,
    label: tp.name,
  }))

  if (!id || (!isLoading && controller === null)) {
    return (
      <div className="flex flex-col h-full">
        <div className="flex items-center justify-between px-4 shrink-0 h-[60px] border-b border-border bg-white"><PageHeader /></div>
        <div className="flex-1 flex items-center justify-center flex-col">
          <p className="text-sm text-gray-500">Agent Controller not found.</p>
          <button onClick={() => navigate('/edge-components')} className="mt-4 text-xs text-gray-600 hover:text-gray-700 font-medium">← Back to Edge Components</button>
        </div>
      </div>
    )
  }

  const disabled = !isEditing
  const selectedLabel = DETAIL_SECTIONS.find((s) => s.value === selectedSection)?.label ?? ''

  async function handleEditToggle() {
    if (isEditing) {
      setError(null)
      try {
        await saveMutation.mutateAsync({
          name,
          description,
          isActive: status === 'active',
          trustProviderId: trustProviderId || undefined,
          tlsHostname: allowedTlsHostname || undefined,
          tags,
        })
        setIsEditing(false)
      } catch (err) {
        setError(apiErrorMessage(err, 'Failed to save agent controller'))
      }
    } else {
      setIsEditing(true)
    }
  }

  function handleCancel() {
    if (isEditing) {
      if (controller) {
        setStatus(controller.isActive ? 'active' : 'inactive')
        setName(controller.name)
        setDescription(controller.description ?? '')
        setTrustProviderId(controller.trustProviderId ?? '')
        setAllowedTlsHostname(controller.tlsHostname ?? '')
        setTags((controller.tags ?? []).map((t) => ({ key: t.key, value: t.value })))
      }
      setIsEditing(false)
    } else {
      navigate('/edge-components')
    }
  }

  async function handleDelete() {
    if (!id) return
    await deleteMutation.mutateAsync(id)
    navigate('/edge-components')
  }

  return (
    <div className="flex flex-col h-full">
      {error && (
        <div className="px-6 py-2 bg-red-50 border-b border-red-200 text-sm text-red-700 shrink-0">
          {error}
        </div>
      )}

      <div className="flex items-center justify-between px-4 shrink-0 h-[60px] border-b border-border bg-white">
        <PageHeader />
        <div className="flex items-center gap-3">
          <button type="button" onClick={handleEditToggle} className="px-4 py-2 bg-gray-800 hover:bg-gray-900 text-white text-sm font-medium rounded-md transition-colors">
            {isEditing ? 'Save' : 'Edit'}
          </button>
          <button type="button" onClick={handleCancel} className="px-4 py-2 border border-gray-300 text-sm font-medium text-gray-600 hover:text-gray-900 hover:border-gray-400 rounded-md transition-colors">
            {isEditing ? 'Cancel' : 'Back'}
          </button>
          {!isEditing && (
            <button type="button" onClick={() => setShowDeleteConfirm(true)} className="px-4 py-2 border border-red-300 text-sm font-medium text-red-600 hover:bg-red-50 hover:border-red-400 rounded-md transition-colors">
              Delete
            </button>
          )}
          <div className="relative">
            <button type="button" onClick={() => setSectionOpen(!sectionOpen)} className="flex items-center gap-1.5 w-[160px] px-4 py-2 bg-[#28282B] hover:bg-[#333336] text-white text-sm font-medium rounded-md transition-colors">
              <FileText size={14} className="shrink-0" />
              <span className="truncate">{selectedLabel}</span>
              <svg className={`w-3.5 h-3.5 ml-auto shrink-0 transition-transform ${sectionOpen ? 'rotate-180' : ''}`} fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" /></svg>
            </button>
            {sectionOpen && (
              <div className="absolute top-full right-0 mt-1 bg-white border border-border rounded-md shadow-lg z-50 w-[180px] overflow-hidden">
                {DETAIL_SECTIONS.map((opt) => (
                  <button key={opt.value} type="button" onClick={() => { setSelectedSection(opt.value); setSectionOpen(false) }} className={`w-full text-left px-4 py-2.5 text-sm transition-colors ${selectedSection === opt.value ? 'bg-gray-50 text-gray-700 font-medium' : 'text-gray-600 hover:bg-gray-50'}`}>{opt.label}</button>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto bg-white">
        {selectedSection === 'details' && (
          <div className="flex h-full">
            <div className="w-[60%] px-6 py-6 overflow-y-auto">
              <div className={`space-y-5 ${disabled ? 'opacity-60 pointer-events-none' : ''}`}>
                <div className="border border-border rounded-lg p-5">
                  <h4 className="text-sm font-semibold text-gray-800 mb-4">Details</h4>
                  <div className="space-y-4">
                    <FloatingDropdown label="Status" options={[{ value: 'active', label: 'Active' }, { value: 'inactive', label: 'Inactive' }]} value={status} onChange={setStatus} disabled={disabled} />
                    <FloatingInput label="Name" value={name} onChange={setName} disabled={disabled} required />
                    <FloatingTextarea label="Description (Optional)" value={description} onChange={setDescription} disabled={disabled} rows={2} />
                  </div>
                </div>

                <div className="border border-border rounded-lg p-5">
                  <h4 className="text-sm font-semibold text-gray-800 mb-4">Configuration</h4>
                  <div className="space-y-4">
                    <FloatingDropdown label="Trust Provider" options={trustProviderOptions} value={trustProviderId} onChange={setTrustProviderId} disabled={disabled} />
                    <FloatingInput label="Allowed TLS Hostname (Optional)" value={allowedTlsHostname} onChange={setAllowedTlsHostname} disabled={disabled} />
                  </div>
                </div>

                <div className="border border-border rounded-lg p-5">
                  <h4 className="text-sm font-semibold text-gray-800 mb-4">Tags</h4>
                  {disabled && tags.length === 0 ? (
                    <div className="px-3 py-2 border border-border rounded-md text-sm text-gray-500 bg-gray-50">No tags</div>
                  ) : (
                    <TagsField value={tags} onChange={setTags} disabled={disabled} />
                  )}
                </div>
              </div>
            </div>

            <div className="w-[40%] px-6 py-6 overflow-y-auto">
              <div className="sticky top-0 space-y-5">
                {controller && (
                  <div className="border border-border rounded-lg p-5">
                    <h4 className="text-sm font-semibold text-gray-800 mb-4">Metadata</h4>
                    <div className="space-y-4">
                      <div>
                        <label className="block text-[11px] text-gray-500 mb-1">ID</label>
                        <input type="text" readOnly value={controller.id} className="w-full px-3 py-2 border border-border rounded-md text-sm text-gray-700 bg-gray-50" />
                      </div>
                      <div>
                        <label className="block text-[11px] text-gray-500 mb-1">Created At</label>
                        <input type="text" readOnly value={new Date(controller.createdAt).toLocaleString()} className="w-full px-3 py-2 border border-border rounded-md text-sm text-gray-700 bg-gray-50" />
                      </div>
                      {controller.deviceCode && (
                        <div>
                          <label className="block text-[11px] text-gray-500 mb-1">Device Code</label>
                          <input type="text" readOnly value={controller.deviceCode} className="w-full px-3 py-2 border border-border rounded-md text-sm text-gray-700 bg-gray-50 font-mono" />
                        </div>
                      )}
                    </div>
                  </div>
                )}

                <div className="border border-border rounded-lg p-5">
                  <p className="text-xs text-gray-700 leading-relaxed mb-3">
                    <span className="font-semibold text-gray-700">Agent Controllers</span> manage Agent Proxy registration and token provisioning. They act as a registration broker between Agent Proxies and the Dipole Cloud.
                  </p>
                  <p className="text-xs text-gray-700 leading-relaxed mb-3">
                    Associating a <span className="font-semibold">Trust Provider</span> enables automated identity attestation during registration, removing the need for manual device codes.
                  </p>
                  <p className="text-xs text-gray-700 leading-relaxed">
                    The <span className="font-semibold">Allowed TLS Hostname</span> specifies the FQDN or wildcard domain included in the managed TLS certificate for secure communication.
                  </p>
                </div>
              </div>
            </div>
          </div>
        )}

        {selectedSection === 'metrics' && (
          <div className="flex items-center justify-center h-full"><p className="text-sm text-gray-400">Metrics — coming soon</p></div>
        )}
        {selectedSection === 'logs' && (
          <div className="flex items-center justify-center h-full"><p className="text-sm text-gray-400">Logs — coming soon</p></div>
        )}
      </div>

      {showDeleteConfirm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
          <div className="bg-white rounded-lg shadow-xl p-6 w-[400px]">
            <h3 className="text-sm font-semibold text-gray-900 mb-2">Delete Agent Controller</h3>
            <p className="text-xs text-gray-600 mb-5">
              Are you sure you want to delete this agent controller? This action cannot be undone.
            </p>
            <div className="flex items-center justify-end gap-3">
              <button onClick={() => setShowDeleteConfirm(false)} className="px-4 py-2 text-xs font-medium text-gray-600 border border-gray-300 rounded-md hover:bg-gray-50 transition-colors">Cancel</button>
              <button onClick={handleDelete} disabled={deleteMutation.isPending} className="px-4 py-2 text-xs font-medium text-white bg-red-600 hover:bg-red-700 rounded-md transition-colors disabled:opacity-50">
                {deleteMutation.isPending ? 'Deleting…' : 'Delete'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

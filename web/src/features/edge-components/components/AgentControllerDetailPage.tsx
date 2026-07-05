import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { FileText } from 'lucide-react'
import { PageHeader } from '@/shared/components/PageHeader'
import { FloatingInput, FloatingTextarea, FloatingDropdown } from '@/shared/components/FloatingInput'
import { TagsField } from '@/shared/components/TagsField'
import { agentControllersData } from '@/data/edge-components/agentControllers'

const TRUST_PROVIDER_OPTIONS = [
  { value: 'tp_1', label: 'AWS Production' },
  { value: 'tp_2', label: 'AWS Staging' },
  { value: 'tp_3', label: 'Kubernetes Production' },
]

const DETAIL_SECTIONS = [
  { value: 'details', label: 'Details' },
  { value: 'metrics', label: 'Metrics' },
  { value: 'logs', label: 'Logs' },
] as const

type DetailSection = (typeof DETAIL_SECTIONS)[number]['value']

export function AgentControllerDetailPage() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const controller = agentControllersData.find((ac) => ac.id === id)

  const [selectedSection, setSelectedSection] = useState<DetailSection>('details')
  const [sectionOpen, setSectionOpen] = useState(false)
  const [isEditing, setIsEditing] = useState(false)
  const [status, setStatus] = useState('active')
  const [name, setName] = useState(controller?.name ?? '')
  const [description, setDescription] = useState(controller?.description ?? '')
  const [trustProviderId, setTrustProviderId] = useState(controller?.trustProviderId ?? '')
  const [allowedTlsHostname, setAllowedTlsHostname] = useState(controller?.allowedTlsHostname ?? '')

  if (!controller) {
    return (
      <div className="flex flex-col h-full">
        <div className="flex items-center justify-between px-4 py-3 shrink-0 min-h-[36px] border-b border-border bg-white"><PageHeader /></div>
        <div className="flex-1 flex items-center justify-center flex-col">
          <p className="text-sm text-gray-500">Agent Controller not found.</p>
          <button onClick={() => navigate('/edge-components')} className="mt-4 text-xs text-orange-500 hover:text-orange-600 font-medium">← Back to Edge Components</button>
        </div>
      </div>
    )
  }

  const disabled = !isEditing
  const selectedLabel = DETAIL_SECTIONS.find((s) => s.value === selectedSection)?.label ?? ''

  function handleEditToggle() {
    if (isEditing) {
      console.info('[save-agent-controller]', { status, name, description, trustProviderId, allowedTlsHostname })
      setIsEditing(false)
    } else {
      setIsEditing(true)
    }
  }

  function handleCancel() {
    if (isEditing) {
      setName(controller?.name ?? '')
      setDescription(controller?.description ?? '')
      setTrustProviderId(controller?.trustProviderId ?? '')
      setAllowedTlsHostname(controller?.allowedTlsHostname ?? '')
      setStatus('active')
      setIsEditing(false)
    } else {
      navigate('/edge-components')
    }
  }

  return (
    <div className="flex flex-col h-full">
      {/* Header bar */}
      <div className="flex items-center justify-between px-4 py-3 shrink-0 min-h-[36px] border-b border-border bg-white">
        <PageHeader />
        <div className="flex items-center gap-3">
          <button type="button" onClick={handleEditToggle} className="px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white text-sm font-medium rounded-md transition-colors">{isEditing ? 'Save' : 'Edit'}</button>
          <button type="button" onClick={handleCancel} className="px-4 py-2 border border-gray-300 text-sm font-medium text-gray-600 hover:text-gray-900 hover:border-gray-400 rounded-md transition-colors">{isEditing ? 'Cancel' : 'Back'}</button>
          <div className="relative">
            <button type="button" onClick={() => setSectionOpen(!sectionOpen)} className="flex items-center gap-1.5 w-[160px] px-4 py-2 bg-[#28282B] hover:bg-[#333336] text-white text-sm font-medium rounded-md transition-colors">
              <FileText size={14} className="shrink-0" />
              <span className="truncate">{selectedLabel}</span>
              <svg className={`w-3.5 h-3.5 ml-auto shrink-0 transition-transform ${sectionOpen ? 'rotate-180' : ''}`} fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" /></svg>
            </button>
            {sectionOpen && (
              <div className="absolute top-full right-0 mt-1 bg-white border border-border rounded-md shadow-lg z-50 w-[180px] overflow-hidden">
                {DETAIL_SECTIONS.map((opt) => (
                  <button key={opt.value} type="button" onClick={() => { setSelectedSection(opt.value); setSectionOpen(false) }} className={`w-full text-left px-4 py-2.5 text-sm transition-colors ${selectedSection === opt.value ? 'bg-orange-50 text-orange-600 font-medium' : 'text-gray-600 hover:bg-gray-50'}`}>{opt.label}</button>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto bg-white">
        {selectedSection === 'details' && (
          <div className="flex h-full">
            {/* Form — left 60% */}
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
                    <FloatingDropdown label="Trust Provider" options={TRUST_PROVIDER_OPTIONS} value={trustProviderId} onChange={setTrustProviderId} disabled={disabled} />
                    <FloatingInput label="Allowed TLS Hostname (Optional)" value={allowedTlsHostname} onChange={setAllowedTlsHostname} disabled={disabled} />
                  </div>
                </div>

                <div className="border border-border rounded-lg p-5">
                  <h4 className="text-sm font-semibold text-gray-800 mb-4">Tags</h4>
                  {disabled ? (
                    <div className="px-3 py-2 border border-border rounded-md text-sm text-gray-500 bg-gray-50">No tags</div>
                  ) : (
                    <TagsField />
                  )}
                </div>
              </div>
            </div>

            {/* Help panel — right 40% */}
            <div className="w-[40%] px-6 py-6 overflow-y-auto">
              <div className="border border-border rounded-lg p-5 mb-5">
                <h4 className="text-sm font-semibold text-gray-800 mb-4">Metadata</h4>
                <div className="space-y-4">
                  <FloatingInput label="ID" value={controller.id} disabled />
                  <FloatingInput label="Version" value={controller.version} disabled />
                  <FloatingInput label="Last Modified" value={controller.modifiedAt} disabled />
                </div>
              </div>

              <div className="border border-border rounded-lg p-5">
                <p className="text-xs text-gray-700 leading-relaxed mb-3">
                  <span className="font-semibold text-orange-600">Agent Controllers</span> manage Agent Proxy registration and token provisioning. They act as a registration broker between Agent Proxies and the Dipole Cloud.
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
        )}

        {selectedSection === 'metrics' && (
          <div className="flex items-center justify-center h-full"><p className="text-sm text-gray-400">Metrics — coming soon</p></div>
        )}
        {selectedSection === 'logs' && (
          <div className="flex items-center justify-center h-full"><p className="text-sm text-gray-400">Logs — coming soon</p></div>
        )}
      </div>
    </div>
  )
}

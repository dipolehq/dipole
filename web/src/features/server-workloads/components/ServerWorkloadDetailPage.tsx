import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { FileText } from 'lucide-react'
import { PageHeader } from '@/shared/components/PageHeader'
import { FloatingDropdown, FloatingInput } from '@/shared/components/FloatingInput'
import { ServerWorkloadForm } from '@/shared/components/forms/ServerWorkloadForm'
import { useServerWorkload, useSaveServerWorkload } from '@/features/server-workloads/hooks/useServerWorkloads'
import type { ServerWorkloadFormValues } from '@/shared/components/forms/types'

const DETAIL_SECTIONS = [
  { value: 'details', label: 'Details' },
  { value: 'metrics', label: 'Metrics' },
  { value: 'logs', label: 'Logs' },
] as const

type DetailSection = (typeof DETAIL_SECTIONS)[number]['value']

export function ServerWorkloadDetailPage() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const { data, isLoading, isError } = useServerWorkload(id)
  const saveMutation = useSaveServerWorkload(id ?? '')

  const [selectedSection, setSelectedSection] = useState<DetailSection>('details')
  const [sectionOpen, setSectionOpen] = useState(false)
  const [isEditing, setIsEditing] = useState(false)
  const [localValues, setLocalValues] = useState<ServerWorkloadFormValues | null>(null)
  const [localStatus, setLocalStatus] = useState<string | null>(null)

  if (!id || isError || (!isLoading && data === null)) {
    return (
      <div className="flex flex-col h-full">
        <div className="flex items-center justify-between px-4 py-3 shrink-0 min-h-[36px] border-b border-border bg-white"><PageHeader /></div>
        <div className="flex-1 flex items-center justify-center flex-col">
          <p className="text-sm text-gray-500">Server workload not found.</p>
          <button onClick={() => navigate('/server-workloads')} className="mt-4 text-xs text-orange-500 hover:text-orange-600 font-medium">← Back to Server Workloads</button>
        </div>
      </div>
    )
  }

  if (isLoading || !data) return null

  const currentValues = localValues ?? data.values
  const status = localStatus ?? (data.metadata.isActive ? 'active' : 'inactive')
  const disabled = !isEditing
  const selectedLabel = DETAIL_SECTIONS.find((s) => s.value === selectedSection)?.label ?? ''

  function handleEditToggle() {
    if (isEditing) {
      void saveMutation.mutateAsync(currentValues).then(() => { setIsEditing(false); setLocalValues(null); setLocalStatus(null) })
    } else {
      if (data) setLocalValues({ ...data.values })
      setLocalStatus(data?.metadata.isActive ? 'active' : 'inactive')
      setIsEditing(true)
    }
  }

  function handleCancel() {
    if (isEditing) { setLocalValues(null); setLocalStatus(null); setIsEditing(false) }
    else navigate('/server-workloads')
  }

  function handleValuesChange(next: ServerWorkloadFormValues) { setLocalValues(next) }

  return (
    <div className="flex flex-col h-full">
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

      <div className="flex-1 overflow-y-auto bg-white">
        {selectedSection === 'details' && (
          <div className="flex h-full">
            <div className="w-[60%] px-6 py-6 overflow-y-auto">
              <div className="space-y-6">
                <ServerWorkloadForm
                  values={currentValues}
                  onChange={handleValuesChange}
                  disabled={disabled}
                  statusSlot={
                    <FloatingDropdown label="Status" options={[{ value: 'active', label: 'Active' }, { value: 'inactive', label: 'Inactive' }]} value={status} onChange={setLocalStatus} disabled={disabled} />
                  }
                />
              </div>
            </div>
            <div className="w-[40%] px-6 py-6 overflow-y-auto">
              <div className="border border-border rounded-lg p-5 mb-5">
                <h4 className="text-sm font-semibold text-gray-800 mb-4">Metadata</h4>
                <div className="space-y-4">
                  <FloatingInput label="ID" value={data.metadata.entityId} disabled />
                  <FloatingInput label="Created At" value={new Date(data.metadata.createdAt).toLocaleString()} disabled />
                  <FloatingInput label="Last Modified" value={data.metadata.lastModifiedAt ? new Date(data.metadata.lastModifiedAt).toLocaleString() : '—'} disabled />
                </div>
              </div>
              <div className="border border-border rounded-lg p-5">
                <p className="text-xs text-gray-700 leading-relaxed mb-3">
                  <span className="font-semibold text-orange-600">Server Workloads</span> represent the services, APIs, or databases that Client Workloads access. They're identified by host, port, and protocol.
                </p>
                <p className="text-xs text-gray-700 leading-relaxed mb-3">
                  The <span className="font-semibold">authentication method</span> determines how credentials are injected — HTTP Bearer, Basic, API Key header, database password auth, AWS SigV4, or mTLS.
                </p>
                <p className="text-xs text-gray-700 leading-relaxed">
                  Dipole supports HTTP, PostgreSQL, MySQL, Redis, Snowflake, Redshift, Oracle, MCP, and TCP passthrough protocols.
                </p>
              </div>
            </div>
          </div>
        )}
        {selectedSection === 'metrics' && (<div className="flex items-center justify-center h-full"><p className="text-sm text-gray-400">Metrics — coming soon</p></div>)}
        {selectedSection === 'logs' && (<div className="flex items-center justify-center h-full"><p className="text-sm text-gray-400">Logs — coming soon</p></div>)}
      </div>
    </div>
  )
}

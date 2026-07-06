import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { FileText } from 'lucide-react'
import { PageHeader } from '@/shared/components/PageHeader'
import { FloatingInput, FloatingTextarea, FloatingDropdown } from '@/shared/components/FloatingInput'
import { logStreamsData, LOG_STREAM_EVENT_TYPE_OPTIONS, LOG_STREAM_DESTINATION_OPTIONS } from '@/data/administration/logStreams'
import type { LogStreamEventType, LogStreamDestinationType } from '@/data/administration/logStreams'

const DETAIL_SECTIONS = [
  { value: 'details', label: 'Details' },
  { value: 'metrics', label: 'Metrics' },
  { value: 'logs', label: 'Logs' },
] as const

type DetailSection = (typeof DETAIL_SECTIONS)[number]['value']

export function LogStreamDetailPage() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const ls = logStreamsData.find((l) => l.id === id)

  const [selectedSection, setSelectedSection] = useState<DetailSection>('details')
  const [sectionOpen, setSectionOpen] = useState(false)
  const [isEditing, setIsEditing] = useState(false)
  const [name, setName] = useState(ls?.name ?? '')
  const [description, setDescription] = useState(ls?.description ?? '')
  const [eventType, setEventType] = useState<LogStreamEventType>(ls?.eventType ?? 'access_authorization')
  const [destinationType, setDestinationType] = useState<LogStreamDestinationType>(ls?.destinationType ?? 'aws_s3')
  const [s3Region, setS3Region] = useState(ls?.s3BucketRegion ?? '')
  const [s3Bucket, setS3Bucket] = useState(ls?.s3BucketName ?? '')
  const [s3Prefix, setS3Prefix] = useState(ls?.s3PathPrefix ?? '')
  const [gcsBucket, setGcsBucket] = useState(ls?.gcsBucketName ?? '')
  const [gcsAudience, setGcsAudience] = useState(ls?.gcsAudience ?? '')
  const [gcsEmail, setGcsEmail] = useState(ls?.gcsServiceAccountEmail ?? '')
  const [gcsLifetime, setGcsLifetime] = useState(ls?.gcsTokenLifetime ?? '3600')
  const [hostPort, setHostPort] = useState(ls?.hostPort ?? '')
  const [tls, setTls] = useState(ls?.tls ?? false)
  const [tlsVerification, setTlsVerification] = useState(ls?.tlsVerification ?? 'full')
  const [apiKey, setApiKey] = useState(ls?.apiKey ?? '')
  const [authToken, setAuthToken] = useState(ls?.authToken ?? '')
  const [sourceName, setSourceName] = useState(ls?.sourceName ?? '')

  if (!ls) {
    return (
      <div className="flex flex-col h-full">
        <div className="flex items-center justify-between px-4 py-3 shrink-0 min-h-[36px] border-b border-border bg-white"><PageHeader /></div>
        <div className="flex-1 flex items-center justify-center flex-col">
          <p className="text-sm text-gray-500">Log Stream not found.</p>
          <button onClick={() => navigate('/administration')} className="mt-4 text-xs text-orange-500 hover:text-orange-600 font-medium">← Back to Administration</button>
        </div>
      </div>
    )
  }

  const disabled = !isEditing
  const selectedLabel = DETAIL_SECTIONS.find((s) => s.value === selectedSection)?.label ?? ''

  function handleEditToggle() {
    if (isEditing) {
      console.info('[save-log-stream]', { name, description, eventType, destinationType })
      setIsEditing(false)
    } else { setIsEditing(true) }
  }

  function handleCancel() {
    if (isEditing) {
      setName(ls?.name ?? ''); setDescription(ls?.description ?? ''); setIsEditing(false)
    } else { navigate('/administration') }
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
            <div className="w-[60%] px-6 py-6 overflow-y-auto">
              <div className={`space-y-5 ${disabled ? 'opacity-60 pointer-events-none' : ''}`}>
                <div className="border border-border rounded-lg p-5">
                  <h4 className="text-sm font-semibold text-gray-800 mb-4">Stream Details</h4>
                  <div className="space-y-4">
                    <FloatingInput label="Name" value={name} onChange={setName} disabled={disabled} required />
                    <FloatingTextarea label="Description (Optional)" value={description} onChange={setDescription} disabled={disabled} rows={2} />
                    <FloatingDropdown label="Event Type" options={LOG_STREAM_EVENT_TYPE_OPTIONS.map((o) => ({ value: o.value, label: o.label }))} value={eventType} onChange={(v) => setEventType(v as LogStreamEventType)} disabled={disabled} />
                    <FloatingDropdown label="Destination Type" options={LOG_STREAM_DESTINATION_OPTIONS.map((o) => ({ value: o.value, label: o.label }))} value={destinationType} onChange={(v) => setDestinationType(v as LogStreamDestinationType)} disabled={disabled} />
                  </div>
                </div>

                {destinationType === 'aws_s3' && (
                  <div className="border border-border rounded-lg p-5">
                    <h4 className="text-sm font-semibold text-gray-800 mb-4">AWS S3 Configuration</h4>
                    <div className="space-y-4">
                      <FloatingInput label="S3 Bucket Region" value={s3Region} onChange={setS3Region} disabled={disabled} />
                      <FloatingInput label="S3 Bucket Name" value={s3Bucket} onChange={setS3Bucket} disabled={disabled} />
                      <FloatingInput label="S3 Path Prefix" value={s3Prefix} onChange={setS3Prefix} disabled={disabled} />
                      <div className="bg-gray-50 rounded-md p-4">
                        <label className="block text-[10px] text-gray-500 mb-0.5">Source AWS Account ID</label>
                        <div className="text-xs text-gray-700 font-mono">123456789012</div>
                      </div>
                    </div>
                  </div>
                )}

                {destinationType === 'gcs_bucket' && (
                  <div className="border border-border rounded-lg p-5">
                    <h4 className="text-sm font-semibold text-gray-800 mb-4">GCS Configuration</h4>
                    <div className="space-y-4">
                      <div className="bg-gray-50 rounded-md p-4">
                        <label className="block text-[10px] text-gray-500 mb-0.5">OIDC Issuer URL</label>
                        <div className="text-xs text-gray-700 font-mono">https://acme.id.ap-south-1.dipolehq.com/</div>
                      </div>
                      <FloatingInput label="Bucket Name" value={gcsBucket} onChange={setGcsBucket} disabled={disabled} />
                      <FloatingInput label="Audience" value={gcsAudience} onChange={setGcsAudience} disabled={disabled} />
                      <FloatingInput label="Service Account Email" value={gcsEmail} onChange={setGcsEmail} disabled={disabled} />
                      <FloatingInput label="Token Lifetime (seconds)" value={gcsLifetime} onChange={setGcsLifetime} disabled={disabled} />
                    </div>
                  </div>
                )}

                {destinationType === 'crowdstrike_siem' && (
                  <div className="border border-border rounded-lg p-5">
                    <h4 className="text-sm font-semibold text-gray-800 mb-4">CrowdStrike SIEM Configuration</h4>
                    <div className="space-y-4">
                      <div className="relative">
                        <FloatingInput label="Host/Port" value={hostPort} onChange={setHostPort} disabled={disabled} />
                        <label className="absolute right-3 top-1/2 -translate-y-1/2 flex items-center gap-1.5">
                          <input type="checkbox" checked={tls} onChange={(e) => setTls(e.target.checked)} disabled={disabled} className="w-4 h-4 accent-orange-500 cursor-pointer disabled:opacity-60" />
                          <span className="text-sm text-gray-700">TLS</span>
                        </label>
                      </div>
                      <FloatingDropdown label="TLS Verification" options={[{ value: 'full', label: 'Full' }, { value: 'none', label: 'None' }]} value={tlsVerification} onChange={setTlsVerification} disabled={disabled} />
                      <FloatingInput label="API Key" value={apiKey} onChange={setApiKey} disabled={disabled} type="password" />
                      <FloatingInput label="Source Name" value={sourceName} onChange={setSourceName} disabled={disabled} />
                    </div>
                  </div>
                )}

                {destinationType === 'splunk_siem' && (
                  <div className="border border-border rounded-lg p-5">
                    <h4 className="text-sm font-semibold text-gray-800 mb-4">Splunk SIEM Configuration</h4>
                    <div className="space-y-4">
                      <div className="relative">
                        <FloatingInput label="Host/Port" value={hostPort} onChange={setHostPort} disabled={disabled} />
                        <label className="absolute right-3 top-1/2 -translate-y-1/2 flex items-center gap-1.5">
                          <input type="checkbox" checked={tls} onChange={(e) => setTls(e.target.checked)} disabled={disabled} className="w-4 h-4 accent-orange-500 cursor-pointer disabled:opacity-60" />
                          <span className="text-sm text-gray-700">TLS</span>
                        </label>
                      </div>
                      <FloatingDropdown label="TLS Verification" options={[{ value: 'full', label: 'Full' }, { value: 'none', label: 'None' }]} value={tlsVerification} onChange={setTlsVerification} disabled={disabled} />
                      <FloatingInput label="Authentication Token" value={authToken} onChange={setAuthToken} disabled={disabled} type="password" />
                      <FloatingInput label="Source Name" value={sourceName} onChange={setSourceName} disabled={disabled} />
                    </div>
                  </div>
                )}
              </div>
            </div>

            {/* Right panel — 40% */}
            <div className="w-[40%] px-6 py-6 overflow-y-auto">
              <div className="border border-border rounded-lg p-5 mb-5">
                <h4 className="text-sm font-semibold text-gray-800 mb-4">Metadata</h4>
                <div className="space-y-4">
                  <FloatingInput label="ID" value={ls.id} disabled />
                  <FloatingInput label="Status" value={ls.isActive ? 'Active' : 'Inactive'} disabled />
                  <FloatingInput label="Event Type" value={LOG_STREAM_EVENT_TYPE_OPTIONS.find((o) => o.value === ls.eventType)?.label ?? ''} disabled />
                </div>
              </div>
              <div className="border border-border rounded-lg p-5">
                <p className="text-xs text-gray-700 leading-relaxed mb-3">
                  <span className="font-semibold text-orange-600">Log Streams</span> export access authorization events, audit logs, or workload events to external SIEM systems for centralized monitoring and compliance.
                </p>
                <p className="text-xs text-gray-700 leading-relaxed">
                  Choose a <span className="font-semibold">Destination Type</span> based on where you want events delivered — AWS S3, GCS, CrowdStrike, or Splunk.
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

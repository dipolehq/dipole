import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { FileText } from 'lucide-react'
import { PageHeader } from '@/shared/components/PageHeader'
import { FloatingInput, FloatingTextarea, FloatingDropdown } from '@/shared/components/FloatingInput'
import { LOG_STREAM_EVENT_TYPE_OPTIONS, LOG_STREAM_DESTINATION_OPTIONS } from '@/shared/constants/administration'
import { useLogStream, useSaveLogStream, useDeleteLogStream } from '@/features/administration/hooks/useLogStreams'
import { apiErrorMessage } from '@/features/administration/api/logStreamsApi'
import type { LogStreamEventType, LogStreamDestinationType } from '@/shared/constants/administration'

const DETAIL_SECTIONS = [
  { value: 'details', label: 'Details' },
  { value: 'metrics', label: 'Metrics' },
  { value: 'logs', label: 'Logs' },
] as const

type DetailSection = (typeof DETAIL_SECTIONS)[number]['value']

export function LogStreamDetailPage() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()

  const { data: ls, isLoading } = useLogStream(id)
  const saveMutation = useSaveLogStream(id ?? '')
  const deleteMutation = useDeleteLogStream()

  const [selectedSection, setSelectedSection] = useState<DetailSection>('details')
  const [sectionOpen, setSectionOpen] = useState(false)
  const [isEditing, setIsEditing] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)

  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [eventType, setEventType] = useState<LogStreamEventType>('access_authorization')
  const [destinationType, setDestinationType] = useState<LogStreamDestinationType>('aws_s3')
  const [s3Region, setS3Region] = useState('')
  const [s3Bucket, setS3Bucket] = useState('')
  const [s3Prefix, setS3Prefix] = useState('')
  const [gcsBucket, setGcsBucket] = useState('')
  const [gcsPathPrefix, setGcsPathPrefix] = useState('')
  const [gcsAudience, setGcsAudience] = useState('')
  const [gcsEmail, setGcsEmail] = useState('')
  const [gcsLifetime, setGcsLifetime] = useState('3600')
  const [hostPort, setHostPort] = useState('')
  const [tls, setTls] = useState(false)
  const [tlsVerification, setTlsVerification] = useState('full')
  const [apiKey, setApiKey] = useState('')
  const [authToken, setAuthToken] = useState('')
  const [sourceName, setSourceName] = useState('')
  const [initialized, setInitialized] = useState(false)

  if (ls && !initialized) {
    const config = (ls.config ?? {}) as Record<string, unknown>
    setName(ls.name)
    setDescription(ls.description)
    setEventType(ls.eventType as LogStreamEventType)
    setDestinationType(ls.destinationType as LogStreamDestinationType)
    setS3Region((config.s3BucketRegion as string) ?? '')
    setS3Bucket((config.s3BucketName as string) ?? '')
    setS3Prefix((config.s3PathPrefix as string) ?? '')
    setGcsBucket((config.gcsBucketName as string) ?? '')
    setGcsPathPrefix((config.gcsPathPrefix as string) ?? '')
    setGcsAudience((config.gcsAudience as string) ?? '')
    setGcsEmail((config.gcsServiceAccountEmail as string) ?? '')
    setGcsLifetime((config.gcsTokenLifetime as string) ?? '3600')
    setHostPort((config.hostPort as string) ?? '')
    setTls((config.tls as boolean) ?? false)
    setTlsVerification((config.tlsVerification as string) ?? 'full')
    setApiKey((config.apiKey as string) ?? '')
    setAuthToken((config.authToken as string) ?? '')
    setSourceName((config.sourceName as string) ?? '')
    setInitialized(true)
  }

  if (initialized && ls && ls.id !== id) {
    setInitialized(false)
  }

  if (!id || (!isLoading && ls === null)) {
    return (
      <div className="flex flex-col h-full">
        <div className="flex items-center justify-between px-4 shrink-0 h-[60px] border-b border-border bg-white"><PageHeader /></div>
        <div className="flex-1 flex items-center justify-center flex-col">
          <p className="text-sm text-gray-500">Log Stream not found.</p>
          <button onClick={() => navigate('/administration')} className="mt-4 text-xs text-orange-500 hover:text-orange-600 font-medium">← Back to Administration</button>
        </div>
      </div>
    )
  }

  const disabled = !isEditing
  const selectedLabel = DETAIL_SECTIONS.find((s) => s.value === selectedSection)?.label ?? ''

  async function handleEditToggle() {
    if (isEditing) {
      setError(null)
      const config: Record<string, unknown> = {}
      if (destinationType === 'aws_s3') {
        config.s3BucketRegion = s3Region; config.s3BucketName = s3Bucket; config.s3PathPrefix = s3Prefix
      } else if (destinationType === 'gcs_bucket') {
        config.gcsBucketName = gcsBucket; config.gcsPathPrefix = gcsPathPrefix; config.gcsAudience = gcsAudience; config.gcsServiceAccountEmail = gcsEmail; config.gcsTokenLifetime = gcsLifetime
      } else if (destinationType === 'crowdstrike_siem') {
        config.hostPort = hostPort; config.tls = tls; config.tlsVerification = tlsVerification; config.apiKey = apiKey; config.sourceName = sourceName
      } else if (destinationType === 'splunk_siem') {
        config.hostPort = hostPort; config.tls = tls; config.tlsVerification = tlsVerification; config.authToken = authToken; config.sourceName = sourceName
      }
      try {
        await saveMutation.mutateAsync({ name, description, isActive: true, eventType, destinationType, config })
        setIsEditing(false)
      } catch (err) {
        setError(apiErrorMessage(err, 'Failed to save log stream'))
      }
    } else { setIsEditing(true) }
  }

  function handleCancel() {
    if (isEditing) {
      if (ls) {
        const config = (ls.config ?? {}) as Record<string, unknown>
        setName(ls.name); setDescription(ls.description)
        setEventType(ls.eventType as LogStreamEventType); setDestinationType(ls.destinationType as LogStreamDestinationType)
        setS3Region((config.s3BucketRegion as string) ?? ''); setS3Bucket((config.s3BucketName as string) ?? ''); setS3Prefix((config.s3PathPrefix as string) ?? '')
        setGcsBucket((config.gcsBucketName as string) ?? ''); setGcsPathPrefix((config.gcsPathPrefix as string) ?? ''); setGcsAudience((config.gcsAudience as string) ?? '')
        setGcsEmail((config.gcsServiceAccountEmail as string) ?? ''); setGcsLifetime((config.gcsTokenLifetime as string) ?? '3600')
        setHostPort((config.hostPort as string) ?? ''); setTls((config.tls as boolean) ?? false)
        setTlsVerification((config.tlsVerification as string) ?? 'full')
        setApiKey((config.apiKey as string) ?? ''); setAuthToken((config.authToken as string) ?? ''); setSourceName((config.sourceName as string) ?? '')
      }
      setIsEditing(false)
    } else { navigate('/administration') }
  }

  async function handleDelete() {
    if (!id) return
    try {
      await deleteMutation.mutateAsync(id)
      navigate('/administration')
    } catch (err) {
      setError(apiErrorMessage(err, 'Failed to delete log stream'))
    }
  }

  return (
    <div className="flex flex-col h-full">
      {error && (
        <div className="px-6 py-2 bg-red-50 border-b border-red-200 text-sm text-red-700 shrink-0">
          {error}
        </div>
      )}
      {/* Header bar */}
      <div className="flex items-center justify-between px-4 shrink-0 h-[60px] border-b border-border bg-white">
        <PageHeader />
        <div className="flex items-center gap-3">
          <button type="button" onClick={handleEditToggle} className="px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white text-sm font-medium rounded-md transition-colors">{isEditing ? 'Save' : 'Edit'}</button>
          <button type="button" onClick={handleCancel} className="px-4 py-2 border border-gray-300 text-sm font-medium text-gray-600 hover:text-gray-900 hover:border-gray-400 rounded-md transition-colors">{isEditing ? 'Cancel' : 'Back'}</button>
          {!isEditing && (
            <button type="button" onClick={() => setShowDeleteConfirm(true)} className="px-4 py-2 border border-red-300 text-sm font-medium text-red-600 hover:bg-red-50 hover:border-red-400 rounded-md transition-colors">Delete</button>
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
                      <FloatingInput label="Path Prefix" value={gcsPathPrefix} onChange={setGcsPathPrefix} disabled={disabled} />
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
                  {ls && (
                    <>
                      <FloatingInput label="ID" value={ls.id} disabled />
                      <FloatingInput label="Status" value={ls.isActive ? 'Active' : 'Inactive'} disabled />
                      <FloatingInput label="Created At" value={new Date(ls.createdAt).toLocaleString()} disabled />
                    </>
                  )}
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

      {showDeleteConfirm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
          <div className="bg-white rounded-lg shadow-xl p-6 w-[400px]">
            <h3 className="text-sm font-semibold text-gray-900 mb-2">Delete Log Stream</h3>
            <p className="text-xs text-gray-600 mb-5">
              Are you sure you want to delete this log stream? Events will no longer be exported to this destination.
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

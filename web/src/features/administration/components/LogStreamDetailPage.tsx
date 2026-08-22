import { useState } from 'react'
import { identityIssuerUrl, dipoleAwsAccountId } from '@/shared/lib/platform-urls'
import { useNavigate, useParams } from 'react-router-dom'
import { Pencil, Trash2 } from 'lucide-react'
import { FloatingInput, FloatingTextarea, FloatingDropdown } from '@/shared/components/FloatingInput'
import { DeleteConfirmDialog } from '@/shared/components/ui/DeleteConfirmDialog'
import { LOG_STREAM_EVENT_TYPE_OPTIONS, LOG_STREAM_DESTINATION_OPTIONS } from '@/shared/constants/administration'
import { useLogStream, useSaveLogStream, useDeleteLogStream } from '@/features/administration/hooks/useLogStreams'
import { apiErrorMessage } from '@/features/administration/api/logStreamsApi'
import type { LogStreamEventType, LogStreamDestinationType } from '@/shared/constants/administration'

export function LogStreamDetailPage() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()

  const { data: ls, isLoading } = useLogStream(id)
  const saveMutation = useSaveLogStream(id ?? '')
  const deleteMutation = useDeleteLogStream()

  const [isEditing, setIsEditing] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)

  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [lsStatus, setLsStatus] = useState('active')
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
    setLsStatus(ls.isActive ? 'active' : 'inactive')
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
        <div className="flex-1 flex items-center justify-center">
          <p className="text-sm text-gray-400">Log Stream not found.</p>
        </div>
      </div>
    )
  }

  const disabled = !isEditing
  const displayName = name || 'Untitled'
  const statusValue = lsStatus

  async function handleSave() {
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
      await saveMutation.mutateAsync({ name, description, isActive: lsStatus === 'active', eventType, destinationType, config })
      setIsEditing(false)
    } catch (err) {
      setError(apiErrorMessage(err, 'Failed to save log stream'))
    }
  }

  function handleEdit() {
    setError(null)
    setIsEditing(true)
  }

  function handleCancel() {
    setError(null)
    if (ls) {
      const config = (ls.config ?? {}) as Record<string, unknown>
      setName(ls.name); setDescription(ls.description)
      setLsStatus(ls.isActive ? 'active' : 'inactive')
      setEventType(ls.eventType as LogStreamEventType); setDestinationType(ls.destinationType as LogStreamDestinationType)
      setS3Region((config.s3BucketRegion as string) ?? ''); setS3Bucket((config.s3BucketName as string) ?? ''); setS3Prefix((config.s3PathPrefix as string) ?? '')
      setGcsBucket((config.gcsBucketName as string) ?? ''); setGcsPathPrefix((config.gcsPathPrefix as string) ?? ''); setGcsAudience((config.gcsAudience as string) ?? '')
      setGcsEmail((config.gcsServiceAccountEmail as string) ?? ''); setGcsLifetime((config.gcsTokenLifetime as string) ?? '3600')
      setHostPort((config.hostPort as string) ?? ''); setTls((config.tls as boolean) ?? false)
      setTlsVerification((config.tlsVerification as string) ?? 'full')
      setApiKey((config.apiKey as string) ?? ''); setAuthToken((config.authToken as string) ?? ''); setSourceName((config.sourceName as string) ?? '')
    }
    setIsEditing(false)
  }

  async function handleDelete() {
    if (!id) return
    try {
      await deleteMutation.mutateAsync(id)
      navigate('/administration/log-streams')
    } catch (err) {
      setError(apiErrorMessage(err, 'Failed to delete log stream'))
    }
  }

  return (
    <div className="flex flex-col h-full">

      <div className="flex items-center justify-between gap-3 px-4 py-3 shrink-0">
        <div className="flex items-center gap-3">
          <h2 className="text-sm font-medium text-gray-900">{displayName}</h2>
          <span className={`text-xs font-medium px-2 py-0.5 rounded ${statusValue === 'active' ? 'bg-green-50 text-green-700' : 'bg-gray-100 text-gray-500'}`}>
            {statusValue === 'active' ? 'Active' : 'Inactive'}
          </span>
        </div>
        <div className="flex items-center gap-2">
          {isEditing ? (
            <>
              <button type="button" onClick={handleSave} disabled={saveMutation.isPending} className="inline-flex items-center gap-2 h-9 px-4 rounded-md border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors disabled:opacity-50">
                {saveMutation.isPending ? 'Saving…' : 'Save'}
              </button>
              <button type="button" onClick={handleCancel} className="inline-flex items-center gap-2 h-9 px-4 rounded-md border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors">
                Cancel
              </button>
            </>
          ) : (
            <>
              <button type="button" onClick={handleEdit} className="inline-flex items-center gap-2 h-9 px-4 rounded-md border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors">
                <Pencil size={14} strokeWidth={2} className="opacity-60" /> Edit
              </button>
              <button type="button" onClick={() => setDeleteDialogOpen(true)} className="inline-flex items-center gap-2 h-9 px-4 rounded-md border border-red-200 bg-white text-sm font-medium text-red-600 hover:bg-red-50 transition-colors">
                <Trash2 size={14} strokeWidth={2} className="opacity-60" /> Delete
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

      <div className="flex-1 min-h-0 mx-4 border border-border bg-white overflow-y-auto">
        <div className="grid grid-cols-[1fr_380px] h-full">
          <div className="px-6 py-5 overflow-y-auto border-r border-border">
            <fieldset disabled={disabled} className={disabled ? 'opacity-60 pointer-events-none' : ''}>
              <div className="space-y-6">
                <div>
                  <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider mb-4">Stream Details</h4>
                  <div className="space-y-4">
                    <FloatingInput label="Name" value={name} onChange={setName} disabled={disabled} required />
                    <FloatingTextarea label="Description (Optional)" value={description} onChange={setDescription} disabled={disabled} rows={2} />
                    <FloatingDropdown label="Event Type" options={LOG_STREAM_EVENT_TYPE_OPTIONS.map((o) => ({ value: o.value, label: o.label }))} value={eventType} onChange={(v) => setEventType(v as LogStreamEventType)} disabled={disabled} />
                    <FloatingDropdown label="Destination Type" options={LOG_STREAM_DESTINATION_OPTIONS.map((o) => ({ value: o.value, label: o.label }))} value={destinationType} onChange={(v) => setDestinationType(v as LogStreamDestinationType)} disabled={disabled} />
                  </div>
                </div>

                {destinationType === 'aws_s3' && (
                  <div>
                    <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider mb-4">AWS S3 Configuration</h4>
                    <div className="space-y-4">
                      <FloatingInput label="S3 Bucket Region" value={s3Region} onChange={setS3Region} disabled={disabled} />
                      <FloatingInput label="S3 Bucket Name" value={s3Bucket} onChange={setS3Bucket} disabled={disabled} />
                      <FloatingInput label="S3 Path Prefix" value={s3Prefix} onChange={setS3Prefix} disabled={disabled} />
                      <div className="bg-gray-50 rounded-md p-4">
                        <label className="block text-[10px] text-gray-500 mb-0.5">Source AWS Account ID</label>
                        <div className="text-xs text-gray-700 font-mono">{dipoleAwsAccountId()}</div>
                      </div>
                    </div>
                  </div>
                )}

                {destinationType === 'gcs_bucket' && (
                  <div>
                    <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider mb-4">GCS Configuration</h4>
                    <div className="space-y-4">
                      <div className="bg-gray-50 rounded-md p-4">
                        <label className="block text-[10px] text-gray-500 mb-0.5">OIDC Issuer URL</label>
                        <div className="text-xs text-gray-700 font-mono">{identityIssuerUrl('<tenant>')}</div>
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
                  <div>
                    <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider mb-4">CrowdStrike SIEM Configuration</h4>
                    <div className="space-y-4">
                      <div className="relative">
                        <FloatingInput label="Host/Port" value={hostPort} onChange={setHostPort} disabled={disabled} />
                        <label className="absolute right-3 top-1/2 -translate-y-1/2 flex items-center gap-1.5">
                          <input type="checkbox" checked={tls} onChange={(e) => setTls(e.target.checked)} disabled={disabled} className="w-4 h-4 accent-gray-600 cursor-pointer disabled:opacity-60" />
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
                  <div>
                    <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider mb-4">Splunk SIEM Configuration</h4>
                    <div className="space-y-4">
                      <div className="relative">
                        <FloatingInput label="Host/Port" value={hostPort} onChange={setHostPort} disabled={disabled} />
                        <label className="absolute right-3 top-1/2 -translate-y-1/2 flex items-center gap-1.5">
                          <input type="checkbox" checked={tls} onChange={(e) => setTls(e.target.checked)} disabled={disabled} className="w-4 h-4 accent-gray-600 cursor-pointer disabled:opacity-60" />
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
            </fieldset>
          </div>

          <div className="px-5 py-5 overflow-y-auto">
            <div className="space-y-4">
              <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider">Metadata</h4>
              {ls && (
                <>
                  <FloatingInput label="ID" value={ls.id} disabled />
                  <FloatingInput label="Created" value={new Date(ls.createdAt).toLocaleString('en-IN', { dateStyle: 'medium', timeStyle: 'short' })} disabled />
                </>
              )}
            </div>
          </div>
        </div>
      </div>

      <DeleteConfirmDialog
        open={deleteDialogOpen}
        onClose={() => setDeleteDialogOpen(false)}
        onConfirm={() => { setDeleteDialogOpen(false); handleDelete() }}
        resourceName={displayName}
        resourceId={id ?? ''}
        resourceType="log stream"
        loading={deleteMutation.isPending}
      />
    </div>
  )
}

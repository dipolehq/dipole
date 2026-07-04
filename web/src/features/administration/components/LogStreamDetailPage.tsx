import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { PageHeader } from '@/shared/components/PageHeader'
import { Dropdown } from '@/shared/components/Dropdown'
import { logStreamsData, LOG_STREAM_EVENT_TYPE_OPTIONS, LOG_STREAM_DESTINATION_OPTIONS } from '@/data/administration/logStreams'
import type { LogStreamEventType, LogStreamDestinationType } from '@/data/administration/logStreams'

type TabKey = 'details' | 'configuration'

export function LogStreamDetailPage() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const ls = logStreamsData.find((l) => l.id === id)

  const [isEditing, setIsEditing] = useState(false)
  const [activeTab, setActiveTab] = useState<TabKey>('details')
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
      <div className="p-6">
        <PageHeader />
        <p className="mt-6 text-sm text-gray-500">Log Stream not found.</p>
        <button onClick={() => navigate('/administration')} className="mt-4 text-xs text-orange-500 hover:text-orange-600 font-medium">← Back to Administration</button>
      </div>
    )
  }

  const disabled = !isEditing

  function handleEditToggle() {
    if (isEditing) {
      console.info('[save-log-stream]', { name, description, eventType, destinationType })
      setIsEditing(false)
    } else {
      setIsEditing(true)
    }
  }

  function handleCancel() {
    if (isEditing) {
      setName(ls?.name ?? '')
      setDescription(ls?.description ?? '')
      setIsEditing(false)
    } else {
      navigate('/administration')
    }
  }

  return (
    <div className="px-4 py-3 flex flex-col h-full">
      <div className="flex items-center justify-between mb-4 shrink-0 min-h-[36px]">
        <PageHeader />
      </div>

      <div className="bg-white rounded-lg border border-border flex-1 flex min-h-0">
        <nav className="w-48 border-r border-border py-3 shrink-0 flex flex-col">
          <div className="flex-1">
            <button type="button" onClick={() => setActiveTab('details')} className={`w-full text-left px-5 py-2.5 text-xs font-medium transition-colors relative ${activeTab === 'details' ? 'text-orange-600 bg-orange-50' : 'text-gray-500 hover:text-gray-800 hover:bg-gray-50'}`}>
              {activeTab === 'details' && <span className="absolute left-0 top-0 bottom-0 w-0.5 bg-orange-500 rounded-r" />}
              <span>Stream Details</span>
            </button>
            <button type="button" onClick={() => setActiveTab('configuration')} className={`w-full text-left px-5 py-2.5 text-xs font-medium transition-colors relative ${activeTab === 'configuration' ? 'text-orange-600 bg-orange-50' : 'text-gray-500 hover:text-gray-800 hover:bg-gray-50'}`}>
              {activeTab === 'configuration' && <span className="absolute left-0 top-0 bottom-0 w-0.5 bg-orange-500 rounded-r" />}
              <span>Stream Configuration</span>
            </button>
          </div>
          <div className="px-4 py-3 border-t border-border space-y-2">
            <button type="button" onClick={handleEditToggle} className="w-full px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors">
              {isEditing ? 'Save Log Stream' : 'Edit Log Stream'}
            </button>
            <button type="button" onClick={handleCancel} className="w-full px-4 py-2 border border-gray-300 text-xs font-medium text-gray-600 hover:text-gray-900 hover:border-gray-400 rounded-md transition-colors">
              {isEditing ? 'Cancel' : 'Back'}
            </button>
          </div>
        </nav>

        <div className="flex-1 overflow-y-auto p-6">
          {activeTab === 'details' ? (
            <div>
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-sm font-semibold text-gray-900">Stream Details</h3>
                {isEditing && <button type="button" onClick={handleEditToggle} className="px-4 py-1.5 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors">Save</button>}
              </div>
              <div className="border-b border-border mb-6" />
              <div className="space-y-4">
                <div>
                  <label className="block text-xs text-gray-500 mb-1">Name</label>
                  <input type="text" value={name} onChange={(e) => setName(e.target.value)} disabled={disabled} className={`w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400 ${disabled ? 'bg-gray-50 text-gray-600' : ''}`} />
                </div>
                <div>
                  <label className="block text-xs text-gray-500 mb-1">Description</label>
                  <textarea value={description} onChange={(e) => setDescription(e.target.value)} disabled={disabled} rows={2} className={`w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400 resize-y ${disabled ? 'bg-gray-50 text-gray-600' : ''}`} />
                </div>
                <div>
                  <label className="block text-xs text-gray-500 mb-1">Event Type</label>
                  {disabled ? (
                    <div className="px-3 py-2 border border-border rounded-md text-sm text-gray-600 bg-gray-50">{LOG_STREAM_EVENT_TYPE_OPTIONS.find((o) => o.value === eventType)?.label}</div>
                  ) : (
                    <Dropdown options={LOG_STREAM_EVENT_TYPE_OPTIONS.map((o) => ({ value: o.value, label: o.label }))} value={eventType} onChange={(v) => setEventType(v as LogStreamEventType)} placeholder="Select event type..." />
                  )}
                </div>
                <div>
                  <label className="block text-xs text-gray-500 mb-1">Destination Type</label>
                  {disabled ? (
                    <div className="px-3 py-2 border border-border rounded-md text-sm text-gray-600 bg-gray-50">{LOG_STREAM_DESTINATION_OPTIONS.find((o) => o.value === destinationType)?.label}</div>
                  ) : (
                    <Dropdown options={LOG_STREAM_DESTINATION_OPTIONS.map((o) => ({ value: o.value, label: o.label }))} value={destinationType} onChange={(v) => setDestinationType(v as LogStreamDestinationType)} placeholder="Select destination..." />
                  )}
                </div>
              </div>
            </div>
          ) : (
            <div>
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-sm font-semibold text-gray-900">Stream Configuration</h3>
                {isEditing && <button type="button" onClick={handleEditToggle} className="px-4 py-1.5 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors">Save</button>}
              </div>
              <div className="border-b border-border mb-6" />
              <div className="space-y-4">
                {destinationType === 'aws_s3' && (
                  <>
                    <div>
                      <label className="block text-xs text-gray-500 mb-1">S3 Bucket Region</label>
                      <input type="text" value={s3Region} onChange={(e) => setS3Region(e.target.value)} disabled={disabled} className={`w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400 ${disabled ? 'bg-gray-50 text-gray-600' : ''}`} />
                    </div>
                    <div>
                      <label className="block text-xs text-gray-500 mb-1">S3 Bucket Name</label>
                      <input type="text" value={s3Bucket} onChange={(e) => setS3Bucket(e.target.value)} disabled={disabled} className={`w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400 ${disabled ? 'bg-gray-50 text-gray-600' : ''}`} />
                    </div>
                    <div>
                      <label className="block text-xs text-gray-500 mb-1">S3 Path Prefix</label>
                      <input type="text" value={s3Prefix} onChange={(e) => setS3Prefix(e.target.value)} disabled={disabled} className={`w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400 ${disabled ? 'bg-gray-50 text-gray-600' : ''}`} />
                    </div>
                    <h4 className="text-sm font-semibold text-orange-600 pt-2">Destination Configuration</h4>
                    <div>
                      <label className="block text-xs text-gray-500 mb-1">Source AWS Account ID</label>
                      <div className="px-3 py-2 border border-border rounded-md text-sm text-gray-500 bg-gray-50">123456789012</div>
                    </div>
                    <div>
                      <label className="block text-xs text-gray-500 mb-1">Destination Bucket Policy (Recommended)</label>
                      <pre className="px-3 py-3 border border-border rounded-md text-[11px] text-gray-700 bg-gray-50 font-mono whitespace-pre-wrap">{`{\n  "Version": "2012-10-17",\n  "Statement": [\n    {\n      "Sid": "DipoleLogStream",\n      "Effect": "Allow",\n      "Principal": {\n        "AWS": "arn:aws:iam::123456789012:root"\n      },\n      "Action": ["s3:PutObject"],\n      "Resource": ["arn:aws:s3:::${s3Bucket || '*'}/*"]\n    }\n  ]\n}`}</pre>
                    </div>
                  </>
                )}

                {destinationType === 'gcs_bucket' && (
                  <>
                    <div>
                      <label className="block text-xs text-gray-500 mb-1">OIDC Issuer URL</label>
                      <div className="px-3 py-2 border border-border rounded-md text-sm text-gray-500 bg-gray-50 font-mono">https://acme.id.ap-south-1.dipolehq.com/</div>
                    </div>
                    <div>
                      <label className="block text-xs text-gray-500 mb-1">Bucket Name</label>
                      <input type="text" value={gcsBucket} onChange={(e) => setGcsBucket(e.target.value)} disabled={disabled} className={`w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400 ${disabled ? 'bg-gray-50 text-gray-600' : ''}`} />
                    </div>
                    <div>
                      <label className="block text-xs text-gray-500 mb-1">Audience</label>
                      <input type="text" value={gcsAudience} onChange={(e) => setGcsAudience(e.target.value)} disabled={disabled} className={`w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400 ${disabled ? 'bg-gray-50 text-gray-600' : ''}`} />
                    </div>
                    <div>
                      <label className="block text-xs text-gray-500 mb-1">Service Account Email</label>
                      <input type="text" value={gcsEmail} onChange={(e) => setGcsEmail(e.target.value)} disabled={disabled} className={`w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400 ${disabled ? 'bg-gray-50 text-gray-600' : ''}`} />
                    </div>
                    <div>
                      <label className="block text-xs text-gray-500 mb-1">Token Lifetime</label>
                      <div className="flex items-center gap-2">
                        <input type="number" value={gcsLifetime} onChange={(e) => setGcsLifetime(e.target.value)} disabled={disabled} className={`w-28 px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400 ${disabled ? 'bg-gray-50 text-gray-600' : ''}`} />
                        <span className="text-sm text-gray-600">Seconds</span>
                      </div>
                    </div>
                  </>
                )}

                {destinationType === 'crowdstrike_siem' && (
                  <>
                    <div>
                      <label className="block text-xs text-gray-500 mb-1">Host/Port</label>
                      <div className="relative">
                        <input type="text" value={hostPort} onChange={(e) => setHostPort(e.target.value)} disabled={disabled} className={`w-full px-3 py-2 pr-20 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400 ${disabled ? 'bg-gray-50 text-gray-600' : ''}`} />
                        <label className="absolute right-3 top-1/2 -translate-y-1/2 flex items-center gap-1.5">
                          <input type="checkbox" checked={tls} onChange={(e) => setTls(e.target.checked)} disabled={disabled} className="w-4 h-4 accent-orange-500 cursor-pointer disabled:opacity-60" />
                          <span className="text-sm text-gray-700">TLS</span>
                        </label>
                      </div>
                    </div>
                    <div>
                      <label className="block text-xs text-gray-500 mb-1">TLS Verification</label>
                      {disabled ? (
                        <div className="px-3 py-2 border border-border rounded-md text-sm text-gray-600 bg-gray-50 capitalize">{tlsVerification}</div>
                      ) : (
                        <Dropdown options={[{ value: 'full', label: 'Full' }, { value: 'none', label: 'None' }]} value={tlsVerification} onChange={setTlsVerification} placeholder="TLS Verification" />
                      )}
                    </div>
                    <div>
                      <label className="block text-xs text-gray-500 mb-1">API Key</label>
                      <input type="password" value={apiKey} onChange={(e) => setApiKey(e.target.value)} disabled={disabled} className={`w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400 ${disabled ? 'bg-gray-50 text-gray-600' : ''}`} />
                    </div>
                    <div>
                      <label className="block text-xs text-gray-500 mb-1">Source Name</label>
                      <input type="text" value={sourceName} onChange={(e) => setSourceName(e.target.value)} disabled={disabled} className={`w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400 ${disabled ? 'bg-gray-50 text-gray-600' : ''}`} />
                    </div>
                  </>
                )}

                {destinationType === 'splunk_siem' && (
                  <>
                    <div>
                      <label className="block text-xs text-gray-500 mb-1">Host/Port</label>
                      <div className="relative">
                        <input type="text" value={hostPort} onChange={(e) => setHostPort(e.target.value)} disabled={disabled} className={`w-full px-3 py-2 pr-20 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400 ${disabled ? 'bg-gray-50 text-gray-600' : ''}`} />
                        <label className="absolute right-3 top-1/2 -translate-y-1/2 flex items-center gap-1.5">
                          <input type="checkbox" checked={tls} onChange={(e) => setTls(e.target.checked)} disabled={disabled} className="w-4 h-4 accent-orange-500 cursor-pointer disabled:opacity-60" />
                          <span className="text-sm text-gray-700">TLS</span>
                        </label>
                      </div>
                    </div>
                    <div>
                      <label className="block text-xs text-gray-500 mb-1">TLS Verification</label>
                      {disabled ? (
                        <div className="px-3 py-2 border border-border rounded-md text-sm text-gray-600 bg-gray-50 capitalize">{tlsVerification}</div>
                      ) : (
                        <Dropdown options={[{ value: 'full', label: 'Full' }, { value: 'none', label: 'None' }]} value={tlsVerification} onChange={setTlsVerification} placeholder="TLS Verification" />
                      )}
                    </div>
                    <div>
                      <label className="block text-xs text-gray-500 mb-1">Authentication Token</label>
                      <input type="password" value={authToken} onChange={(e) => setAuthToken(e.target.value)} disabled={disabled} className={`w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400 ${disabled ? 'bg-gray-50 text-gray-600' : ''}`} />
                    </div>
                    <div>
                      <label className="block text-xs text-gray-500 mb-1">Source Name</label>
                      <input type="text" value={sourceName} onChange={(e) => setSourceName(e.target.value)} disabled={disabled} className={`w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400 ${disabled ? 'bg-gray-50 text-gray-600' : ''}`} />
                    </div>
                  </>
                )}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

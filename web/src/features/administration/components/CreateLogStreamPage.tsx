import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { PageHeader } from '@/shared/components/PageHeader'
import { Dropdown } from '@/shared/components/Dropdown'
import { LOG_STREAM_EVENT_TYPE_OPTIONS, LOG_STREAM_DESTINATION_OPTIONS } from '@/data/administration/logStreams'
import type { LogStreamEventType, LogStreamDestinationType } from '@/data/administration/logStreams'

type TabKey = 'details' | 'configuration'

export function CreateLogStreamPage() {
  const navigate = useNavigate()
  const [activeTab, setActiveTab] = useState<TabKey>('details')

  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [eventType, setEventType] = useState<LogStreamEventType>('access_authorization')
  const [destinationType, setDestinationType] = useState<LogStreamDestinationType>('aws_s3')

  // AWS S3
  const [s3Region, setS3Region] = useState('')
  const [s3Bucket, setS3Bucket] = useState('')
  const [s3Prefix, setS3Prefix] = useState('')

  // GCS
  const [gcsBucket, setGcsBucket] = useState('')
  const [gcsPathPrefix, setGcsPathPrefix] = useState('')
  const [gcsAudience, setGcsAudience] = useState('')
  const [gcsEmail, setGcsEmail] = useState('')
  const [gcsLifetime, setGcsLifetime] = useState('3600')

  // CrowdStrike / Splunk
  const [hostPort, setHostPort] = useState('')
  const [tls, setTls] = useState(false)
  const [tlsVerification, setTlsVerification] = useState('full')
  const [apiKey, setApiKey] = useState('')
  const [authToken, setAuthToken] = useState('')
  const [sourceName, setSourceName] = useState('')

  function handleSave() {
    console.info('[create-log-stream]', { name, description, eventType, destinationType })
    navigate('/administration')
  }

  function handleCancel() {
    navigate('/administration')
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
            <button type="button" onClick={handleSave} className="w-full px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors">Save Log Stream</button>
            <button type="button" onClick={handleCancel} className="w-full px-4 py-2 border border-gray-300 text-xs font-medium text-gray-600 hover:text-gray-900 hover:border-gray-400 rounded-md transition-colors">Cancel</button>
          </div>
        </nav>

        <div className="flex-1 overflow-y-auto p-6">
          {activeTab === 'details' ? (
            <div>
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-sm font-semibold text-gray-900">Stream Details</h3>
                <button type="button" onClick={handleSave} className="px-4 py-1.5 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors">Save</button>
              </div>
              <div className="border-b border-border mb-6" />
              <div className="space-y-4">
                <div>
                  <label className="block text-xs text-gray-500 mb-1">Name <span className="text-red-500">*</span></label>
                  <input type="text" value={name} onChange={(e) => setName(e.target.value)} placeholder="Log Stream name" className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400" />
                </div>
                <div>
                  <label className="block text-xs text-gray-500 mb-1">Description (Optional)</label>
                  <textarea value={description} onChange={(e) => setDescription(e.target.value)} placeholder="Description (Optional)" rows={2} className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400 resize-y" />
                </div>
                <div>
                  <label className="block text-xs text-gray-500 mb-1">Event Type <span className="text-red-500">*</span></label>
                  <Dropdown options={LOG_STREAM_EVENT_TYPE_OPTIONS.map((o) => ({ value: o.value, label: o.label }))} value={eventType} onChange={(v) => setEventType(v as LogStreamEventType)} placeholder="Select event type..." />
                </div>
                <div>
                  <label className="block text-xs text-gray-500 mb-1">Destination Type <span className="text-red-500">*</span></label>
                  <Dropdown options={LOG_STREAM_DESTINATION_OPTIONS.map((o) => ({ value: o.value, label: o.label }))} value={destinationType} onChange={(v) => setDestinationType(v as LogStreamDestinationType)} placeholder="Select destination..." />
                </div>
              </div>
            </div>
          ) : (
            <div>
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-sm font-semibold text-gray-900">Stream Configuration</h3>
                <button type="button" onClick={handleSave} className="px-4 py-1.5 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors">Save</button>
              </div>
              <div className="border-b border-border mb-6" />
              <div className="space-y-4">
                {destinationType === 'aws_s3' && (
                  <>
                    <div>
                      <label className="block text-xs text-gray-500 mb-1">S3 Bucket Region</label>
                      <input type="text" value={s3Region} onChange={(e) => setS3Region(e.target.value)} placeholder="S3 Bucket Region" className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400" />
                    </div>
                    <div>
                      <label className="block text-xs text-gray-500 mb-1">S3 Bucket Name</label>
                      <input type="text" value={s3Bucket} onChange={(e) => setS3Bucket(e.target.value)} placeholder="S3 Bucket Name" className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400" />
                    </div>
                    <div>
                      <label className="block text-xs text-gray-500 mb-1">S3 Path Prefix</label>
                      <input type="text" value={s3Prefix} onChange={(e) => setS3Prefix(e.target.value)} placeholder="S3 Path Prefix" className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400" />
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
                      <input type="text" value={gcsBucket} onChange={(e) => setGcsBucket(e.target.value)} placeholder="Bucket Name" className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400" />
                    </div>
                    <div>
                      <label className="block text-xs text-gray-500 mb-1">Path Prefix</label>
                      <input type="text" value={gcsPathPrefix} onChange={(e) => setGcsPathPrefix(e.target.value)} placeholder="Path Prefix" className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400" />
                    </div>
                    <div>
                      <label className="block text-xs text-gray-500 mb-1">Audience</label>
                      <input type="text" value={gcsAudience} onChange={(e) => setGcsAudience(e.target.value)} placeholder="Audience" className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400" />
                    </div>
                    <div>
                      <label className="block text-xs text-gray-500 mb-1">Service Account Email</label>
                      <input type="text" value={gcsEmail} onChange={(e) => setGcsEmail(e.target.value)} placeholder="Service Account Email" className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400" />
                    </div>
                    <div>
                      <label className="block text-xs text-gray-500 mb-1">Token Lifetime</label>
                      <div className="flex items-center gap-2">
                        <input type="number" value={gcsLifetime} onChange={(e) => setGcsLifetime(e.target.value)} placeholder="3600" className="w-28 px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400" />
                        <span className="text-sm text-gray-600">Seconds</span>
                      </div>
                    </div>
                  </>
                )}

                {destinationType === 'crowdstrike_siem' && (
                  <>
                    <div>
                      <label className="block text-xs text-gray-500 mb-1">CrowdStrike Host/Port</label>
                      <div className="relative">
                        <input type="text" value={hostPort} onChange={(e) => setHostPort(e.target.value)} placeholder="host.ingest.crowdstrike.com:443" className="w-full px-3 py-2 pr-20 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400" />
                        <label className="absolute right-3 top-1/2 -translate-y-1/2 flex items-center gap-1.5 cursor-pointer">
                          <input type="checkbox" checked={tls} onChange={(e) => setTls(e.target.checked)} className="w-4 h-4 accent-orange-500 cursor-pointer" />
                          <span className="text-sm text-gray-700">TLS</span>
                        </label>
                      </div>
                    </div>
                    <div>
                      <label className="block text-xs text-gray-500 mb-1">TLS Verification</label>
                      <Dropdown options={[{ value: 'full', label: 'Full' }, { value: 'none', label: 'None' }]} value={tlsVerification} onChange={setTlsVerification} placeholder="TLS Verification" />
                    </div>
                    <div>
                      <label className="block text-xs text-gray-500 mb-1">API Key</label>
                      <input type="password" value={apiKey} onChange={(e) => setApiKey(e.target.value)} placeholder="API Key" className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400" />
                    </div>
                    <div>
                      <label className="block text-xs text-gray-500 mb-1">Source Name</label>
                      <input type="text" value={sourceName} onChange={(e) => setSourceName(e.target.value)} placeholder="Connector name" className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400" />
                    </div>
                  </>
                )}

                {destinationType === 'splunk_siem' && (
                  <>
                    <div>
                      <label className="block text-xs text-gray-500 mb-1">Splunk Host/Port</label>
                      <div className="relative">
                        <input type="text" value={hostPort} onChange={(e) => setHostPort(e.target.value)} placeholder="splunk.internal.com:8088" className="w-full px-3 py-2 pr-20 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400" />
                        <label className="absolute right-3 top-1/2 -translate-y-1/2 flex items-center gap-1.5 cursor-pointer">
                          <input type="checkbox" checked={tls} onChange={(e) => setTls(e.target.checked)} className="w-4 h-4 accent-orange-500 cursor-pointer" />
                          <span className="text-sm text-gray-700">TLS</span>
                        </label>
                      </div>
                    </div>
                    <div>
                      <label className="block text-xs text-gray-500 mb-1">TLS Verification</label>
                      <Dropdown options={[{ value: 'full', label: 'Full' }, { value: 'none', label: 'None' }]} value={tlsVerification} onChange={setTlsVerification} placeholder="TLS Verification" />
                    </div>
                    <div>
                      <label className="block text-xs text-gray-500 mb-1">Authentication Token</label>
                      <input type="password" value={authToken} onChange={(e) => setAuthToken(e.target.value)} placeholder="HEC Token" className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400" />
                    </div>
                    <div>
                      <label className="block text-xs text-gray-500 mb-1">Source Name</label>
                      <input type="text" value={sourceName} onChange={(e) => setSourceName(e.target.value)} placeholder="Source name" className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400" />
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

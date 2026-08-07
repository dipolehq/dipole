import { useState } from 'react'
import { identityIssuerUrl } from '@/shared/lib/platform-urls'
import { useNavigate } from 'react-router-dom'
import { PageHeader } from '@/shared/components/PageHeader'
import { FloatingInput, FloatingTextarea, FloatingDropdown } from '@/shared/components/FloatingInput'
import { LOG_STREAM_EVENT_TYPE_OPTIONS, LOG_STREAM_DESTINATION_OPTIONS } from '@/shared/constants/administration'
import { useCreateLogStream } from '@/features/administration/hooks/useLogStreams'
import { apiErrorMessage } from '@/features/administration/api/logStreamsApi'
import type { LogStreamEventType, LogStreamDestinationType } from '@/shared/constants/administration'

export function CreateLogStreamPage() {
  const navigate = useNavigate()
  const createMutation = useCreateLogStream()
  const [error, setError] = useState<string | null>(null)

  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [eventType, setEventType] = useState<LogStreamEventType | ''>('')
  const [destinationType, setDestinationType] = useState<LogStreamDestinationType | ''>('')

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
  const [tlsVerification, setTlsVerification] = useState('')
  const [apiKey, setApiKey] = useState('')
  const [authToken, setAuthToken] = useState('')
  const [sourceName, setSourceName] = useState('')

  async function handleSave() {
    setError(null)
    const config: Record<string, unknown> = {}
    if (destinationType === 'aws_s3') {
      config.s3BucketRegion = s3Region
      config.s3BucketName = s3Bucket
      config.s3PathPrefix = s3Prefix
    } else if (destinationType === 'gcs_bucket') {
      config.gcsBucketName = gcsBucket
      config.gcsPathPrefix = gcsPathPrefix
      config.gcsAudience = gcsAudience
      config.gcsServiceAccountEmail = gcsEmail
      config.gcsTokenLifetime = gcsLifetime
    } else if (destinationType === 'crowdstrike_siem') {
      config.hostPort = hostPort
      config.tls = tls
      config.tlsVerification = tlsVerification
      config.apiKey = apiKey
      config.sourceName = sourceName
    } else if (destinationType === 'splunk_siem') {
      config.hostPort = hostPort
      config.tls = tls
      config.tlsVerification = tlsVerification
      config.authToken = authToken
      config.sourceName = sourceName
    }
    try {
      await createMutation.mutateAsync({
        name,
        description,
        isActive: true,
        eventType,
        destinationType,
        config,
      })
      navigate('/administration?tab=log_streams')
    } catch (err) {
      setError(apiErrorMessage(err, 'Failed to create log stream'))
    }
  }

  function handleCancel() {
    navigate('/administration?tab=log_streams')
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
          <button type="button" onClick={handleSave} disabled={createMutation.isPending} className="px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white text-sm font-medium rounded-md transition-colors disabled:opacity-60">
            {createMutation.isPending ? 'Saving…' : 'Save'}
          </button>
          <button type="button" onClick={handleCancel} className="px-4 py-2 border border-gray-300 text-sm font-medium text-gray-600 hover:text-gray-900 hover:border-gray-400 rounded-md transition-colors">Cancel</button>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto bg-white">
        <div className="flex h-full">
          {/* Form — left 60% */}
          <div className="w-[60%] px-6 py-6 overflow-y-auto">
            <div className="space-y-5">
              <div className="border border-border rounded-lg p-5">
                <h4 className="text-sm font-semibold text-gray-800 mb-4">Stream Details</h4>
                <div className="space-y-4">
                  <FloatingInput label="Name" value={name} onChange={setName} required />
                  <FloatingTextarea label="Description (Optional)" value={description} onChange={setDescription} rows={2} />
                  <FloatingDropdown label="Event Type" options={LOG_STREAM_EVENT_TYPE_OPTIONS.map((o) => ({ value: o.value, label: o.label }))} value={eventType} onChange={(v) => setEventType(v as LogStreamEventType)} required />
                  <FloatingDropdown label="Destination Type" options={LOG_STREAM_DESTINATION_OPTIONS.map((o) => ({ value: o.value, label: o.label }))} value={destinationType} onChange={(v) => setDestinationType(v as LogStreamDestinationType)} required />
                </div>
              </div>

              {destinationType === 'aws_s3' && (
                <div className="border border-border rounded-lg p-5">
                  <h4 className="text-sm font-semibold text-gray-800 mb-4">AWS S3 Configuration</h4>
                  <div className="space-y-4">
                    <FloatingInput label="S3 Bucket Region" value={s3Region} onChange={setS3Region} />
                    <FloatingInput label="S3 Bucket Name" value={s3Bucket} onChange={setS3Bucket} />
                    <FloatingInput label="S3 Path Prefix" value={s3Prefix} onChange={setS3Prefix} />
                    <div className="bg-gray-50 rounded-md p-4">
                      <label className="block text-[10px] text-gray-500 mb-0.5">Source AWS Account ID</label>
                      <div className="text-xs text-gray-700 font-mono">123456789012</div>
                    </div>
                    <div>
                      <label className="block text-xs text-gray-500 mb-1">Destination Bucket Policy (Recommended)</label>
                      <pre className="px-3 py-3 border border-border rounded-md text-[11px] text-gray-700 bg-gray-50 font-mono whitespace-pre-wrap">{`{\n  "Version": "2012-10-17",\n  "Statement": [\n    {\n      "Sid": "DipoleLogStream",\n      "Effect": "Allow",\n      "Principal": {\n        "AWS": "arn:aws:iam::123456789012:root"\n      },\n      "Action": ["s3:PutObject"],\n      "Resource": ["arn:aws:s3:::${s3Bucket || '*'}/*"]\n    }\n  ]\n}`}</pre>
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
                      <div className="text-xs text-gray-700 font-mono">{identityIssuerUrl('<tenant>')}</div>
                    </div>
                    <FloatingInput label="Bucket Name" value={gcsBucket} onChange={setGcsBucket} />
                    <FloatingInput label="Path Prefix" value={gcsPathPrefix} onChange={setGcsPathPrefix} />
                    <FloatingInput label="Audience" value={gcsAudience} onChange={setGcsAudience} />
                    <FloatingInput label="Service Account Email" value={gcsEmail} onChange={setGcsEmail} />
                    <FloatingInput label="Token Lifetime (seconds)" value={gcsLifetime} onChange={setGcsLifetime} />
                  </div>
                </div>
              )}

              {destinationType === 'crowdstrike_siem' && (
                <div className="border border-border rounded-lg p-5">
                  <h4 className="text-sm font-semibold text-gray-800 mb-4">CrowdStrike SIEM Configuration</h4>
                  <div className="space-y-4">
                    <div className="relative">
                      <FloatingInput label="Host/Port" value={hostPort} onChange={setHostPort} />
                      <label className="absolute right-3 top-1/2 -translate-y-1/2 flex items-center gap-1.5 cursor-pointer">
                        <input type="checkbox" checked={tls} onChange={(e) => setTls(e.target.checked)} className="w-4 h-4 accent-orange-500 cursor-pointer" />
                        <span className="text-sm text-gray-700">TLS</span>
                      </label>
                    </div>
                    <FloatingDropdown label="TLS Verification" options={[{ value: 'full', label: 'Full' }, { value: 'none', label: 'None' }]} value={tlsVerification} onChange={setTlsVerification} />
                    <FloatingInput label="API Key" value={apiKey} onChange={setApiKey} type="password" />
                    <FloatingInput label="Source Name" value={sourceName} onChange={setSourceName} />
                  </div>
                </div>
              )}

              {destinationType === 'splunk_siem' && (
                <div className="border border-border rounded-lg p-5">
                  <h4 className="text-sm font-semibold text-gray-800 mb-4">Splunk SIEM Configuration</h4>
                  <div className="space-y-4">
                    <div className="relative">
                      <FloatingInput label="Host/Port" value={hostPort} onChange={setHostPort} />
                      <label className="absolute right-3 top-1/2 -translate-y-1/2 flex items-center gap-1.5 cursor-pointer">
                        <input type="checkbox" checked={tls} onChange={(e) => setTls(e.target.checked)} className="w-4 h-4 accent-orange-500 cursor-pointer" />
                        <span className="text-sm text-gray-700">TLS</span>
                      </label>
                    </div>
                    <FloatingDropdown label="TLS Verification" options={[{ value: 'full', label: 'Full' }, { value: 'none', label: 'None' }]} value={tlsVerification} onChange={setTlsVerification} />
                    <FloatingInput label="Authentication Token" value={authToken} onChange={setAuthToken} type="password" />
                    <FloatingInput label="Source Name" value={sourceName} onChange={setSourceName} />
                  </div>
                </div>
              )}
            </div>
          </div>

          {/* Help panel — right 40% */}
          <div className="w-[40%] px-6 py-6 overflow-y-auto">
            <div className="border border-border rounded-lg p-5 sticky top-0">
              <p className="text-xs text-gray-700 leading-relaxed mb-3">
                <span className="font-semibold text-orange-600">Log Streams</span> export access authorization events, audit logs, or workload events to external SIEM systems for centralized monitoring and compliance.
              </p>
              <p className="text-xs text-gray-700 leading-relaxed mb-3">
                Choose a <span className="font-semibold">Destination Type</span> based on where you want events delivered — AWS S3, GCS, CrowdStrike, or Splunk.
              </p>
              <p className="text-xs text-gray-700 leading-relaxed">
                For AWS S3 destinations, you&apos;ll need to configure a bucket policy allowing Dipole to write objects. The recommended policy is shown in the configuration section.
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

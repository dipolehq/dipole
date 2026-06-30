import { useState } from 'react'
import { Dropdown } from '@/shared/components/Dropdown'

interface TrustProviderConfigProps {
  type: string
}

const MATCH_RULES: Record<string, { value: string; label: string }[]> = {
  aws_metadata: [
    { value: 'accountId', label: 'accountId' },
    { value: 'architecture', label: 'architecture' },
    { value: 'availabilityZone', label: 'availabilityZone' },
    { value: 'billingProducts', label: 'billingProducts' },
    { value: 'imageId', label: 'imageId' },
    { value: 'instanceId', label: 'instanceId' },
    { value: 'instanceType', label: 'instanceType' },
    { value: 'kernelId', label: 'kernelId' },
    { value: 'marketplaceProductCodes', label: 'marketplaceProductCodes' },
    { value: 'pendingTime', label: 'pendingTime' },
    { value: 'privateIP', label: 'privateIP' },
    { value: 'region', label: 'region' },
    { value: 'version', label: 'version' },
  ],
  aws_role: [
    { value: 'accountId', label: 'accountId' },
    { value: 'assumedRole', label: 'assumedRole' },
    { value: 'roleArn', label: 'roleArn' },
    { value: 'username', label: 'username' },
  ],
  azure_metadata: [
    { value: 'sku', label: 'sku' },
    { value: 'subscriptionId', label: 'subscriptionId' },
    { value: 'vmId', label: 'vmId' },
  ],
  gcp_identity_token: [
    { value: 'email', label: 'email' },
  ],
  kubernetes_service_account: [
    { value: 'iss', label: 'iss' },
    { value: 'kubernetes.io.namespace', label: 'kubernetes.io { namespace }' },
    { value: 'kubernetes.io.pod.name', label: 'kubernetes.io { pod { name } }' },
    { value: 'kubernetes.io.serviceaccount.name', label: 'kubernetes.io { serviceaccount { name } }' },
    { value: 'sub', label: 'sub' },
  ],
  github_action: [
    { value: 'actor', label: 'actor' },
    { value: 'repository', label: 'repository' },
    { value: 'workflow', label: 'workflow' },
  ],
  gitlab_job: [
    { value: 'namespace_path', label: 'namespace_path' },
    { value: 'project_path', label: 'project_path' },
    { value: 'ref_path', label: 'ref_path' },
    { value: 'subject', label: 'subject' },
  ],
  terraform_cloud: [
    { value: 'terraform_organization_id', label: 'terraform_organization_id' },
    { value: 'terraform_project_id', label: 'terraform_project_id' },
    { value: 'terraform_workspace_id', label: 'terraform_workspace_id' },
  ],
  kerberos: [
    { value: 'principal', label: 'Principal' },
    { value: 'realm', label: 'Realm/Domain' },
    { value: 'source_ip', label: 'Source IP' },
  ],
  oidc_id_token: [
    { value: 'aud', label: 'Audience (aud)' },
    { value: 'iss', label: 'Issuer (iss)' },
    { value: 'sub', label: 'Subject (sub)' },
    { value: 'custom_claim', label: 'Custom Claim' },
  ],
  samlv2_response: [
    { value: 'aud', label: 'Audience (aud)' },
    { value: 'iss', label: 'Issuer (iss)' },
    { value: 'sub', label: 'Subject (sub)' },
  ],
  certificate_signed: [
    { value: 'custom', label: 'Custom Attribute' },
  ],
}

const ATTESTATION_METHODS = [
  { value: 'oidc_discovery', label: 'OIDC Discovery' },
  { value: 'symmetric_key', label: 'Symmetric Key' },
  { value: 'upload_jwks', label: 'Upload JWKS' },
  { value: 'upload_public_key', label: 'Upload Public Key' },
  { value: 'tenant_oidc_id_token', label: 'Dipole Tenant OIDC ID Token' },
]

export function TrustProviderConfig({ type }: TrustProviderConfigProps) {
  const [matchRules, setMatchRules] = useState<{ attribute: string; value: string }[]>([
    { attribute: '', value: '' },
  ])
  const [certificate, setCertificate] = useState('')
  const [thumbprint, setThumbprint] = useState('')
  const [subject, setSubject] = useState('')
  const [expiresAt, setExpiresAt] = useState('')
  const [oidcEndpoint, setOidcEndpoint] = useState('')
  const [attestationMethod, setAttestationMethod] = useState('oidc_discovery')

  const rules = MATCH_RULES[type] || []

  function addMatchRule() {
    setMatchRules([...matchRules, { attribute: '', value: '' }])
  }

  function updateMatchRule(idx: number, field: 'attribute' | 'value', val: string) {
    setMatchRules(matchRules.map((r, i) => i === idx ? { ...r, [field]: val } : r))
  }

  function removeMatchRule(idx: number) {
    if (matchRules.length > 1) {
      setMatchRules(matchRules.filter((_, i) => i !== idx))
    }
  }

  if (!type) return null

  const showCertificate = type === 'aws_metadata'
  const showCertificateTable = type === 'certificate_signed'
  const showOidcEndpoint = ['kubernetes_service_account', 'gitlab_job'].includes(type)
  const showAttestationMethod = type === 'oidc_id_token'
  const showMatchRules = !['certificate_signed'].includes(type)
  const showEdgeSdkAudience = ['gcp_identity_token', 'github_action', 'gitlab_job', 'terraform_cloud', 'kubernetes_service_account'].includes(type)
  const showSamlConfig = type === 'samlv2_response'

  return (
    <div className="space-y-5">
      {/* Edge SDK Audience */}
      {showEdgeSdkAudience && (
        <div>
          <label className="block text-[11px] text-gray-500 mb-1">Edge SDK Audience</label>
          <div className="flex items-center border border-border rounded-md overflow-hidden bg-gray-50">
            <input type="text" readOnly value="https://<tenant>.id.ap-south-1.dipolehq.com" className="flex-1 px-3 py-2 text-sm text-gray-500 bg-gray-50 focus:outline-none" />
            <button
              onClick={() => navigator.clipboard.writeText('https://<tenant>.id.ap-south-1.dipolehq.com')}
              className="px-3 py-2 border-l border-border text-gray-400 hover:text-gray-600"
              title="Copy to clipboard"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><rect x="9" y="9" width="13" height="13" rx="2" ry="2" strokeWidth="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" strokeWidth="2"/></svg>
            </button>
          </div>
        </div>
      )}
      {/* Match Rules */}
      {showMatchRules && (
        <div className="border border-border rounded-md p-4">
          <h4 className="text-xs font-semibold text-gray-800 mb-3">Match Rules</h4>
          <div className="space-y-2">
            {matchRules.map((rule, idx) => (
              <div key={idx} className="flex items-center gap-2">
                <Dropdown
                  options={rules}
                  value={rule.attribute}
                  onChange={(v) => updateMatchRule(idx, 'attribute', v)}
                  placeholder="Attribute"
                  className="w-48"
                />
                <input
                  type="text"
                  placeholder="Value"
                  value={rule.value}
                  onChange={(e) => updateMatchRule(idx, 'value', e.target.value)}
                  className="flex-1 px-3 py-2 border border-border rounded-md text-xs focus:outline-none focus:ring-1 focus:ring-gray-300"
                />
                <button onClick={() => removeMatchRule(idx)} className="text-orange-400 hover:text-red-500 text-sm">×</button>
              </div>
            ))}
          </div>
          <button onClick={addMatchRule} className="mt-2 text-xs text-orange-500 hover:text-orange-600 font-medium">
            + New
          </button>
        </div>
      )}

      {/* Certificate Table (Certificate Signed Attestation) */}
      {showCertificateTable && (
        <CertificateSignedSection />
      )}

      {/* SAML Configuration */}
      {showSamlConfig && (
        <div className="space-y-4">
          <div>
            <div className="flex items-center border border-border rounded-md overflow-hidden">
              <span className="px-3 py-2 text-sm text-gray-500 bg-gray-50 border-r border-border">https://</span>
              <input
                type="text"
                placeholder="Metadata URL"
                className="flex-1 px-3 py-2 text-sm focus:outline-none"
              />
            </div>
          </div>
          <div>
            <label className="block text-xs font-semibold text-gray-700 mb-2">Metadata XML</label>
            <div className="border border-border rounded-md overflow-hidden">
              <div className="flex items-center gap-2 px-3 py-1.5 bg-gray-50 border-b border-border">
                <label className="cursor-pointer text-orange-500 hover:text-orange-600" title="Upload XML">
                  <input type="file" accept=".xml" className="hidden" />
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" /></svg>
                </label>
                <button className="text-orange-500 hover:text-red-500" title="Clear">
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" /></svg>
                </button>
              </div>
              <textarea
                placeholder=""
                rows={6}
                className="w-full px-3 py-2 text-sm resize-y focus:outline-none"
              />
            </div>
          </div>
        </div>
      )}

      {/* Certificate (AWS Metadata, Certificate Signed) */}
      {showCertificate && (
        <>
          <div>
            <label className="block text-xs font-medium text-gray-700 mb-1.5">
              Certificate
            </label>
            <div className="flex gap-2">
              <textarea
                placeholder="Certificate"
                value={certificate}
                onChange={(e) => setCertificate(e.target.value)}
                rows={5}
                className="flex-1 px-3 py-2 border border-border rounded-md text-sm resize-y focus:outline-none focus:ring-1 focus:ring-gray-300"
              />
              <div className="flex flex-col gap-2">
                <label className="cursor-pointer text-orange-500 hover:text-orange-600" title="Upload certificate">
                  <input type="file" accept=".pem,.cer,.crt" className="hidden" onChange={(e) => {
                    const file = e.target.files?.[0]
                    if (file) {
                      const reader = new FileReader()
                      reader.onload = (ev) => setCertificate(ev.target?.result as string || '')
                      reader.readAsText(file)
                    }
                  }} />
                  <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" /></svg>
                </label>
                <button onClick={() => setCertificate('')} className="text-orange-500 hover:text-red-500" title="Clear certificate">
                  <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" /></svg>
                </button>
              </div>
            </div>
          </div>
          <div>
            <label className="block text-xs font-medium text-gray-700 mb-1.5">Thumbprint</label>
            <input
              type="text"
              placeholder="Thumbprint"
              value={thumbprint}
              onChange={(e) => setThumbprint(e.target.value)}
              className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-gray-300"
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-gray-700 mb-1.5">Subject</label>
            <input
              type="text"
              placeholder="Subject"
              value={subject}
              onChange={(e) => setSubject(e.target.value)}
              className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-gray-300"
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-gray-700 mb-1.5">Expires At</label>
            <input
              type="text"
              placeholder="Expires At"
              value={expiresAt}
              onChange={(e) => setExpiresAt(e.target.value)}
              className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-gray-300"
            />
          </div>
        </>
      )}

      {/* OIDC Endpoint (Kubernetes, GitLab) */}
      {showOidcEndpoint && (
        <div>
          <label className="block text-xs font-medium text-gray-700 mb-1.5">OIDC Endpoint</label>
          <input
            type="text"
            placeholder={type === 'gitlab_job' ? 'https://gitlab.com' : 'https://oidc.eks.us-east-2.amazonaws.com/id/...'}
            value={oidcEndpoint}
            onChange={(e) => setOidcEndpoint(e.target.value)}
            className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-gray-300"
          />
        </div>
      )}

      {/* Attestation Method (OIDC ID Token) */}
      {showAttestationMethod && (
        <div>
          <label className="block text-xs font-medium text-gray-700 mb-1.5">Attestation Method</label>
          <Dropdown
            options={ATTESTATION_METHODS}
            value={attestationMethod}
            onChange={setAttestationMethod}
            placeholder="Select method..."
          />
          {attestationMethod === 'oidc_discovery' && (
            <div className="mt-3">
              <label className="block text-xs font-medium text-gray-700 mb-1.5">OIDC Endpoint</label>
              <input
                type="text"
                placeholder="https://identity-provider.example.com"
                value={oidcEndpoint}
                onChange={(e) => setOidcEndpoint(e.target.value)}
                className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-gray-300"
              />
            </div>
          )}
          {attestationMethod === 'symmetric_key' && (
            <div className="mt-3">
              <label className="block text-xs font-medium text-gray-700 mb-1.5">Symmetric Key (Base64)</label>
              <input
                type="text"
                placeholder="Base64 encoded key"
                className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-gray-300"
              />
            </div>
          )}
          {attestationMethod === 'upload_jwks' && (
            <div className="mt-3">
              <label className="block text-xs font-medium text-gray-700 mb-1.5">JWKS</label>
              <textarea
                placeholder='{"keys": [...]}'
                rows={4}
                className="w-full px-3 py-2 border border-border rounded-md text-sm resize-y focus:outline-none focus:ring-1 focus:ring-gray-300"
              />
            </div>
          )}
          {attestationMethod === 'upload_public_key' && (
            <div className="mt-3 space-y-3">
              <div>
                <label className="block text-xs font-medium text-gray-700 mb-1.5">Public Key (PEM)</label>
                <textarea
                  placeholder="-----BEGIN PUBLIC KEY-----"
                  rows={4}
                  className="w-full px-3 py-2 border border-border rounded-md text-sm resize-y focus:outline-none focus:ring-1 focus:ring-gray-300"
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-700 mb-1.5">Thumbprint</label>
                <input
                  type="text"
                  placeholder="Thumbprint"
                  className="w-full px-3 py-2 border border-border rounded-md text-sm bg-gray-50 focus:outline-none"
                  readOnly
                />
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  )
}

function CertificateSignedSection() {
  const [certs, setCerts] = useState<{ id: number; certificate: string; thumbprint: string; subject: string; expiresAt: string; expanded: boolean }[]>([])
  const [nextId, setNextId] = useState(1)

  const [menuOpenId, setMenuOpenId] = useState<number | null>(null)

  function addCert() {
    setCerts([...certs, { id: nextId, certificate: '', thumbprint: '', subject: '', expiresAt: '', expanded: true }])
    setNextId(nextId + 1)
  }

  function updateCert(id: number, field: string, val: string) {
    setCerts(certs.map((c) => c.id === id ? { ...c, [field]: val } : c))
  }

  function toggleExpand(id: number) {
    setCerts(certs.map((c) => c.id === id ? { ...c, expanded: !c.expanded } : c))
  }

  function removeCert(id: number) {
    setCerts(certs.filter((c) => c.id !== id))
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-3">
        <input
          type="text"
          placeholder="Search"
          className="px-3 py-1.5 border border-border rounded-md text-xs w-56 focus:outline-none focus:ring-1 focus:ring-gray-300"
        />
        <button onClick={addCert} className="flex items-center gap-1 px-3 py-1.5 border border-orange-500 text-orange-500 hover:bg-orange-50 text-xs font-medium rounded-md transition-colors">
          + Add
        </button>
      </div>
      <div className="border border-border rounded-md overflow-hidden">
        <table className="w-full text-xs">
          <thead className="bg-gray-100">
            <tr>
              <th className="px-3 py-2 text-left font-semibold text-gray-600">Name</th>
              <th className="px-3 py-2 text-center font-semibold text-gray-600">Subject</th>
              <th className="px-3 py-2 text-right font-semibold text-gray-600">Expires At ↑</th>
            </tr>
          </thead>
          <tbody>
            {certs.length === 0 && (
              <tr>
                <td colSpan={3} className="px-3 py-4 text-center text-gray-400 text-xs">No certificates found.</td>
              </tr>
            )}
            {certs.map((cert) => (
              <tr key={cert.id}>
                <td colSpan={3} className="p-0">
                  <div className="border border-border rounded-md mx-2 my-2 overflow-hidden">
                    {/* Row header */}
                    <div className="flex items-center justify-between px-4 py-2.5 bg-white cursor-pointer" onClick={() => toggleExpand(cert.id)}>
                      <div className="flex items-center gap-6 flex-1">
                        <span className="text-gray-500 text-xs select-none">
                          {cert.expanded ? (
                            <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" d="M5 15l7-7 7 7" /></svg>
                          ) : (
                            <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" /></svg>
                          )}
                        </span>
                        <span className="text-xs text-gray-700 w-[30%] truncate">{cert.subject || '-'}</span>
                        <span className="text-xs text-gray-500 w-[30%] truncate text-center">{cert.thumbprint || '-'}</span>
                        <span className="text-xs text-gray-500 flex-1 text-right">{cert.expiresAt || '-'}</span>
                      </div>
                      <button onClick={(e) => { e.stopPropagation(); setMenuOpenId(menuOpenId === cert.id ? null : cert.id) }} className="text-gray-600 hover:text-gray-800 text-lg font-bold relative ml-4">
                          ⋮
                          {menuOpenId === cert.id && (
                            <div className="absolute right-0 top-5 bg-white border border-border rounded-md shadow-md z-20 py-1 w-28">
                              <button
                                onClick={(e) => { e.stopPropagation(); removeCert(cert.id); setMenuOpenId(null) }}
                                className="flex items-center gap-2 w-full px-3 py-1.5 text-xs text-red-500 hover:bg-gray-50 text-left"
                              >
                                <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" /></svg>
                                Delete
                              </button>
                            </div>
                          )}
                        </button>
                    </div>
                    {/* Expanded form */}
                    {cert.expanded && (
                      <div className="px-4 pb-4 pt-3 space-y-3 bg-gray-50/50">
                        <div>
                          <div className="flex gap-2">
                            <textarea
                              placeholder="Certificate"
                              value={cert.certificate}
                              onChange={(e) => updateCert(cert.id, 'certificate', e.target.value)}
                              rows={5}
                              className="flex-1 px-3 py-2 border border-gray-200 rounded-md text-sm bg-white resize-y focus:outline-none focus:ring-1 focus:ring-gray-300"
                            />
                            <div className="flex flex-col gap-2">
                              <label className="cursor-pointer text-orange-500 hover:text-orange-600" title="Upload">
                                <input type="file" accept=".pem,.cer,.crt" className="hidden" onChange={(e) => {
                                  const file = e.target.files?.[0]
                                  if (file) {
                                    const reader = new FileReader()
                                    reader.onload = (ev) => updateCert(cert.id, 'certificate', ev.target?.result as string || '')
                                    reader.readAsText(file)
                                  }
                                }} />
                                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" /></svg>
                              </label>
                              <button onClick={() => updateCert(cert.id, 'certificate', '')} className="text-orange-500 hover:text-red-500" title="Clear">
                                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" /></svg>
                              </button>
                            </div>
                          </div>
                          <span className="text-[10px] text-red-500 font-medium">Required</span>
                        </div>
                        <input
                          type="text"
                          placeholder="Thumbprint"
                          value={cert.thumbprint}
                          onChange={(e) => updateCert(cert.id, 'thumbprint', e.target.value)}
                          className="w-full px-3 py-2 border border-gray-200 rounded-md text-sm bg-white focus:outline-none focus:ring-1 focus:ring-gray-300"
                        />
                        <input
                          type="text"
                          placeholder="Subject"
                          value={cert.subject}
                          onChange={(e) => updateCert(cert.id, 'subject', e.target.value)}
                          className="w-full px-3 py-2 border border-gray-200 rounded-md text-sm bg-white focus:outline-none focus:ring-1 focus:ring-gray-300"
                        />
                        <input
                          type="text"
                          placeholder="Expires At"
                          value={cert.expiresAt}
                          onChange={(e) => updateCert(cert.id, 'expiresAt', e.target.value)}
                          className="w-full px-3 py-2 border border-gray-200 rounded-md text-sm bg-white focus:outline-none focus:ring-1 focus:ring-gray-300"
                        />
                      </div>
                    )}
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <div className="flex items-center justify-between mt-2">
        <select className="px-2 py-1 border border-border rounded text-xs text-gray-600">
          <option>10</option>
          <option>25</option>
          <option>50</option>
        </select>
        <div className="flex items-center gap-1">
          <button className="px-1.5 py-0.5 text-[10px] text-gray-400">←</button>
          <button className="w-5 h-5 rounded text-[10px] font-medium bg-orange-500 text-white">1</button>
          <button className="px-1.5 py-0.5 text-[10px] text-gray-400">→</button>
        </div>
      </div>
    </div>
  )
}
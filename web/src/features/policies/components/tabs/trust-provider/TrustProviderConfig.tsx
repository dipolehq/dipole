import { useState } from 'react'
import {
  FloatingInput,
  FloatingTextarea,
  FloatingDropdown,
} from '@/shared/components/FloatingInput'
import type { TrustProviderMatchRule } from '@/shared/components/forms/types'

interface TrustProviderConfigProps {
  type: string
  matchRules: TrustProviderMatchRule[]
  onMatchRulesChange: (rules: TrustProviderMatchRule[]) => void
  certificate: string
  onCertificateChange: (v: string) => void
  thumbprint: string
  onThumbprintChange: (v: string) => void
  subject: string
  onSubjectChange: (v: string) => void
  expiresAt: string
  onExpiresAtChange: (v: string) => void
  oidcEndpoint: string
  onOidcEndpointChange: (v: string) => void
  attestationMethod: string
  onAttestationMethodChange: (v: string) => void
  samlMetadataUrl: string
  onSamlMetadataUrlChange: (v: string) => void
  samlMetadataXml: string
  onSamlMetadataXmlChange: (v: string) => void
  symmetricKey: string
  onSymmetricKeyChange: (v: string) => void
  jwks: string
  onJwksChange: (v: string) => void
  publicKey: string
  onPublicKeyChange: (v: string) => void
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
  gcp_identity_token: [{ value: 'email', label: 'email' }],
  kubernetes_service_account: [
    { value: 'iss', label: 'iss' },
    { value: 'kubernetes.io.namespace', label: 'kubernetes.io { namespace }' },
    { value: 'kubernetes.io.pod.name', label: 'kubernetes.io { pod { name } }' },
    {
      value: 'kubernetes.io.serviceaccount.name',
      label: 'kubernetes.io { serviceaccount { name } }',
    },
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
  certificate_signed: [{ value: 'custom', label: 'Custom Attribute' }],
}

const ATTESTATION_METHODS = [
  { value: 'oidc_discovery', label: 'OIDC Discovery' },
  { value: 'symmetric_key', label: 'Symmetric Key' },
  { value: 'upload_jwks', label: 'Upload JWKS' },
  { value: 'upload_public_key', label: 'Upload Public Key' },
  { value: 'tenant_oidc_id_token', label: 'Dipole Tenant OIDC ID Token' },
]

export function TrustProviderConfig({
  type,
  matchRules,
  onMatchRulesChange,
  certificate,
  onCertificateChange,
  thumbprint,
  onThumbprintChange,
  subject,
  onSubjectChange,
  expiresAt,
  onExpiresAtChange,
  oidcEndpoint,
  onOidcEndpointChange,
  attestationMethod,
  onAttestationMethodChange,
  samlMetadataUrl,
  onSamlMetadataUrlChange,
  samlMetadataXml,
  onSamlMetadataXmlChange,
  symmetricKey,
  onSymmetricKeyChange,
  jwks,
  onJwksChange,
  publicKey,
  onPublicKeyChange,
}: TrustProviderConfigProps) {
  const rules = MATCH_RULES[type] || []

  function addMatchRule() {
    onMatchRulesChange([...matchRules, { attribute: '', value: '' }])
  }

  function updateMatchRule(idx: number, field: 'attribute' | 'value', val: string) {
    onMatchRulesChange(matchRules.map((r, i) => (i === idx ? { ...r, [field]: val } : r)))
  }

  function removeMatchRule(idx: number) {
    if (matchRules.length > 1) {
      onMatchRulesChange(matchRules.filter((_, i) => i !== idx))
    }
  }

  if (!type) return null

  const showCertificate = type === 'aws_metadata'
  const showCertificateTable = type === 'certificate_signed'
  const showOidcEndpoint = ['kubernetes_service_account', 'gitlab_job'].includes(type)
  const showAttestationMethod = type === 'oidc_id_token'
  const showMatchRules = !['certificate_signed'].includes(type)
  const showEdgeSdkAudience = [
    'gcp_identity_token',
    'github_action',
    'gitlab_job',
    'terraform_cloud',
    'kubernetes_service_account',
  ].includes(type)
  const showSamlConfig = type === 'samlv2_response'

  return (
    <div className="space-y-5">
      {/* Edge SDK Audience */}
      {showEdgeSdkAudience && (
        <FloatingInput
          label="Edge SDK Audience"
          value="https://<tenant>.id.ap-south-1.dipolehq.com"
          disabled
        />
      )}

      {/* Match Rules */}
      {showMatchRules && (
        <div className="border border-border rounded-md p-4">
          <h4 className="text-xs font-semibold text-gray-800 mb-3">Match Rules</h4>
          <div className="space-y-3">
            {matchRules.map((rule, idx) => (
              <div key={idx} className="flex items-center gap-2">
                <FloatingDropdown
                  label="Attribute"
                  options={rules}
                  value={rule.attribute}
                  onChange={(v) => updateMatchRule(idx, 'attribute', v)}
                  className="w-48"
                />
                <div className="flex-1">
                  <FloatingInput
                    label="Value"
                    value={rule.value}
                    onChange={(v) => updateMatchRule(idx, 'value', v)}
                  />
                </div>
                <button
                  onClick={() => removeMatchRule(idx)}
                  className="text-orange-400 hover:text-red-500 text-sm"
                >
                  ×
                </button>
              </div>
            ))}
          </div>
          <button
            onClick={addMatchRule}
            className="mt-3 text-xs text-orange-500 hover:text-orange-600 font-medium"
          >
            + New
          </button>
        </div>
      )}

      {/* Certificate Table (Certificate Signed Attestation) */}
      {showCertificateTable && <CertificateSignedSection />}

      {/* SAML Configuration */}
      {showSamlConfig && (
        <div className="space-y-4">
          <FloatingInput
            label="Metadata URL"
            value={samlMetadataUrl}
            onChange={onSamlMetadataUrlChange}
          />
          <FloatingTextarea
            label="Metadata XML"
            value={samlMetadataXml}
            onChange={onSamlMetadataXmlChange}
            rows={6}
          />
        </div>
      )}

      {/* Certificate (AWS Metadata) */}
      {showCertificate && (
        <div className="space-y-4">
          <div className="flex gap-2">
            <div className="flex-1">
              <FloatingTextarea
                label="Certificate (PEM)"
                value={certificate}
                onChange={onCertificateChange}
                rows={5}
              />
            </div>
            <div className="flex flex-col gap-2 pt-2">
              <label
                className="cursor-pointer text-orange-500 hover:text-orange-600"
                title="Upload certificate"
              >
                <input
                  type="file"
                  accept=".pem,.cer,.crt"
                  className="hidden"
                  onChange={(e) => {
                    const file = e.target.files?.[0]
                    if (file) {
                      const reader = new FileReader()
                      reader.onload = (ev) =>
                        onCertificateChange((ev.target?.result as string) || '')
                      reader.readAsText(file)
                    }
                  }}
                />
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth="2"
                    d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12"
                  />
                </svg>
              </label>
              <button
                onClick={() => onCertificateChange('')}
                className="text-orange-500 hover:text-red-500"
                title="Clear certificate"
              >
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth="2"
                    d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                  />
                </svg>
              </button>
            </div>
          </div>
          <FloatingInput label="Thumbprint" value={thumbprint} onChange={onThumbprintChange} />
          <FloatingInput label="Subject" value={subject} onChange={onSubjectChange} />
          <FloatingInput label="Expires At" value={expiresAt} onChange={onExpiresAtChange} />
        </div>
      )}

      {/* OIDC Endpoint (Kubernetes, GitLab) */}
      {showOidcEndpoint && (
        <FloatingInput label="OIDC Endpoint" value={oidcEndpoint} onChange={onOidcEndpointChange} />
      )}

      {/* Attestation Method (OIDC ID Token) */}
      {showAttestationMethod && (
        <div className="space-y-4">
          <FloatingDropdown
            label="Attestation Method"
            options={ATTESTATION_METHODS}
            value={attestationMethod}
            onChange={onAttestationMethodChange}
          />
          {attestationMethod === 'oidc_discovery' && (
            <FloatingInput
              label="OIDC Endpoint"
              value={oidcEndpoint}
              onChange={onOidcEndpointChange}
            />
          )}
          {attestationMethod === 'symmetric_key' && (
            <FloatingInput
              label="Symmetric Key (Base64)"
              value={symmetricKey}
              onChange={onSymmetricKeyChange}
            />
          )}
          {attestationMethod === 'upload_jwks' && (
            <FloatingTextarea label="JWKS" value={jwks} onChange={onJwksChange} rows={4} />
          )}
          {attestationMethod === 'upload_public_key' && (
            <div className="space-y-4">
              <FloatingTextarea
                label="Public Key (PEM)"
                value={publicKey}
                onChange={onPublicKeyChange}
                rows={4}
              />
              <FloatingInput label="Thumbprint" value="" disabled />
            </div>
          )}
        </div>
      )}
    </div>
  )
}

function CertificateSignedSection() {
  const [certs, setCerts] = useState<
    {
      id: number
      certificate: string
      thumbprint: string
      subject: string
      expiresAt: string
      expanded: boolean
    }[]
  >([])
  const [nextId, setNextId] = useState(1)
  const [menuOpenId, setMenuOpenId] = useState<number | null>(null)

  function addCert() {
    setCerts([
      ...certs,
      { id: nextId, certificate: '', thumbprint: '', subject: '', expiresAt: '', expanded: true },
    ])
    setNextId(nextId + 1)
  }

  function updateCert(id: number, field: string, val: string) {
    setCerts(certs.map((c) => (c.id === id ? { ...c, [field]: val } : c)))
  }

  function toggleExpand(id: number) {
    setCerts(certs.map((c) => (c.id === id ? { ...c, expanded: !c.expanded } : c)))
  }

  function removeCert(id: number) {
    setCerts(certs.filter((c) => c.id !== id))
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-3">
        <FloatingInput label="Search" value="" onChange={() => {}} />
        <button
          onClick={addCert}
          className="flex items-center gap-1 px-3 py-1.5 border border-orange-500 text-orange-500 hover:bg-orange-50 text-xs font-medium rounded-md transition-colors"
        >
          + Add
        </button>
      </div>
      <div className="border border-border rounded-md overflow-hidden">
        <table className="w-full text-xs">
          <thead className="bg-gray-100">
            <tr>
              <th className="px-3 py-2 text-left font-semibold text-gray-600">Name</th>
              <th className="px-3 py-2 text-center font-semibold text-gray-600">Subject</th>
              <th className="px-3 py-2 text-right font-semibold text-gray-600">Expires At</th>
            </tr>
          </thead>
          <tbody>
            {certs.length === 0 && (
              <tr>
                <td colSpan={3} className="px-3 py-4 text-center text-gray-400 text-xs">
                  No certificates found.
                </td>
              </tr>
            )}
            {certs.map((cert) => (
              <tr key={cert.id}>
                <td colSpan={3} className="p-0">
                  <div className="border border-border rounded-md mx-2 my-2 overflow-hidden">
                    <div
                      className="flex items-center justify-between px-4 py-2.5 bg-white cursor-pointer"
                      onClick={() => toggleExpand(cert.id)}
                    >
                      <div className="flex items-center gap-6 flex-1">
                        <span className="text-gray-500 text-xs select-none">
                          {cert.expanded ? '▲' : '▼'}
                        </span>
                        <span className="text-xs text-gray-700 w-[30%] truncate">
                          {cert.subject || '—'}
                        </span>
                        <span className="text-xs text-gray-500 w-[30%] truncate text-center">
                          {cert.thumbprint || '—'}
                        </span>
                        <span className="text-xs text-gray-500 flex-1 text-right">
                          {cert.expiresAt || '—'}
                        </span>
                      </div>
                      <button
                        onClick={(e) => {
                          e.stopPropagation()
                          setMenuOpenId(menuOpenId === cert.id ? null : cert.id)
                        }}
                        className="text-gray-600 hover:text-gray-800 text-lg font-bold relative ml-4"
                      >
                        ⋮
                        {menuOpenId === cert.id && (
                          <div className="absolute right-0 top-5 bg-white border border-border rounded-md shadow-md z-20 py-1 w-28">
                            <button
                              onClick={(e) => {
                                e.stopPropagation()
                                removeCert(cert.id)
                                setMenuOpenId(null)
                              }}
                              className="flex items-center gap-2 w-full px-3 py-1.5 text-xs text-red-500 hover:bg-gray-50 text-left"
                            >
                              Delete
                            </button>
                          </div>
                        )}
                      </button>
                    </div>
                    {cert.expanded && (
                      <div className="px-4 pb-4 pt-3 space-y-4 bg-gray-50/50">
                        <div className="flex gap-2">
                          <div className="flex-1">
                            <FloatingTextarea
                              label="Certificate (PEM)"
                              value={cert.certificate}
                              onChange={(v) => updateCert(cert.id, 'certificate', v)}
                              rows={5}
                              required
                            />
                          </div>
                          <div className="flex flex-col gap-2 pt-2">
                            <label
                              className="cursor-pointer text-orange-500 hover:text-orange-600"
                              title="Upload"
                            >
                              <input
                                type="file"
                                accept=".pem,.cer,.crt"
                                className="hidden"
                                onChange={(e) => {
                                  const file = e.target.files?.[0]
                                  if (file) {
                                    const reader = new FileReader()
                                    reader.onload = (ev) =>
                                      updateCert(
                                        cert.id,
                                        'certificate',
                                        (ev.target?.result as string) || '',
                                      )
                                    reader.readAsText(file)
                                  }
                                }}
                              />
                              <svg
                                className="w-5 h-5"
                                fill="none"
                                stroke="currentColor"
                                viewBox="0 0 24 24"
                              >
                                <path
                                  strokeLinecap="round"
                                  strokeLinejoin="round"
                                  strokeWidth="2"
                                  d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12"
                                />
                              </svg>
                            </label>
                            <button
                              onClick={() => updateCert(cert.id, 'certificate', '')}
                              className="text-orange-500 hover:text-red-500"
                              title="Clear"
                            >
                              <svg
                                className="w-5 h-5"
                                fill="none"
                                stroke="currentColor"
                                viewBox="0 0 24 24"
                              >
                                <path
                                  strokeLinecap="round"
                                  strokeLinejoin="round"
                                  strokeWidth="2"
                                  d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                                />
                              </svg>
                            </button>
                          </div>
                        </div>
                        <FloatingInput
                          label="Thumbprint"
                          value={cert.thumbprint}
                          onChange={(v) => updateCert(cert.id, 'thumbprint', v)}
                        />
                        <FloatingInput
                          label="Subject"
                          value={cert.subject}
                          onChange={(v) => updateCert(cert.id, 'subject', v)}
                        />
                        <FloatingInput
                          label="Expires At"
                          value={cert.expiresAt}
                          onChange={(v) => updateCert(cert.id, 'expiresAt', v)}
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
    </div>
  )
}

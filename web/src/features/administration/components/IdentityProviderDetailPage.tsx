import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { FileText, Plus, X } from 'lucide-react'
import { PageHeader } from '@/shared/components/PageHeader'
import { FloatingInput, FloatingTextarea, FloatingDropdown } from '@/shared/components/FloatingInput'
import { identityProvidersData, IDP_TYPE_OPTIONS } from '@/data/administration/identityProviders'
import type { IdpType, AuthMethod } from '@/data/administration/identityProviders'

const ROLE_OPTIONS = [
  { value: 'SuperAdmin', label: 'SuperAdmin' },
  { value: 'Auditor', label: 'Auditor' },
]

const DETAIL_SECTIONS = [
  { value: 'details', label: 'Details' },
  { value: 'metrics', label: 'Metrics' },
  { value: 'logs', label: 'Logs' },
] as const

type DetailSection = (typeof DETAIL_SECTIONS)[number]['value']

export function IdentityProviderDetailPage() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const idp = identityProvidersData.find((p) => p.id === id)

  const [selectedSection, setSelectedSection] = useState<DetailSection>('details')
  const [sectionOpen, setSectionOpen] = useState(false)
  const [isEditing, setIsEditing] = useState(false)

  const [name, setName] = useState(idp?.name ?? '')
  const [description, setDescription] = useState(idp?.description ?? '')
  const [idpType, setIdpType] = useState<IdpType>(idp?.type ?? 'saml_2.0')
  const [metadataUrl, setMetadataUrl] = useState(idp?.metadataUrl ?? '')
  const [baseUrl, setBaseUrl] = useState(idp?.baseUrl ?? '')
  const [clientId, setClientId] = useState(idp?.clientId ?? '')
  const [scopes, setScopes] = useState(idp?.scopes ?? 'openid profile email')
  const [pkceRequired, setPkceRequired] = useState(idp?.pkceRequired ?? true)
  const [authMethod, setAuthMethod] = useState<AuthMethod>(idp?.authMethod ?? 'client_secret')
  const [clientSecret, setClientSecret] = useState('')
  const [mappings, setMappings] = useState<{ attributeName: string; attributeValue: string; role: string }[]>([])

  if (!idp) {
    return (
      <div className="flex flex-col h-full">
        <div className="flex items-center justify-between px-4 py-3 shrink-0 min-h-[36px] border-b border-border bg-white"><PageHeader /></div>
        <div className="flex-1 flex items-center justify-center flex-col">
          <p className="text-sm text-gray-500">Identity Provider not found.</p>
          <button onClick={() => navigate('/administration')} className="mt-4 text-xs text-orange-500 hover:text-orange-600 font-medium">← Back to Administration</button>
        </div>
      </div>
    )
  }

  const disabled = !isEditing
  const selectedLabel = DETAIL_SECTIONS.find((s) => s.value === selectedSection)?.label ?? ''

  function handleAddMapping() { setMappings([...mappings, { attributeName: '', attributeValue: '', role: '' }]) }
  function handleRemoveMapping(index: number) { setMappings(mappings.filter((_, i) => i !== index)) }
  function handleMappingChange(index: number, field: string, value: string) {
    const updated = [...mappings]
    const current = updated[index]
    if (current) { updated[index] = { ...current, [field]: value } }
    setMappings(updated)
  }

  function handleEditToggle() {
    if (isEditing) {
      console.info('[save-idp]', { name, description, idpType, metadataUrl, baseUrl, clientId, scopes, pkceRequired, authMethod, mappings })
      setIsEditing(false)
    } else { setIsEditing(true) }
  }

  function handleCancel() {
    if (isEditing) {
      setName(idp?.name ?? ''); setDescription(idp?.description ?? '')
      setIdpType(idp?.type ?? 'saml_2.0'); setMetadataUrl(idp?.metadataUrl ?? '')
      setBaseUrl(idp?.baseUrl ?? ''); setClientId(idp?.clientId ?? '')
      setScopes(idp?.scopes ?? 'openid profile email')
      setPkceRequired(idp?.pkceRequired ?? true); setAuthMethod(idp?.authMethod ?? 'client_secret')
      setIsEditing(false)
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
                  <h4 className="text-sm font-semibold text-gray-800 mb-4">Identity Provider Details</h4>
                  <div className="space-y-4">
                    <FloatingInput label="Name" value={name} onChange={setName} disabled={disabled} required />
                    <FloatingTextarea label="Description (Optional)" value={description} onChange={setDescription} disabled={disabled} rows={2} />
                    <FloatingDropdown label="Identity Provider Type" options={IDP_TYPE_OPTIONS.map((o) => ({ value: o.value, label: o.label }))} value={idpType} onChange={(v) => setIdpType(v as IdpType)} disabled={disabled} required />
                  </div>
                </div>

                {idpType === 'saml_2.0' && (
                  <div className="border border-border rounded-lg p-5">
                    <h4 className="text-sm font-semibold text-gray-800 mb-4">SAML 2.0 Configuration</h4>
                    <div className="space-y-4">
                      <div className="bg-gray-50 rounded-md p-4">
                        <label className="block text-[10px] text-gray-500 mb-0.5">Service Provider Entity ID</label>
                        <div className="text-xs text-gray-700 font-mono">{idp.spEntityId ?? 'https://acme.dipolehq.com/saml/metadata'}</div>
                      </div>
                      <div className="bg-gray-50 rounded-md p-4">
                        <label className="block text-[10px] text-gray-500 mb-0.5">Service Provider Single Sign On URL</label>
                        <div className="text-xs text-gray-700 font-mono">{idp.ssoUrl ?? 'https://acme.dipolehq.com/saml/acs'}</div>
                      </div>
                      <FloatingInput label="Metadata URL" value={metadataUrl} onChange={setMetadataUrl} disabled={disabled} />
                    </div>
                  </div>
                )}

                {idpType === 'oidc_1.0' && (
                  <div className="border border-border rounded-lg p-5">
                    <h4 className="text-sm font-semibold text-gray-800 mb-4">OIDC 1.0 Configuration</h4>
                    <div className="space-y-4">
                      <FloatingInput label="Identity Provider Base URL" value={baseUrl} onChange={setBaseUrl} disabled={disabled} required />
                      <FloatingInput label="Client ID" value={clientId} onChange={setClientId} disabled={disabled} required />
                      <FloatingInput label="Scopes" value={scopes} onChange={setScopes} disabled={disabled} />
                      <div className="flex items-center gap-2">
                        <input type="checkbox" checked={pkceRequired} onChange={(e) => setPkceRequired(e.target.checked)} disabled={disabled} className="w-4 h-4 accent-orange-500 cursor-pointer disabled:cursor-default disabled:opacity-60" />
                        <span className="text-sm text-gray-700">PKCE Required</span>
                      </div>
                      <div>
                        <label className="block text-xs text-orange-600 font-medium mb-2">Authentication Method</label>
                        <div className="flex items-center gap-4">
                          <label className="flex items-center gap-1.5 cursor-pointer">
                            <input type="radio" name="authMethodDetail" checked={authMethod === 'client_secret'} onChange={() => !disabled && setAuthMethod('client_secret')} disabled={disabled} className="w-3.5 h-3.5 accent-orange-500 disabled:opacity-60" />
                            <span className="text-sm text-gray-700">Client Secret</span>
                          </label>
                          <label className="flex items-center gap-1.5 cursor-pointer">
                            <input type="radio" name="authMethodDetail" checked={authMethod === 'public_private_keypair'} onChange={() => !disabled && setAuthMethod('public_private_keypair')} disabled={disabled} className="w-3.5 h-3.5 accent-orange-500 disabled:opacity-60" />
                            <span className="text-sm text-gray-700">Public Private Keypair</span>
                          </label>
                        </div>
                      </div>
                      {authMethod === 'client_secret' && !disabled && (
                        <FloatingInput label="Client Secret" value={clientSecret} onChange={setClientSecret} type="password" />
                      )}
                      {authMethod === 'public_private_keypair' && (
                        <div className="bg-gray-50 rounded-md p-4">
                          <label className="block text-[10px] text-gray-500 mb-0.5">JWKS URL</label>
                          <div className="text-xs text-gray-700 font-mono">https://acme.dipolehq.com/.well-known/jwks.json</div>
                        </div>
                      )}
                      <div className="bg-gray-50 rounded-md p-4">
                        <label className="block text-[10px] text-gray-500 mb-0.5">Redirect URL</label>
                        <div className="text-xs text-gray-700 font-mono">{idp.redirectUrl ?? 'https://acme.dipolehq.com/oidc/callback'}</div>
                      </div>
                    </div>
                  </div>
                )}

                <div className="border border-border rounded-lg p-5">
                  <h4 className="text-sm font-semibold text-gray-800 mb-4">Role Mappings</h4>
                  <p className="text-xs text-gray-600 mb-4">Map your IdP&apos;s group claims to Dipole user roles for automatic user creation.</p>
                  {mappings.length > 0 && (
                    <div className="border border-border rounded-lg overflow-hidden mb-4">
                      <table className="w-full text-xs table-fixed">
                        <colgroup><col className="w-[30%]" /><col className="w-[30%]" /><col className="w-[30%]" /><col className="w-[32px]" /></colgroup>
                        <thead>
                          <tr className="bg-gray-50 border-b border-border">
                            <th className="px-4 py-2 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Attribute Name</th>
                            <th className="px-4 py-2 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Attribute Value</th>
                            <th className="px-4 py-2 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Dipole Role</th>
                            <th className="px-2 py-2" />
                          </tr>
                        </thead>
                        <tbody className="divide-y divide-border">
                          {mappings.map((mapping, index) => (
                            <tr key={index}>
                              <td className="px-4 py-2"><input type="text" value={mapping.attributeName} onChange={(e) => handleMappingChange(index, 'attributeName', e.target.value)} disabled={disabled} placeholder="e.g., groups" className="w-full px-2 py-1 border border-border rounded text-xs focus:outline-none focus:ring-1 focus:ring-orange-300" /></td>
                              <td className="px-4 py-2"><input type="text" value={mapping.attributeValue} onChange={(e) => handleMappingChange(index, 'attributeValue', e.target.value)} disabled={disabled} placeholder="e.g., DipoleAdmins" className="w-full px-2 py-1 border border-border rounded text-xs focus:outline-none focus:ring-1 focus:ring-orange-300" /></td>
                              <td className="px-4 py-2"><FloatingDropdown label="Role" options={ROLE_OPTIONS} value={mapping.role} onChange={(v) => handleMappingChange(index, 'role', v)} disabled={disabled} /></td>
                              <td className="px-2 py-2">{!disabled && <button type="button" onClick={() => handleRemoveMapping(index)} className="text-orange-400 hover:text-orange-600 transition-colors"><X size={14} /></button>}</td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  )}
                  {!disabled && (
                    <button type="button" onClick={handleAddMapping} className="flex items-center gap-1.5 text-xs text-orange-500 hover:text-orange-600 font-medium"><Plus size={13} />New Mapping</button>
                  )}
                </div>
              </div>
            </div>

            {/* Right panel — 40% */}
            <div className="w-[40%] px-6 py-6 overflow-y-auto">
              <div className="border border-border rounded-lg p-5 mb-5">
                <h4 className="text-sm font-semibold text-gray-800 mb-4">Metadata</h4>
                <div className="space-y-4">
                  <FloatingInput label="ID" value={idp.id} disabled />
                  <FloatingInput label="Type" value={idp.type === 'saml_2.0' ? 'SAML 2.0' : 'OIDC 1.0'} disabled />
                  <FloatingInput label="Status" value={idp.isActive ? 'Active' : 'Inactive'} disabled />
                </div>
              </div>
              <div className="border border-border rounded-lg p-5">
                <p className="text-xs text-gray-700 leading-relaxed mb-3">
                  <span className="font-semibold text-orange-600">Identity Providers</span> enable Single Sign-On (SSO) for your organization using SAML 2.0 or OIDC 1.0.
                </p>
                <p className="text-xs text-gray-700 leading-relaxed">
                  <span className="font-semibold">Role Mappings</span> allow automatic user provisioning based on IdP group claims.
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

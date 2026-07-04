import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { Plus, X } from 'lucide-react'
import { PageHeader } from '@/shared/components/PageHeader'
import { Dropdown } from '@/shared/components/Dropdown'
import { identityProvidersData, IDP_TYPE_OPTIONS } from '@/data/administration/identityProviders'
import type { IdpType, AuthMethod } from '@/data/administration/identityProviders'

const ROLE_OPTIONS = [
  { value: 'SuperAdmin', label: 'SuperAdmin' },
  { value: 'Auditor', label: 'Auditor' },
]

type TabKey = 'details' | 'mappings'

export function IdentityProviderDetailPage() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()

  const idp = identityProvidersData.find((p) => p.id === id)

  const [isEditing, setIsEditing] = useState(false)
  const [activeTab, setActiveTab] = useState<TabKey>('details')

  // Details state
  const [name, setName] = useState(idp?.name ?? '')
  const [description, setDescription] = useState(idp?.description ?? '')
  const [idpType, setIdpType] = useState<IdpType>(idp?.type ?? 'saml_2.0')

  // SAML state
  const [metadataUrl, setMetadataUrl] = useState(idp?.metadataUrl ?? '')

  // OIDC state
  const [baseUrl, setBaseUrl] = useState(idp?.baseUrl ?? '')
  const [clientId, setClientId] = useState(idp?.clientId ?? '')
  const [scopes, setScopes] = useState(idp?.scopes ?? 'openid profile email')
  const [pkceRequired, setPkceRequired] = useState(idp?.pkceRequired ?? true)
  const [authMethod, setAuthMethod] = useState<AuthMethod>(idp?.authMethod ?? 'client_secret')
  const [clientSecret, setClientSecret] = useState('')

  // Mappings state
  const [mappings, setMappings] = useState<{ attributeName: string; attributeValue: string; role: string }[]>([])

  if (!idp) {
    return (
      <div className="p-6">
        <PageHeader />
        <p className="mt-6 text-sm text-gray-500">Identity Provider not found.</p>
        <button
          onClick={() => navigate('/administration')}
          className="mt-4 text-xs text-orange-500 hover:text-orange-600 font-medium"
        >
          ← Back to Administration
        </button>
      </div>
    )
  }

  const disabled = !isEditing

  function handleAddMapping() {
    setMappings([...mappings, { attributeName: '', attributeValue: '', role: '' }])
  }

  function handleRemoveMapping(index: number) {
    setMappings(mappings.filter((_, i) => i !== index))
  }

  function handleMappingChange(index: number, field: string, value: string) {
    const updated = [...mappings]
    const current = updated[index]
    if (current) {
      updated[index] = { ...current, [field]: value }
    }
    setMappings(updated)
  }

  function handleEditToggle() {
    if (isEditing) {
      console.info('[save-idp]', { name, description, idpType, metadataUrl, baseUrl, clientId, scopes, pkceRequired, authMethod, mappings })
      setIsEditing(false)
    } else {
      setIsEditing(true)
    }
  }

  function handleCancel() {
    if (isEditing) {
      setName(idp?.name ?? '')
      setDescription(idp?.description ?? '')
      setIdpType(idp?.type ?? 'saml_2.0')
      setMetadataUrl(idp?.metadataUrl ?? '')
      setBaseUrl(idp?.baseUrl ?? '')
      setClientId(idp?.clientId ?? '')
      setScopes(idp?.scopes ?? 'openid profile email')
      setPkceRequired(idp?.pkceRequired ?? true)
      setAuthMethod(idp?.authMethod ?? 'client_secret')
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
        {/* Left sidebar */}
        <nav className="w-48 border-r border-border py-3 shrink-0 flex flex-col">
          <div className="flex-1">
            <button
              type="button"
              onClick={() => setActiveTab('details')}
              className={`w-full text-left px-5 py-2.5 text-xs font-medium transition-colors relative ${
                activeTab === 'details' ? 'text-orange-600 bg-orange-50' : 'text-gray-500 hover:text-gray-800 hover:bg-gray-50'
              }`}
            >
              {activeTab === 'details' && <span className="absolute left-0 top-0 bottom-0 w-0.5 bg-orange-500 rounded-r" />}
              <span>Details</span>
            </button>
            <button
              type="button"
              onClick={() => setActiveTab('mappings')}
              className={`w-full text-left px-5 py-2.5 text-xs font-medium transition-colors relative ${
                activeTab === 'mappings' ? 'text-orange-600 bg-orange-50' : 'text-gray-500 hover:text-gray-800 hover:bg-gray-50'
              }`}
            >
              {activeTab === 'mappings' && <span className="absolute left-0 top-0 bottom-0 w-0.5 bg-orange-500 rounded-r" />}
              <span>Mappings</span>
            </button>
          </div>
          <div className="px-4 py-3 border-t border-border space-y-2">
            <button
              type="button"
              onClick={handleEditToggle}
              className="w-full px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors"
            >
              {isEditing ? 'Save Identity Provider' : 'Edit Identity Provider'}
            </button>
            <button
              type="button"
              onClick={handleCancel}
              className="w-full px-4 py-2 border border-gray-300 text-xs font-medium text-gray-600 hover:text-gray-900 hover:border-gray-400 rounded-md transition-colors"
            >
              {isEditing ? 'Cancel' : 'Back'}
            </button>
          </div>
        </nav>

        {/* Main content */}
        <div className="flex-1 overflow-y-auto p-6">
          {activeTab === 'details' ? (
            <div>
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-sm font-semibold text-gray-900">Identity Provider</h3>
                {isEditing && (
                  <button
                    type="button"
                    onClick={handleEditToggle}
                    className="px-4 py-1.5 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors"
                  >
                    Save
                  </button>
                )}
              </div>
              <div className="border-b border-border mb-6" />

              <div className="space-y-4">
                <div>
                  <label className="block text-xs text-gray-500 mb-1">Name</label>
                  <input
                    type="text"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    disabled={disabled}
                    placeholder="e.g., Okta SSO"
                    className={`w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400 ${disabled ? 'bg-gray-50 text-gray-600' : ''}`}
                  />
                </div>
                <div>
                  <label className="block text-xs text-gray-500 mb-1">Description</label>
                  <textarea
                    value={description}
                    onChange={(e) => setDescription(e.target.value)}
                    disabled={disabled}
                    placeholder="Description"
                    rows={2}
                    className={`w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400 resize-y ${disabled ? 'bg-gray-50 text-gray-600' : ''}`}
                  />
                </div>
                <div>
                  <label className="block text-xs text-gray-500 mb-1">Identity Provider Type</label>
                  {disabled ? (
                    <div className="px-3 py-2 border border-border rounded-md text-sm text-gray-600 bg-gray-50">
                      {idpType === 'saml_2.0' ? 'SAML 2.0' : 'OIDC 1.0'}
                    </div>
                  ) : (
                    <Dropdown
                      options={IDP_TYPE_OPTIONS.map((o) => ({ value: o.value, label: o.label }))}
                      value={idpType}
                      onChange={(v) => setIdpType(v as IdpType)}
                      placeholder="Select type..."
                    />
                  )}
                </div>

                <div className="border-t border-border pt-4 mt-4" />

                {idpType === 'saml_2.0' ? (
                  <div className="space-y-4">
                    <h4 className="text-[11px] font-bold text-gray-600 uppercase tracking-wider">SAML 2.0 Configuration</h4>
                    <div>
                      <label className="block text-xs text-gray-500 mb-1">Service Provider Entity ID</label>
                      <div className="flex items-center">
                        <div className="flex-1 px-3 py-2 border border-border rounded-md text-sm text-gray-500 bg-gray-50 font-mono">{idp.spEntityId ?? '—'}</div>
                        <button type="button" className="ml-2 text-gray-400 hover:text-gray-600"><svg className="w-4 h-4" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24"><rect x="9" y="9" width="13" height="13" rx="2" /><path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1" /></svg></button>
                      </div>
                    </div>
                    <div>
                      <label className="block text-xs text-gray-500 mb-1">Service Provider Single Sign On URL</label>
                      <div className="flex items-center">
                        <div className="flex-1 px-3 py-2 border border-border rounded-md text-sm text-gray-500 bg-gray-50 font-mono">{idp.ssoUrl ?? '—'}</div>
                        <button type="button" className="ml-2 text-gray-400 hover:text-gray-600"><svg className="w-4 h-4" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24"><rect x="9" y="9" width="13" height="13" rx="2" /><path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1" /></svg></button>
                      </div>
                    </div>
                    <div>
                      <label className="block text-xs text-gray-500 mb-1">Metadata URL</label>
                      <input
                        type="url"
                        value={metadataUrl}
                        onChange={(e) => setMetadataUrl(e.target.value)}
                        disabled={disabled}
                        placeholder="https://your-idp.com/saml/metadata"
                        className={`w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400 ${disabled ? 'bg-gray-50 text-gray-600' : ''}`}
                      />
                    </div>
                  </div>
                ) : (
                  <div className="space-y-4">
                    <h4 className="text-[11px] font-bold text-gray-600 uppercase tracking-wider">OIDC 1.0 Configuration</h4>
                    <div>
                      <label className="block text-xs text-gray-500 mb-1">Identity Provider Base URL</label>
                      <input
                        type="url"
                        value={baseUrl}
                        onChange={(e) => setBaseUrl(e.target.value)}
                        disabled={disabled}
                        placeholder="https://accounts.google.com"
                        className={`w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400 ${disabled ? 'bg-gray-50 text-gray-600' : ''}`}
                      />
                    </div>
                    <div>
                      <label className="block text-xs text-gray-500 mb-1">Client ID</label>
                      <input
                        type="text"
                        value={clientId}
                        onChange={(e) => setClientId(e.target.value)}
                        disabled={disabled}
                        placeholder="Client ID"
                        className={`w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400 ${disabled ? 'bg-gray-50 text-gray-600' : ''}`}
                      />
                    </div>
                    <div>
                      <label className="block text-xs text-gray-500 mb-1">Scopes</label>
                      <input
                        type="text"
                        value={scopes}
                        onChange={(e) => setScopes(e.target.value)}
                        disabled={disabled}
                        placeholder="openid profile email"
                        className={`w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400 ${disabled ? 'bg-gray-50 text-gray-600' : ''}`}
                      />
                    </div>
                    <div className="flex items-center gap-2">
                      <input
                        type="checkbox"
                        checked={pkceRequired}
                        onChange={(e) => setPkceRequired(e.target.checked)}
                        disabled={disabled}
                        className="w-4 h-4 accent-orange-500 cursor-pointer disabled:cursor-default disabled:opacity-60"
                      />
                      <span className="text-sm text-gray-700">PKCE Required</span>
                    </div>
                    <div>
                      <label className="block text-xs text-orange-600 font-medium mb-2">Authentication Method</label>
                      <div className="flex items-center gap-4">
                        <label className="flex items-center gap-1.5 cursor-pointer">
                          <input
                            type="radio"
                            name="authMethodDetail"
                            checked={authMethod === 'client_secret'}
                            onChange={() => !disabled && setAuthMethod('client_secret')}
                            disabled={disabled}
                            className="w-3.5 h-3.5 accent-orange-500 disabled:opacity-60"
                          />
                          <span className="text-sm text-gray-700">Client Secret</span>
                        </label>
                        <label className="flex items-center gap-1.5 cursor-pointer">
                          <input
                            type="radio"
                            name="authMethodDetail"
                            checked={authMethod === 'public_private_keypair'}
                            onChange={() => !disabled && setAuthMethod('public_private_keypair')}
                            disabled={disabled}
                            className="w-3.5 h-3.5 accent-orange-500 disabled:opacity-60"
                          />
                          <span className="text-sm text-gray-700">Public Private Keypair</span>
                        </label>
                      </div>
                    </div>
                    {authMethod === 'client_secret' && !disabled && (
                      <div>
                        <label className="block text-xs text-gray-500 mb-1">Client Secret</label>
                        <input
                          type="password"
                          value={clientSecret}
                          onChange={(e) => setClientSecret(e.target.value)}
                          placeholder="Client secret"
                          className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400"
                        />
                      </div>
                    )}
                    {authMethod === 'public_private_keypair' && (
                      <div className="bg-gray-50 rounded-md p-4">
                        <label className="block text-[10px] text-gray-500 mb-0.5">JWKS URL</label>
                        <div className="text-xs text-gray-700 font-mono">https://acme.dipolehq.com/.well-known/jwks.json</div>
                      </div>
                    )}
                    <div className="bg-gray-50 rounded-md p-4">
                      <label className="block text-[10px] text-gray-500 mb-0.5">Redirect URL</label>
                      <div className="text-xs text-gray-700 font-mono">{idp.redirectUrl ?? '—'}</div>
                    </div>
                  </div>
                )}
              </div>
            </div>
          ) : (
            <div>
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-sm font-semibold text-gray-900">Role Mappings</h3>
                {isEditing && (
                  <button
                    type="button"
                    onClick={handleEditToggle}
                    className="px-4 py-1.5 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors"
                  >
                    Save
                  </button>
                )}
              </div>
              <div className="border-b border-border mb-6" />

              <p className="text-xs text-gray-600 mb-4">
                Map your IdP&apos;s group claims to Dipole user roles for automatic user creation.
              </p>

              {(mappings.length > 0 || !disabled) && (
                <div className="border border-border rounded-lg overflow-hidden mb-4">
                  <table className="w-full text-xs table-fixed">
                    <colgroup>
                      <col className="w-[30%]" />
                      <col className="w-[30%]" />
                      <col className="w-[30%]" />
                      <col className="w-[32px]" />
                    </colgroup>
                    <thead>
                      <tr className="bg-gray-50 border-b border-border">
                        <th className="px-4 py-2 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Attribute Name</th>
                        <th className="px-4 py-2 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Attribute Value</th>
                        <th className="px-4 py-2 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Dipole Role</th>
                        <th className="px-2 py-2" />
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-border">
                      {mappings.length === 0 && (
                        <tr>
                          <td colSpan={4} className="px-4 py-3 text-center text-gray-400 text-xs">No mappings configured.</td>
                        </tr>
                      )}
                      {mappings.map((mapping, index) => (
                        <tr key={index}>
                          <td className="px-4 py-2">
                            <input
                              type="text"
                              value={mapping.attributeName}
                              onChange={(e) => handleMappingChange(index, 'attributeName', e.target.value)}
                              disabled={disabled}
                              placeholder="e.g., groups"
                              className={`w-full px-2 py-1 border border-border rounded text-xs focus:outline-none focus:ring-1 focus:ring-orange-300 ${disabled ? 'bg-gray-50 text-gray-600' : ''}`}
                            />
                          </td>
                          <td className="px-4 py-2">
                            <input
                              type="text"
                              value={mapping.attributeValue}
                              onChange={(e) => handleMappingChange(index, 'attributeValue', e.target.value)}
                              disabled={disabled}
                              placeholder="e.g., DipoleAdmins"
                              className={`w-full px-2 py-1 border border-border rounded text-xs focus:outline-none focus:ring-1 focus:ring-orange-300 ${disabled ? 'bg-gray-50 text-gray-600' : ''}`}
                            />
                          </td>
                          <td className="px-4 py-2">
                            {disabled ? (
                              <div className="px-2 py-1 text-xs text-gray-600">{mapping.role || '—'}</div>
                            ) : (
                              <Dropdown
                                options={ROLE_OPTIONS}
                                value={mapping.role}
                                onChange={(v) => handleMappingChange(index, 'role', v)}
                                placeholder="Select role"
                              />
                            )}
                          </td>
                          <td className="px-2 py-2">
                            {!disabled && (
                              <button
                                type="button"
                                onClick={() => handleRemoveMapping(index)}
                                className="text-orange-400 hover:text-orange-600 transition-colors"
                              >
                                <X size={14} />
                              </button>
                            )}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}

              {!disabled && (
                <button
                  type="button"
                  onClick={handleAddMapping}
                  className="flex items-center gap-1.5 text-xs text-orange-500 hover:text-orange-600 font-medium"
                >
                  <Plus size={13} />
                  New Mapping
                </button>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

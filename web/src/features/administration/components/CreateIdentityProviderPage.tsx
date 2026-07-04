import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Plus, X } from 'lucide-react'
import { PageHeader } from '@/shared/components/PageHeader'
import { Dropdown } from '@/shared/components/Dropdown'
import { IDP_TYPE_OPTIONS } from '@/data/administration/identityProviders'
import type { IdpType, AuthMethod } from '@/data/administration/identityProviders'

const ROLE_OPTIONS = [
  { value: 'SuperAdmin', label: 'SuperAdmin' },
  { value: 'Auditor', label: 'Auditor' },
]

type TabKey = 'details' | 'mappings'

export function CreateIdentityProviderPage() {
  const navigate = useNavigate()
  const [activeTab, setActiveTab] = useState<TabKey>('details')

  // Details state
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [idpType, setIdpType] = useState<IdpType>('saml_2.0')

  // SAML state
  const [metadataUrl, setMetadataUrl] = useState('')
  const [metadataXml, setMetadataXml] = useState('')

  // OIDC state
  const [baseUrl, setBaseUrl] = useState('')
  const [clientId, setClientId] = useState('')
  const [scopes, setScopes] = useState('openid profile email')
  const [pkceRequired, setPkceRequired] = useState(true)
  const [authMethod, setAuthMethod] = useState<AuthMethod>('client_secret')
  const [clientSecret, setClientSecret] = useState('')

  // Mappings state
  const [mappings, setMappings] = useState<{ attributeName: string; attributeValue: string; role: string }[]>([])

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

  function handleSave() {
    console.info('[create-idp] submitted', { name, description, idpType, metadataUrl, baseUrl, clientId, scopes, pkceRequired, authMethod, mappings })
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
              onClick={handleSave}
              className="w-full px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors"
            >
              Save Identity Provider
            </button>
            <button
              type="button"
              onClick={handleCancel}
              className="w-full px-4 py-2 border border-gray-300 text-xs font-medium text-gray-600 hover:text-gray-900 hover:border-gray-400 rounded-md transition-colors"
            >
              Cancel
            </button>
          </div>
        </nav>

        {/* Main content */}
        <div className="flex-1 overflow-y-auto p-6">
          {activeTab === 'details' ? (
            <div>
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-sm font-semibold text-gray-900">Identity Provider</h3>
                <button
                  type="button"
                  onClick={handleSave}
                  className="px-4 py-1.5 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors"
                >
                  Save
                </button>
              </div>
              <div className="border-b border-border mb-6" />

              <div className="space-y-4">
                <div>
                  <label className="block text-xs text-gray-500 mb-1">Name <span className="text-red-500">*</span></label>
                  <input
                    type="text"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    placeholder="e.g., Okta SSO"
                    className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400"
                  />
                </div>
                <div>
                  <label className="block text-xs text-gray-500 mb-1">Description (Optional)</label>
                  <textarea
                    value={description}
                    onChange={(e) => setDescription(e.target.value)}
                    placeholder="Description"
                    rows={2}
                    className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400 resize-y"
                  />
                </div>
                <div>
                  <label className="block text-xs text-gray-500 mb-1">Identity Provider Type <span className="text-red-500">*</span></label>
                  <Dropdown
                    options={IDP_TYPE_OPTIONS.map((o) => ({ value: o.value, label: o.label }))}
                    value={idpType}
                    onChange={(v) => setIdpType(v as IdpType)}
                    placeholder="Select type..."
                  />
                </div>

                <div className="border-t border-border pt-4 mt-4" />

                {idpType === 'saml_2.0' ? (
                  <div className="space-y-4">
                    <div>
                      <label className="block text-xs text-gray-500 mb-1">Service Provider Entity ID</label>
                      <div className="flex items-center">
                        <div className="flex-1 px-3 py-2 border border-border rounded-md text-sm text-gray-500 bg-gray-50">https://acme.dipolehq.com/saml/metadata</div>
                        <button type="button" className="ml-2 text-gray-400 hover:text-gray-600"><svg className="w-4 h-4" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24"><rect x="9" y="9" width="13" height="13" rx="2" /><path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1" /></svg></button>
                      </div>
                    </div>
                    <div>
                      <label className="block text-xs text-gray-500 mb-1">Service Provider Single Sign On URL</label>
                      <div className="flex items-center">
                        <div className="flex-1 px-3 py-2 border border-border rounded-md text-sm text-gray-500 bg-gray-50">https://acme.dipolehq.com/saml/acs</div>
                        <button type="button" className="ml-2 text-gray-400 hover:text-gray-600"><svg className="w-4 h-4" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24"><rect x="9" y="9" width="13" height="13" rx="2" /><path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1" /></svg></button>
                      </div>
                    </div>
                    <div>
                      <label className="block text-xs text-gray-500 mb-1">Metadata URL</label>
                      <input
                        type="url"
                        value={metadataUrl}
                        onChange={(e) => setMetadataUrl(e.target.value)}
                        placeholder="https://your-idp.com/saml/metadata"
                        className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400"
                      />
                    </div>
                    <div>
                      <div className="flex items-center justify-between mb-1">
                        <label className="block text-xs text-gray-500">Metadata XML</label>
                        <div className="flex items-center gap-1.5">
                          <button type="button" className="w-7 h-7 flex items-center justify-center bg-orange-500 hover:bg-orange-600 text-white rounded-md transition-colors">
                            <svg className="w-4 h-4" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" /></svg>
                          </button>
                          <button type="button" onClick={() => setMetadataXml('')} className="w-7 h-7 flex items-center justify-center bg-orange-500 hover:bg-orange-600 text-white rounded-md transition-colors">
                            <svg className="w-4 h-4" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" /></svg>
                          </button>
                        </div>
                      </div>
                      <textarea
                        value={metadataXml}
                        onChange={(e) => setMetadataXml(e.target.value)}
                        placeholder="Paste XML metadata here..."
                        rows={4}
                        className="w-full px-3 py-2 border border-border rounded-md text-sm font-mono focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400 resize-y"
                      />
                    </div>
                  </div>
                ) : (
                  <div className="space-y-4">
                    <div>
                      <label className="block text-xs text-gray-500 mb-1">Identity Provider Base URL <span className="text-red-500">*</span></label>
                      <input
                        type="url"
                        value={baseUrl}
                        onChange={(e) => setBaseUrl(e.target.value)}
                        placeholder="https://accounts.google.com"
                        className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400"
                      />
                    </div>
                    <div>
                      <label className="block text-xs text-gray-500 mb-1">Identity Provider Client ID <span className="text-red-500">*</span></label>
                      <input
                        type="text"
                        value={clientId}
                        onChange={(e) => setClientId(e.target.value)}
                        placeholder="Client ID from your IdP"
                        className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400"
                      />
                    </div>
                    <div>
                      <label className="block text-xs text-gray-500 mb-1">Identity Provider Scopes</label>
                      <input
                        type="text"
                        value={scopes}
                        onChange={(e) => setScopes(e.target.value)}
                        placeholder="openid profile email"
                        className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400"
                      />
                    </div>
                    <div className="flex items-center gap-2">
                      <input
                        type="checkbox"
                        checked={pkceRequired}
                        onChange={(e) => setPkceRequired(e.target.checked)}
                        className="w-4 h-4 accent-orange-500 cursor-pointer"
                      />
                      <span className="text-sm text-gray-700">PKCE Required</span>
                    </div>
                    <div>
                      <label className="block text-xs text-orange-600 font-medium mb-2">Authentication Method</label>
                      <div className="flex items-center gap-4">
                        <label className="flex items-center gap-1.5 cursor-pointer">
                          <input
                            type="radio"
                            name="authMethod"
                            checked={authMethod === 'client_secret'}
                            onChange={() => setAuthMethod('client_secret')}
                            className="w-3.5 h-3.5 accent-orange-500"
                          />
                          <span className="text-sm text-gray-700">Client Secret</span>
                        </label>
                        <label className="flex items-center gap-1.5 cursor-pointer">
                          <input
                            type="radio"
                            name="authMethod"
                            checked={authMethod === 'public_private_keypair'}
                            onChange={() => setAuthMethod('public_private_keypair')}
                            className="w-3.5 h-3.5 accent-orange-500"
                          />
                          <span className="text-sm text-gray-700">Public Private Keypair</span>
                        </label>
                      </div>
                    </div>
                    {authMethod === 'client_secret' && (
                      <div>
                        <label className="block text-xs text-gray-500 mb-1">Client Secret</label>
                        <input
                          type="password"
                          value={clientSecret}
                          onChange={(e) => setClientSecret(e.target.value)}
                          placeholder="Client secret from your IdP"
                          className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400"
                        />
                      </div>
                    )}
                    {authMethod === 'public_private_keypair' && (
                      <div className="bg-gray-50 rounded-md p-4">
                        <label className="block text-[10px] text-gray-500 mb-0.5">JWKS URL (configure in your IdP)</label>
                        <div className="text-xs text-gray-700 font-mono">https://acme.dipolehq.com/.well-known/jwks.json</div>
                      </div>
                    )}
                    {(baseUrl || clientId) && (
                      <div className="bg-gray-50 rounded-md p-4">
                        <label className="block text-[10px] text-gray-500 mb-0.5">Redirect URL (configure in your IdP)</label>
                        <div className="text-xs text-gray-700 font-mono">https://acme.dipolehq.com/oidc/callback</div>
                      </div>
                    )}
                  </div>
                )}
              </div>
            </div>
          ) : (
            <div>
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-sm font-semibold text-gray-900">Role Mappings</h3>
                <button
                  type="button"
                  onClick={handleSave}
                  className="px-4 py-1.5 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors"
                >
                  Save
                </button>
              </div>
              <div className="border-b border-border mb-6" />

              <p className="text-xs text-gray-600 mb-4">
                Map your IdP&apos;s group claims to Dipole user roles. When a user authenticates via SSO, Dipole uses these mappings to automatically create user accounts with the correct roles.
              </p>

              {/* Mappings table */}
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
                        <td colSpan={4} className="px-4 py-3 text-center text-gray-400 text-xs">No mappings configured. Click + New Mapping to add one.</td>
                      </tr>
                    )}
                    {mappings.map((mapping, index) => (
                      <tr key={index}>
                        <td className="px-4 py-2">
                          <input
                            type="text"
                            value={mapping.attributeName}
                            onChange={(e) => handleMappingChange(index, 'attributeName', e.target.value)}
                            placeholder="e.g., groups"
                            className="w-full px-2 py-1 border border-border rounded text-xs focus:outline-none focus:ring-1 focus:ring-orange-300"
                          />
                        </td>
                        <td className="px-4 py-2">
                          <input
                            type="text"
                            value={mapping.attributeValue}
                            onChange={(e) => handleMappingChange(index, 'attributeValue', e.target.value)}
                            placeholder="e.g., DipoleAdmins"
                            className="w-full px-2 py-1 border border-border rounded text-xs focus:outline-none focus:ring-1 focus:ring-orange-300"
                          />
                        </td>
                        <td className="px-4 py-2">
                          <Dropdown
                            options={ROLE_OPTIONS}
                            value={mapping.role}
                            onChange={(v) => handleMappingChange(index, 'role', v)}
                            placeholder="Select role"
                          />
                        </td>
                        <td className="px-2 py-2">
                          <button
                            type="button"
                            onClick={() => handleRemoveMapping(index)}
                            className="text-orange-400 hover:text-orange-600 transition-colors"
                          >
                            <X size={14} />
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>

              <button
                type="button"
                onClick={handleAddMapping}
                className="flex items-center gap-1.5 text-xs text-orange-500 hover:text-orange-600 font-medium"
              >
                <Plus size={13} />
                New Mapping
              </button>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

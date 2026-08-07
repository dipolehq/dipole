import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Plus, X } from 'lucide-react'
import { PageHeader } from '@/shared/components/PageHeader'
import { FloatingInput, FloatingTextarea, FloatingDropdown } from '@/shared/components/FloatingInput'
import { IDP_TYPE_OPTIONS } from '@/shared/constants/administration'
import { useCreateIdentityProvider } from '@/features/administration/hooks/useIdentityProviders'
import { apiErrorMessage } from '@/features/administration/api/identityProvidersApi'
import type { IdpType, AuthMethod } from '@/shared/constants/administration'

const ROLE_OPTIONS = [
  { value: 'SuperAdmin', label: 'SuperAdmin' },
  { value: 'Auditor', label: 'Auditor' },
]

export function CreateIdentityProviderPage() {
  const navigate = useNavigate()
  const createMutation = useCreateIdentityProvider()
  const [error, setError] = useState<string | null>(null)

  // Details state
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [idpType, setIdpType] = useState<IdpType | ''>('')

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

  async function handleSave() {
    setError(null)
    const config: Record<string, unknown> = {}
    if (idpType === 'saml_2.0') {
      config.metadataUrl = metadataUrl
      config.metadataXml = metadataXml
    } else if (idpType === 'oidc_1.0') {
      config.baseUrl = baseUrl
      config.clientId = clientId
      config.scopes = scopes
      config.pkceRequired = pkceRequired
      config.authMethod = authMethod
      if (authMethod === 'client_secret') config.clientSecret = clientSecret
    }
    config.mappings = mappings
    try {
      await createMutation.mutateAsync({
        name,
        description,
        isActive: true,
        type: idpType,
        config,
      })
      navigate('/administration?tab=identity_providers')
    } catch (err) {
      setError(apiErrorMessage(err, 'Failed to create identity provider'))
    }
  }

  function handleCancel() {
    navigate('/administration?tab=identity_providers')
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
                <h4 className="text-sm font-semibold text-gray-800 mb-4">Identity Provider Details</h4>
                <div className="space-y-4">
                  <FloatingInput label="Name" value={name} onChange={setName} required />
                  <FloatingTextarea label="Description (Optional)" value={description} onChange={setDescription} rows={2} />
                  <FloatingDropdown
                    label="Identity Provider Type"
                    options={IDP_TYPE_OPTIONS.map((o) => ({ value: o.value, label: o.label }))}
                    value={idpType}
                    onChange={(v) => setIdpType(v as IdpType)}
                    required
                  />
                </div>
              </div>

              {idpType === 'saml_2.0' && (
                <div className="border border-border rounded-lg p-5">
                  <h4 className="text-sm font-semibold text-gray-800 mb-4">SAML 2.0 Configuration</h4>
                  <div className="space-y-4">
                    <div className="bg-gray-50 rounded-md p-4">
                      <label className="block text-[10px] text-gray-500 mb-0.5">Service Provider Entity ID</label>
                      <div className="text-xs text-gray-700 font-mono">https://acme.dipolehq.com/saml/metadata</div>
                    </div>
                    <div className="bg-gray-50 rounded-md p-4">
                      <label className="block text-[10px] text-gray-500 mb-0.5">Service Provider Single Sign On URL</label>
                      <div className="text-xs text-gray-700 font-mono">https://acme.dipolehq.com/saml/acs</div>
                    </div>
                    <FloatingInput label="Metadata URL" value={metadataUrl} onChange={setMetadataUrl} />
                    <FloatingTextarea label="Metadata XML" value={metadataXml} onChange={setMetadataXml} rows={4} />
                  </div>
                </div>
              )}

              {idpType === 'oidc_1.0' && (
                <div className="border border-border rounded-lg p-5">
                  <h4 className="text-sm font-semibold text-gray-800 mb-4">OIDC 1.0 Configuration</h4>
                  <div className="space-y-4">
                    <FloatingInput label="Identity Provider Base URL" value={baseUrl} onChange={setBaseUrl} required />
                    <FloatingInput label="Client ID" value={clientId} onChange={setClientId} required />
                    <FloatingInput label="Scopes" value={scopes} onChange={setScopes} />
                    <div className="flex items-center gap-2">
                      <input type="checkbox" checked={pkceRequired} onChange={(e) => setPkceRequired(e.target.checked)} className="w-4 h-4 accent-orange-500 cursor-pointer" />
                      <span className="text-sm text-gray-700">PKCE Required</span>
                    </div>
                    <div>
                      <label className="block text-xs text-orange-600 font-medium mb-2">Authentication Method</label>
                      <div className="flex items-center gap-4">
                        <label className="flex items-center gap-1.5 cursor-pointer">
                          <input type="radio" name="authMethod" checked={authMethod === 'client_secret'} onChange={() => setAuthMethod('client_secret')} className="w-3.5 h-3.5 accent-orange-500" />
                          <span className="text-sm text-gray-700">Client Secret</span>
                        </label>
                        <label className="flex items-center gap-1.5 cursor-pointer">
                          <input type="radio" name="authMethod" checked={authMethod === 'public_private_keypair'} onChange={() => setAuthMethod('public_private_keypair')} className="w-3.5 h-3.5 accent-orange-500" />
                          <span className="text-sm text-gray-700">Public Private Keypair</span>
                        </label>
                      </div>
                    </div>
                    {authMethod === 'client_secret' && (
                      <FloatingInput label="Client Secret" value={clientSecret} onChange={setClientSecret} type="password" />
                    )}
                    {authMethod === 'public_private_keypair' && (
                      <div className="bg-gray-50 rounded-md p-4">
                        <label className="block text-[10px] text-gray-500 mb-0.5">JWKS URL (configure in your IdP)</label>
                        <div className="text-xs text-gray-700 font-mono">{import.meta.env.VITE_API_BASE_URL || 'https://api.dipolehq.com'}/.well-known/openid-configuration/jwks</div>
                      </div>
                    )}
                    <div className="bg-gray-50 rounded-md p-4">
                      <label className="block text-[10px] text-gray-500 mb-0.5">Redirect URL (configure in your IdP)</label>
                      <div className="text-xs text-gray-700 font-mono">{import.meta.env.VITE_API_BASE_URL || 'https://api.dipolehq.com'}/api/v1/auth/sso/callback</div>
                    </div>
                  </div>
                </div>
              )}

              <div className="border border-border rounded-lg p-5">
                <h4 className="text-sm font-semibold text-gray-800 mb-4">Role Mappings</h4>
                <p className="text-xs text-gray-600 mb-4">
                  Map your IdP&apos;s group claims to Dipole user roles. When a user authenticates via SSO, Dipole uses these mappings to automatically create user accounts with the correct roles.
                </p>

                {mappings.length > 0 && (
                  <div className="border border-border rounded-lg mb-4">
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
                        {mappings.map((mapping, index) => (
                          <tr key={index}>
                            <td className="px-4 py-2">
                              <input type="text" value={mapping.attributeName} onChange={(e) => handleMappingChange(index, 'attributeName', e.target.value)} placeholder="e.g., groups" className="w-full px-2 py-1 border border-border rounded text-xs focus:outline-none focus:ring-1 focus:ring-orange-300" />
                            </td>
                            <td className="px-4 py-2">
                              <input type="text" value={mapping.attributeValue} onChange={(e) => handleMappingChange(index, 'attributeValue', e.target.value)} placeholder="e.g., DipoleAdmins" className="w-full px-2 py-1 border border-border rounded text-xs focus:outline-none focus:ring-1 focus:ring-orange-300" />
                            </td>
                            <td className="px-4 py-2">
                              <FloatingDropdown label="Role" options={ROLE_OPTIONS} value={mapping.role} onChange={(v) => handleMappingChange(index, 'role', v)} />
                            </td>
                            <td className="px-2 py-2">
                              <button type="button" onClick={() => handleRemoveMapping(index)} className="text-orange-400 hover:text-orange-600 transition-colors">
                                <X size={14} />
                              </button>
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}

                <button type="button" onClick={handleAddMapping} className="flex items-center gap-1.5 text-xs text-orange-500 hover:text-orange-600 font-medium">
                  <Plus size={13} />
                  New Mapping
                </button>
              </div>
            </div>
          </div>

          {/* Help panel — right 40% */}
          <div className="w-[40%] px-6 py-6 overflow-y-auto">
            <div className="border border-border rounded-lg p-5 sticky top-0">
              <p className="text-xs text-gray-700 leading-relaxed mb-3">
                <span className="font-semibold text-orange-600">Identity Providers</span> enable Single Sign-On (SSO) for your organization. Connect your existing IdP (Okta, Google Workspace, Azure AD) to allow users to authenticate using their corporate credentials.
              </p>
              <p className="text-xs text-gray-700 leading-relaxed mb-3">
                <span className="font-semibold">SAML 2.0</span> is the most common enterprise SSO protocol. <span className="font-semibold">OIDC 1.0</span> is preferred for modern applications.
              </p>
              <p className="text-xs text-gray-700 leading-relaxed">
                <span className="font-semibold">Role Mappings</span> allow automatic user provisioning based on IdP group claims.
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

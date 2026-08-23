import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { Pencil, Trash2, Plus, X } from 'lucide-react'
import { useQuery } from '@tanstack/react-query'
import { FloatingInput, FloatingTextarea, FloatingDropdown } from '@/shared/components/FloatingInput'
import { DeleteConfirmDialog } from '@/shared/components/ui/DeleteConfirmDialog'
import { IDP_TYPE_OPTIONS } from '@/shared/constants/administration'
import { samlEntityId, samlAcsUrl, jwksUrl, adminSsoCallbackUrl, currentTenantSubdomain, displayUrl } from '@/shared/lib/platform-urls'
import { useIdentityProvider, useSaveIdentityProvider, useDeleteIdentityProvider } from '@/features/administration/hooks/useIdentityProviders'
import { apiErrorMessage } from '@/features/administration/api/identityProvidersApi'
import { apiClient } from '@/shared/lib/api-client'
import type { IdpType, AuthMethod } from '@/shared/constants/administration'

export function IdentityProviderDetailPage() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()

  const tenantSubdomain = currentTenantSubdomain()

  const { data: idp, isLoading } = useIdentityProvider(id)
  const saveMutation = useSaveIdentityProvider(id ?? '')
  const deleteMutation = useDeleteIdentityProvider()

  const { data: rolesData } = useQuery({
    queryKey: ['roles', 'dropdown'],
    queryFn: () => apiClient.get<{ items: { id: string; name: string }[] }>('/roles?perPage=100'),
    staleTime: 60_000,
  })
  const roleOptions = (rolesData?.items ?? []).map((r) => ({ value: r.name, label: r.name }))

  const [isEditing, setIsEditing] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)

  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [idpStatus, setIdpStatus] = useState('active')
  const [idpType, setIdpType] = useState<IdpType>('saml_2.0')
  const [metadataUrl, setMetadataUrl] = useState('')
  const [baseUrl, setBaseUrl] = useState('')
  const [clientId, setClientId] = useState('')
  const [scopes, setScopes] = useState('openid profile email')
  const [pkceRequired, setPkceRequired] = useState(true)
  const [authMethod, setAuthMethod] = useState<AuthMethod>('client_secret')
  const [clientSecret, setClientSecret] = useState('')
  const [mappings, setMappings] = useState<{ attributeName: string; attributeValue: string; role: string }[]>([])
  const [initialized, setInitialized] = useState(false)

  if (idp && !initialized) {
    const config = (idp.config ?? {}) as Record<string, unknown>
    setName(idp.name)
    setDescription(idp.description)
    setIdpStatus(idp.isActive ? 'active' : 'inactive')
    setIdpType(idp.type as IdpType)
    setMetadataUrl((config.metadataUrl as string) ?? '')
    setBaseUrl((config.baseUrl as string) ?? '')
    setClientId((config.clientId as string) ?? '')
    setScopes((config.scopes as string) ?? 'openid profile email')
    setPkceRequired((config.pkceRequired as boolean) ?? true)
    setAuthMethod((config.authMethod as AuthMethod) ?? 'client_secret')
    setMappings((config.mappings as typeof mappings) ?? [])
    setInitialized(true)
  }

  if (initialized && idp && idp.id !== id) {
    setInitialized(false)
  }

  if (!id || (!isLoading && idp === null)) {
    return (
      <div className="flex flex-col h-full">
        <div className="flex-1 flex items-center justify-center">
          <p className="text-sm text-gray-400">Identity Provider not found.</p>
        </div>
      </div>
    )
  }

  const disabled = !isEditing
  const displayName = name || 'Untitled'
  const statusValue = idpStatus

  function handleAddMapping() { setMappings([...mappings, { attributeName: '', attributeValue: '', role: '' }]) }
  function handleRemoveMapping(index: number) { setMappings(mappings.filter((_, i) => i !== index)) }
  function handleMappingChange(index: number, field: string, value: string) {
    const updated = [...mappings]
    const current = updated[index]
    if (current) { updated[index] = { ...current, [field]: value } }
    setMappings(updated)
  }

  async function handleSave() {
    setError(null)
    const config: Record<string, unknown> = {}
    if (idpType === 'saml_2.0') {
      config.metadataUrl = metadataUrl
    } else if (idpType === 'oidc_1.0') {
      config.baseUrl = baseUrl
      config.clientId = clientId
      config.scopes = scopes
      config.pkceRequired = pkceRequired
      config.authMethod = authMethod
      if (authMethod === 'client_secret' && clientSecret) config.clientSecret = clientSecret
    }
    config.mappings = mappings
    try {
      await saveMutation.mutateAsync({
        name,
        description,
        isActive: idpStatus === 'active',
        type: idpType,
        config,
      })
      setIsEditing(false)
    } catch (err) {
      setError(apiErrorMessage(err, 'Failed to save identity provider'))
    }
  }

  function handleEdit() {
    setError(null)
    setIsEditing(true)
  }

  function handleCancel() {
    setError(null)
    if (idp) {
      const config = (idp.config ?? {}) as Record<string, unknown>
      setName(idp.name)
      setDescription(idp.description)
      setIdpStatus(idp.isActive ? 'active' : 'inactive')
      setIdpType(idp.type as IdpType)
      setMetadataUrl((config.metadataUrl as string) ?? '')
      setBaseUrl((config.baseUrl as string) ?? '')
      setClientId((config.clientId as string) ?? '')
      setScopes((config.scopes as string) ?? 'openid profile email')
      setPkceRequired((config.pkceRequired as boolean) ?? true)
      setAuthMethod((config.authMethod as AuthMethod) ?? 'client_secret')
      setMappings((config.mappings as typeof mappings) ?? [])
    }
    setIsEditing(false)
  }

  async function handleDelete() {
    if (!id) return
    try {
      await deleteMutation.mutateAsync(id)
      navigate('/administration/identity-providers')
    } catch (err) {
      setError(apiErrorMessage(err, 'Failed to delete identity provider'))
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
                  <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider mb-4">Identity Provider Details</h4>
                  <div className="space-y-4">
                    <FloatingInput label="Name" value={name} onChange={setName} disabled={disabled} required />
                    <FloatingTextarea label="Description (Optional)" value={description} onChange={setDescription} disabled={disabled} rows={2} />
                    <FloatingDropdown label="Identity Provider Type" options={IDP_TYPE_OPTIONS.map((o) => ({ value: o.value, label: o.label }))} value={idpType} onChange={(v) => setIdpType(v as IdpType)} disabled={disabled} required />
                  </div>
                </div>

                {idpType === 'saml_2.0' && (
                  <div>
                    <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider mb-4">SAML 2.0 Configuration</h4>
                    <div className="space-y-4">
                      <div className="bg-gray-50 rounded-md p-4">
                        <label className="block text-[10px] text-gray-500 mb-0.5">Service Provider Entity ID</label>
                        <div className="text-xs text-gray-700 font-mono">{displayUrl(samlEntityId(tenantSubdomain))}</div>
                      </div>
                      <div className="bg-gray-50 rounded-md p-4">
                        <label className="block text-[10px] text-gray-500 mb-0.5">Service Provider Single Sign On URL</label>
                        <div className="text-xs text-gray-700 font-mono">{displayUrl(samlAcsUrl(tenantSubdomain))}</div>
                      </div>
                      <FloatingInput label="Metadata URL" value={metadataUrl} onChange={setMetadataUrl} disabled={disabled} />
                    </div>
                  </div>
                )}

                {idpType === 'oidc_1.0' && (
                  <div>
                    <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider mb-4">OIDC 1.0 Configuration</h4>
                    <div className="space-y-4">
                      <FloatingInput label="Identity Provider Base URL" value={baseUrl} onChange={setBaseUrl} disabled={disabled} required />
                      <FloatingInput label="Client ID" value={clientId} onChange={setClientId} disabled={disabled} required />
                      <FloatingInput label="Scopes" value={scopes} onChange={setScopes} disabled={disabled} />
                      <div className="flex items-center gap-2">
                        <input type="checkbox" checked={pkceRequired} onChange={(e) => setPkceRequired(e.target.checked)} disabled={disabled} className="w-4 h-4 accent-gray-600 cursor-pointer disabled:cursor-default disabled:opacity-60" />
                        <span className="text-sm text-gray-700">PKCE Required</span>
                      </div>
                      <div>
                        <label className="block text-xs text-gray-700 font-medium mb-2">Authentication Method</label>
                        <div className="flex items-center gap-4">
                          <label className="flex items-center gap-1.5 cursor-pointer">
                            <input type="radio" name="authMethodDetail" checked={authMethod === 'client_secret'} onChange={() => !disabled && setAuthMethod('client_secret')} disabled={disabled} className="w-3.5 h-3.5 accent-gray-600 disabled:opacity-60" />
                            <span className="text-sm text-gray-700">Client Secret</span>
                          </label>
                          <label className="flex items-center gap-1.5 cursor-pointer">
                            <input type="radio" name="authMethodDetail" checked={authMethod === 'public_private_keypair'} onChange={() => !disabled && setAuthMethod('public_private_keypair')} disabled={disabled} className="w-3.5 h-3.5 accent-gray-600 disabled:opacity-60" />
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
                          <div className="text-xs text-gray-700 font-mono">{displayUrl(jwksUrl(tenantSubdomain))}</div>
                        </div>
                      )}
                      <div className="bg-gray-50 rounded-md p-4">
                        <label className="block text-[10px] text-gray-500 mb-0.5">Redirect URL</label>
                        <div className="text-xs text-gray-700 font-mono">{adminSsoCallbackUrl()}</div>
                      </div>
                    </div>
                  </div>
                )}

                <div>
                  <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider mb-4">Role Mappings</h4>
                  <p className="text-xs text-gray-600 mb-4">Map your IdP&apos;s group claims to Dipole user roles for automatic user creation.</p>
                  {mappings.length > 0 && (
                    <div className="border border-border rounded-lg mb-4">
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
                              <td className="px-4 py-2"><input type="text" value={mapping.attributeName} onChange={(e) => handleMappingChange(index, 'attributeName', e.target.value)} disabled={disabled} placeholder="e.g., groups" className="w-full px-2 py-1 border border-border rounded text-xs focus:outline-none focus:ring-1 focus:ring-gray-300" /></td>
                              <td className="px-4 py-2"><input type="text" value={mapping.attributeValue} onChange={(e) => handleMappingChange(index, 'attributeValue', e.target.value)} disabled={disabled} placeholder="e.g., DipoleAdmins" className="w-full px-2 py-1 border border-border rounded text-xs focus:outline-none focus:ring-1 focus:ring-gray-300" /></td>
                              <td className="px-4 py-2"><FloatingDropdown label="Role" options={roleOptions} value={mapping.role} onChange={(v) => handleMappingChange(index, 'role', v)} disabled={disabled} /></td>
                              <td className="px-2 py-2">{!disabled && <button type="button" onClick={() => handleRemoveMapping(index)} className="text-gray-500 hover:text-gray-700 transition-colors"><X size={14} /></button>}</td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  )}
                  {!disabled && (
                    <button type="button" onClick={handleAddMapping} className="flex items-center gap-1.5 text-xs text-gray-600 hover:text-gray-700 font-medium"><Plus size={13} />New Mapping</button>
                  )}
                </div>
              </div>
            </fieldset>
          </div>

          <div className="px-5 py-5 overflow-y-auto">
            <div className="space-y-4">
              <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider">Metadata</h4>
              {idp && (
                <>
                  <FloatingInput label="ID" value={idp.id} disabled />
                  <FloatingInput label="Type" value={idp.type === 'saml_2.0' ? 'SAML 2.0' : 'OIDC 1.0'} disabled />
                  <FloatingInput label="Created" value={new Date(idp.createdAt).toLocaleString('en-IN', { dateStyle: 'medium', timeStyle: 'short' })} disabled />
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
        resourceType="identity provider"
        loading={deleteMutation.isPending}
      />
    </div>
  )
}

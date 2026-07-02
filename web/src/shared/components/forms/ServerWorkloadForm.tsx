import { Dropdown } from '@/shared/components/Dropdown'
import { TagsField } from '@/shared/components/TagsField'
import { APP_PROTOCOLS, TRANSPORT_PROTOCOLS, AUTH_METHODS } from '@/data/policies/config'
import type { ServerWorkloadFormValues } from '@/shared/components/forms/types'

interface ServerWorkloadFormProps {
  values: ServerWorkloadFormValues
  onChange: (values: ServerWorkloadFormValues) => void
  disabled?: boolean
}

export function ServerWorkloadForm({ values, onChange, disabled = false }: ServerWorkloadFormProps) {
  function update(partial: Partial<ServerWorkloadFormValues>) {
    onChange({ ...values, ...partial })
  }

  function handleProtocolChange(protocol: string) {
    const updates: Partial<ServerWorkloadFormValues> = { appProtocol: protocol }
    switch (protocol) {
      case 'HTTP': Object.assign(updates, { port: '80', forwardToPort: '80', tls: false, forwardTls: false, authMethod: '', authScheme: '', urlPath: '' }); break
      case 'OAuth': Object.assign(updates, { port: '443', forwardToPort: '443', tls: true, forwardTls: true, authMethod: 'oauth_client', authScheme: 'post_body_form', urlPath: '' }); break
      case 'PostgreSQL': Object.assign(updates, { port: '5432', forwardToPort: '5432', tls: false, forwardTls: false, authMethod: '' }); break
      case 'MySQL': Object.assign(updates, { port: '3306', forwardToPort: '3306', tls: false, forwardTls: false, authMethod: '' }); break
      case 'Redis': Object.assign(updates, { port: '6379', forwardToPort: '6379', tls: false, forwardTls: false, authMethod: '' }); break
      case 'Snowflake': Object.assign(updates, { port: '443', forwardToPort: '443', tls: true, forwardTls: true, authMethod: '' }); break
      case 'Redshift': Object.assign(updates, { port: '5439', forwardToPort: '5439', tls: false, forwardTls: false, authMethod: '' }); break
      case 'OracleDatabase': Object.assign(updates, { port: '1521', forwardToPort: '1521', tls: false, forwardTls: false, authMethod: '' }); break
      case 'MCP': Object.assign(updates, { port: '5439', forwardToPort: '', tls: false, forwardTls: false, authMethod: 'http_auth', authScheme: 'bearer', urlPath: '' }); break
    }
    onChange({ ...values, ...updates })
  }

  const showForwardPort = !['MCP'].includes(values.appProtocol)
  const showPortTls = ['HTTP', 'OAuth', 'MCP', 'OracleDatabase', 'Redis', 'Snowflake'].includes(values.appProtocol)
  const showUrlPath = values.appProtocol === 'MCP' || values.appProtocol === 'OAuth'
  const showAuthScheme = values.authMethod !== ''
  const showMcpAuthUrl = values.appProtocol === 'MCP'
  const showStaticHeaders = values.appProtocol === 'HTTP' || values.appProtocol === ''
  const showForwardTlsVerification = ['PostgreSQL', 'MySQL', 'Redis', 'Redshift', 'OracleDatabase'].includes(values.appProtocol)
  const authLocked = ['TCP', 'MCP'].includes(values.appProtocol)

  function getAuthSchemeOptions() {
    switch (values.authMethod) {
      case 'http_auth': return [{ value: 'basic', label: 'Basic' }, { value: 'bearer', label: 'Bearer' }, { value: 'header', label: 'Header' }, { value: 'aws_sigv4', label: 'AWS Signature v4' }]
      case 'api_key': return [{ value: 'header', label: 'Header' }, { value: 'query', label: 'Query Parameter' }]
      case 'snowflake_jwt': return [{ value: 'snowflake_jwt', label: 'Snowflake JWT' }]
      case 'oauth_client': return [{ value: 'post_body_form', label: 'POST Body Form URL Encoded' }]
      case 'mtls_x509': return [{ value: 'x509', label: 'x509 Certificate' }]
      case 'password_auth': return [{ value: 'password', label: 'Password' }]
      default: return []
    }
  }

  function getAuthOptions() {
    switch (values.appProtocol) {
      case 'PostgreSQL': case 'MySQL': case 'Redis': case 'Redshift': case 'OracleDatabase':
        return [{ value: '', label: 'No Authentication' }, { value: 'password_auth', label: 'Password Authentication' }]
      case 'Snowflake': return [{ value: '', label: 'No Authentication' }, { value: 'snowflake_jwt', label: 'JWT Token Authentication' }, { value: 'password_auth', label: 'Password Authentication' }]
      case 'OAuth': return [{ value: '', label: 'No Authentication' }, { value: 'oauth_client', label: 'OAuth Client Authentication' }]
      case 'MCP': return [{ value: '', label: 'No Authentication' }, { value: 'http_auth', label: 'HTTP Authentication' }]
      case 'TCP': return [{ value: '', label: 'No Authentication' }]
      default: return AUTH_METHODS.map((m) => ({ value: m.value, label: m.label }))
    }
  }

  function handleTlsChange(checked: boolean) {
    const u: Partial<ServerWorkloadFormValues> = { tls: checked }
    if (values.appProtocol === 'Snowflake' || values.appProtocol === 'Redis' || values.appProtocol === 'MCP' || values.appProtocol === 'OAuth') { /* port stays */ }
    else if (values.appProtocol === 'OracleDatabase') { u.port = checked ? '2484' : '1521' }
    else { u.port = checked ? '443' : '80' }
    update(u)
  }

  function handleForwardTlsChange(checked: boolean) {
    const u: Partial<ServerWorkloadFormValues> = { forwardTls: checked }
    if (values.appProtocol === 'Snowflake' || values.appProtocol === 'OAuth') { /* stays */ }
    else if (values.appProtocol === 'OracleDatabase') { u.forwardToPort = checked ? '2484' : '1521' }
    else if (!showForwardTlsVerification) { u.forwardToPort = checked ? '443' : '80' }
    update(u)
  }

  function handleAuthMethodChange(v: string) {
    let scheme = ''
    if (v === 'http_auth') scheme = 'basic'
    else if (v === 'api_key') scheme = 'header'
    else if (v === 'mtls_x509') scheme = 'x509'
    else if (v === 'password_auth') scheme = 'password'
    else if (v === 'snowflake_jwt') scheme = 'snowflake_jwt'
    else if (v === 'oauth_client') scheme = 'post_body_form'
    update({ authMethod: v, authScheme: scheme })
  }

  const disabledClass = disabled ? 'opacity-60 pointer-events-none' : ''

  return (
    <fieldset disabled={disabled} className={disabledClass}>
      <div className="space-y-5">
        <div>
          <label className="block text-xs font-medium text-gray-700 mb-1.5">Name</label>
          <input type="text" placeholder="e.g. Stripe API" value={values.name} onChange={(e) => update({ name: e.target.value })} className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-gray-300" />
        </div>
        <div>
          <label className="block text-xs font-medium text-gray-700 mb-1.5">Description</label>
          <textarea placeholder="Description (Optional)" value={values.description} onChange={(e) => update({ description: e.target.value })} rows={2} className="w-full px-3 py-2 border border-border rounded-md text-sm resize-y focus:outline-none focus:ring-1 focus:ring-gray-300" />
        </div>

        <div>
          <h4 className="text-xs font-medium text-gray-800 mb-3">Service Endpoint</h4>
          <div className="space-y-3">
            <div>
              <label className="block text-[11px] text-gray-500 mb-1">Host</label>
              <input type="text" placeholder="e.g. api.stripe.com" value={values.host} onChange={(e) => update({ host: e.target.value })} className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-gray-300" />
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-[11px] text-gray-500 mb-1">Application Protocol</label>
                <Dropdown options={APP_PROTOCOLS.map((p) => ({ value: p.value, label: p.label }))} value={values.appProtocol} onChange={handleProtocolChange} placeholder="Please select one" />
              </div>
              <div>
                <label className="block text-[11px] text-gray-500 mb-1">Transport Protocol</label>
                <Dropdown options={TRANSPORT_PROTOCOLS.map((p) => ({ value: p.value, label: p.label }))} value={values.transportProtocol} onChange={(v) => update({ transportProtocol: v })} placeholder="TCP" />
              </div>
            </div>
            <div className={`grid gap-3 ${showForwardPort ? 'grid-cols-2' : 'grid-cols-1'}`}>
              <div>
                <label className="block text-[11px] text-gray-500 mb-1">Port</label>
                {showPortTls ? (
                  <div className="flex items-center border border-border rounded-md overflow-hidden">
                    <input type="text" placeholder="443" value={values.port} onChange={(e) => update({ port: e.target.value })} className="flex-1 px-3 py-2 text-sm focus:outline-none" />
                    <label className="flex items-center gap-1.5 px-3 border-l border-border cursor-pointer">
                      <input type="checkbox" checked={values.tls} onChange={(e) => handleTlsChange(e.target.checked)} className="w-3.5 h-3.5 rounded border-gray-300" />
                      <span className="text-xs text-gray-600">TLS</span>
                    </label>
                  </div>
                ) : (
                  <input type="text" placeholder="443" value={values.port} onChange={(e) => update({ port: e.target.value })} className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-gray-300" />
                )}
              </div>
              {showForwardPort && (
                <div>
                  <label className="block text-[11px] text-gray-500 mb-1">Forward to Port</label>
                  <div className="flex items-center border border-border rounded-md overflow-hidden">
                    <input type="text" value={values.forwardToPort} onChange={(e) => update({ forwardToPort: e.target.value })} className="flex-1 px-3 py-2 text-sm focus:outline-none" />
                    <label className="flex items-center gap-1.5 px-3 border-l border-border cursor-pointer">
                      <input type="checkbox" checked={values.forwardTls} onChange={(e) => handleForwardTlsChange(e.target.checked)} className="w-3.5 h-3.5 rounded border-gray-300" />
                      <span className="text-xs text-gray-600">TLS</span>
                    </label>
                  </div>
                  {values.forwardTls && showForwardTlsVerification && (
                    <div className="mt-2">
                      <label className="block text-[11px] text-gray-500 mb-1">Forward TLS Verification</label>
                      <Dropdown options={[{ value: 'full', label: 'Full' }, { value: 'none', label: 'None' }]} value={values.forwardTlsVerification} onChange={(v) => update({ forwardTlsVerification: v })} placeholder="Full" />
                    </div>
                  )}
                </div>
              )}
            </div>
            {showUrlPath && (
              <div>
                <label className="block text-[11px] text-gray-500 mb-1">URL Path</label>
                <input type="text" value={values.urlPath} onChange={(e) => update({ urlPath: e.target.value })} className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-gray-300" />
              </div>
            )}
          </div>
        </div>

        <div>
          <h4 className="text-xs font-medium text-gray-800 mb-3">Authentication (Optional)</h4>
          <div className={`grid gap-3 ${showAuthScheme && !authLocked ? 'grid-cols-2' : authLocked && values.appProtocol === 'MCP' ? 'grid-cols-2' : 'grid-cols-1'}`}>
            <div>
              <label className="block text-[11px] text-gray-500 mb-1">Authentication Method</label>
              {authLocked ? (
                <div className="px-3 py-2 border border-border rounded-md text-sm text-gray-500 bg-gray-50">{values.appProtocol === 'MCP' ? 'HTTP Authentication' : (getAuthOptions()[0]?.label ?? 'No Authentication')}</div>
              ) : (
                <Dropdown options={getAuthOptions()} value={values.authMethod} onChange={handleAuthMethodChange} placeholder="No Authentication" />
              )}
            </div>
            {authLocked && values.appProtocol === 'MCP' && (
              <div>
                <label className="block text-[11px] text-gray-500 mb-1">Authentication Scheme</label>
                <div className="px-3 py-2 border border-border rounded-md text-sm text-gray-500 bg-gray-50">Bearer</div>
              </div>
            )}
            {showAuthScheme && !authLocked && (
              <div>
                <label className="block text-[11px] text-gray-500 mb-1">Authentication Scheme</label>
                <Dropdown options={getAuthSchemeOptions()} value={values.authScheme} onChange={(v) => update({ authScheme: v })} placeholder="Select scheme..." />
              </div>
            )}
          </div>
          {values.authMethod === 'api_key' && (
            <div className="mt-3">
              <label className="block text-[11px] text-gray-500 mb-1">{values.authScheme === 'query' ? 'Query Parameter' : 'Header'}</label>
              <input type="text" className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-gray-300" />
            </div>
          )}
          {values.authMethod === 'http_auth' && values.authScheme === 'header' && (
            <div className="mt-3">
              <label className="block text-[11px] text-gray-500 mb-1">Header</label>
              <input type="text" className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-gray-300" />
            </div>
          )}
          {showMcpAuthUrl && (
            <div className="mt-3">
              <label className="block text-[11px] text-gray-500 mb-1">Dipole MCP Authorization Server URL</label>
              <div className="flex items-center border border-border rounded-md overflow-hidden bg-gray-50">
                <input type="text" readOnly value={`https://${values.host || '<tenant>'}.mcp.ap-south-1.dipolehq.com`} className="flex-1 px-3 py-2 text-sm text-gray-500 bg-gray-50 focus:outline-none" />
                <button onClick={() => navigator.clipboard.writeText(`https://${values.host || '<tenant>'}.mcp.ap-south-1.dipolehq.com`)} className="px-3 py-2 border-l border-border text-gray-400 hover:text-gray-600" title="Copy">
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><rect x="9" y="9" width="13" height="13" rx="2" ry="2" strokeWidth="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" strokeWidth="2"/></svg>
                </button>
              </div>
            </div>
          )}
        </div>

        {showStaticHeaders && <StaticHeadersField />}

        <div>
          <label className="block text-xs font-medium text-gray-700 mb-1.5">Tags</label>
          <TagsField />
        </div>
      </div>
    </fieldset>
  )
}

function StaticHeadersField() {
  return (
    <div>
      <h4 className="text-xs font-medium text-gray-800 mb-2">Static HTTP Headers</h4>
      <button className="text-xs text-orange-500 hover:text-orange-600 font-medium">+ Add Header</button>
    </div>
  )
}

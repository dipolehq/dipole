import { useState } from 'react'
import { Dropdown } from '@/shared/components/Dropdown'
import { TagsField } from '@/shared/components/TagsField'
import { APP_PROTOCOLS, TRANSPORT_PROTOCOLS, AUTH_METHODS } from '@/features/policies/config'

export function ServerWorkloadForm() {
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [host, setHost] = useState('')
  const [appProtocol, setAppProtocol] = useState('')
  const [transportProtocol, setTransportProtocol] = useState('TCP')
  const [port, setPort] = useState('')
  const [forwardToPort, setForwardToPort] = useState('')
  const [tls, setTls] = useState(false)
  const [forwardTls, setForwardTls] = useState(false)
  const [forwardTlsVerification, setForwardTlsVerification] = useState('full')
  const [authMethod, setAuthMethod] = useState('')
  const [authScheme, setAuthScheme] = useState('')
  const [urlPath, setUrlPath] = useState('')

  function handleProtocolChange(protocol: string) {
    setAppProtocol(protocol)
    switch (protocol) {
      case 'HTTP': setPort('80'); setForwardToPort('80'); setTls(false); setForwardTls(false); setAuthMethod(''); setAuthScheme(''); setUrlPath(''); break
      case 'OAuth': setPort('443'); setForwardToPort('443'); setTls(true); setForwardTls(true); setAuthMethod('oauth_client'); setAuthScheme('post_body_form'); setUrlPath(''); break
      case 'PostgreSQL': setPort('5432'); setForwardToPort('5432'); setTls(false); setForwardTls(false); setAuthMethod(''); break
      case 'MySQL': setPort('3306'); setForwardToPort('3306'); setTls(false); setForwardTls(false); setAuthMethod(''); break
      case 'Redis': setPort('6379'); setForwardToPort('6379'); setTls(false); setForwardTls(false); setAuthMethod(''); break
      case 'Snowflake': setPort('443'); setForwardToPort('443'); setTls(true); setForwardTls(true); setAuthMethod(''); break
      case 'Redshift': setPort('5439'); setForwardToPort('5439'); setTls(false); setForwardTls(false); setAuthMethod(''); break
      case 'OracleDatabase': setPort('1521'); setForwardToPort('1521'); setTls(false); setForwardTls(false); setAuthMethod(''); break
      case 'MCP': setPort('5439'); setForwardToPort(''); setTls(false); setForwardTls(false); setAuthMethod('http_auth'); setAuthScheme('bearer'); setUrlPath(''); break
      case 'TCP': setPort('80'); setForwardToPort('80'); setTls(false); setForwardTls(false); setAuthMethod(''); break
    }
  }

  const showForwardPort = !['MCP'].includes(appProtocol)
  const showPortTls = ['HTTP', 'OAuth', 'MCP', 'OracleDatabase', 'Redis', 'Snowflake'].includes(appProtocol)
  const showUrlPath = appProtocol === 'MCP' || appProtocol === 'OAuth'
  const showAuthScheme = authMethod !== ''
  const showMcpAuthUrl = appProtocol === 'MCP'
  const showStaticHeaders = appProtocol === 'HTTP' || appProtocol === ''
  const showForwardTlsVerification = ['PostgreSQL', 'MySQL', 'Redis', 'Redshift', 'OracleDatabase'].includes(appProtocol)
  const authLocked = ['TCP', 'MCP'].includes(appProtocol)

  function getAuthSchemeOptions() {
    switch (authMethod) {
      case 'http_auth':
        return [{ value: 'basic', label: 'Basic' }, { value: 'bearer', label: 'Bearer' }, { value: 'header', label: 'Header' }, { value: 'aws_sigv4', label: 'AWS Signature v4' }]
      case 'api_key':
        return [{ value: 'header', label: 'Header' }, { value: 'query', label: 'Query Parameter' }]
      case 'snowflake_jwt':
        return [{ value: 'snowflake_jwt', label: 'Snowflake JWT' }]
      case 'oauth_client':
        return [{ value: 'post_body_form', label: 'POST Body Form URL Encoded' }]
      case 'mtls_x509':
        return [{ value: 'x509', label: 'x509 Certificate' }]
      case 'password_auth':
        return [{ value: 'password', label: 'Password' }]
      default:
        return []
    }
  }

  function getAuthOptions() {
    switch (appProtocol) {
      case 'PostgreSQL':
      case 'MySQL':
      case 'Redis':
      case 'Redshift':
      case 'OracleDatabase':
        return [{ value: '', label: 'No Authentication' }, { value: 'password_auth', label: 'Password Authentication' }]
      case 'Snowflake':
        return [{ value: '', label: 'No Authentication' }, { value: 'snowflake_jwt', label: 'JWT Token Authentication' }, { value: 'password_auth', label: 'Password Authentication' }]
      case 'OAuth':
        return [{ value: '', label: 'No Authentication' }, { value: 'oauth_client', label: 'OAuth Client Authentication' }]
      case 'MCP':
        return [{ value: '', label: 'No Authentication' }, { value: 'http_auth', label: 'HTTP Authentication' }]
      case 'TCP':
        return [{ value: '', label: 'No Authentication' }]
      default:
        return AUTH_METHODS.map((m) => ({ value: m.value, label: m.label }))
    }
  }

  return (
    <div className="space-y-5">
      <div>
        <label className="block text-xs font-medium text-gray-700 mb-1.5">Name</label>
        <input type="text" placeholder="e.g. Stripe API" value={name} onChange={(e) => setName(e.target.value)} className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-gray-300" />
      </div>

      <div>
        <label className="block text-xs font-medium text-gray-700 mb-1.5">Description</label>
        <textarea placeholder="Description (Optional)" value={description} onChange={(e) => setDescription(e.target.value)} rows={2} className="w-full px-3 py-2 border border-border rounded-md text-sm resize-y focus:outline-none focus:ring-1 focus:ring-gray-300" />
      </div>

      <div>
        <h4 className="text-xs font-medium text-gray-800 mb-3">Service Endpoint</h4>
        <div className="space-y-3">
          <div>
            <label className="block text-[11px] text-gray-500 mb-1">Host</label>
            <input type="text" placeholder="e.g. api.stripe.com" value={host} onChange={(e) => setHost(e.target.value)} className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-gray-300" />
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-[11px] text-gray-500 mb-1">Application Protocol</label>
              <Dropdown options={APP_PROTOCOLS.map((p) => ({ value: p.value, label: p.label }))} value={appProtocol} onChange={handleProtocolChange} placeholder="Please select one" />
            </div>
            <div>
              <label className="block text-[11px] text-gray-500 mb-1">Transport Protocol</label>
              <Dropdown options={TRANSPORT_PROTOCOLS.map((p) => ({ value: p.value, label: p.label }))} value={transportProtocol} onChange={setTransportProtocol} placeholder="TCP" />
            </div>
          </div>

          <div className={`grid gap-3 ${showForwardPort ? 'grid-cols-2' : 'grid-cols-1'}`}>
            <div>
              <label className="block text-[11px] text-gray-500 mb-1">Port</label>
              {showPortTls ? (
                <div className="flex items-center border border-border rounded-md overflow-hidden">
                  <input type="text" placeholder="443" value={port} onChange={(e) => setPort(e.target.value)} className="flex-1 px-3 py-2 text-sm focus:outline-none" />
                  <label className="flex items-center gap-1.5 px-3 border-l border-border cursor-pointer">
                    <input type="checkbox" checked={tls} onChange={(e) => { setTls(e.target.checked); if (appProtocol === 'Snowflake' || appProtocol === 'Redis' || appProtocol === 'MCP' || appProtocol === 'OAuth') { /* port stays same */ } else if (appProtocol === 'OracleDatabase') { setPort(e.target.checked ? '2484' : '1521'); } else { if (e.target.checked) setPort('443'); else setPort('80'); } }} className="w-3.5 h-3.5 rounded border-gray-300" />
                    <span className="text-xs text-gray-600">TLS</span>
                  </label>
                </div>
              ) : (
                <input type="text" placeholder="443" value={port} onChange={(e) => setPort(e.target.value)} className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-gray-300" />
              )}
            </div>
            {showForwardPort && (
              <div>
                <label className="block text-[11px] text-gray-500 mb-1">Forward to Port</label>
                <div className="flex items-center border border-border rounded-md overflow-hidden">
                  <input type="text" placeholder="" value={forwardToPort} onChange={(e) => setForwardToPort(e.target.value)} className="flex-1 px-3 py-2 text-sm focus:outline-none" />
                  <label className="flex items-center gap-1.5 px-3 border-l border-border cursor-pointer">
                    <input type="checkbox" checked={forwardTls} onChange={(e) => {
                      setForwardTls(e.target.checked)
                      if (appProtocol === 'Snowflake') {
                        /* port stays 443 */
                      } else if (appProtocol === 'OAuth') {
                        /* port stays 443 */
                      } else if (appProtocol === 'OracleDatabase') {
                        setForwardToPort(e.target.checked ? '2484' : '1521')
                      } else if (!showForwardTlsVerification) {
                        if (e.target.checked) setForwardToPort('443')
                        else setForwardToPort('80')
                      }
                    }} className="w-3.5 h-3.5 rounded border-gray-300" />
                    <span className="text-xs text-gray-600">TLS</span>
                  </label>
                </div>
                {forwardTls && showForwardTlsVerification && (
                  <div className="mt-2">
                    <label className="block text-[11px] text-gray-500 mb-1">Forward TLS Verification</label>
                    <Dropdown
                      options={[{ value: 'full', label: 'Full' }, { value: 'none', label: 'None' }]}
                      value={forwardTlsVerification}
                      onChange={setForwardTlsVerification}
                      placeholder="Full"
                    />
                  </div>
                )}
              </div>
            )}
          </div>

          {showUrlPath && (
            <div>
              <label className="block text-[11px] text-gray-500 mb-1">URL Path</label>
              <input type="text" placeholder="" value={urlPath} onChange={(e) => setUrlPath(e.target.value)} className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-gray-300" />
            </div>
          )}
        </div>
      </div>

      <div>
        <h4 className="text-xs font-medium text-gray-800 mb-3">Authentication (Optional)</h4>
        <div className={`grid gap-3 ${showAuthScheme && !authLocked ? 'grid-cols-2' : authLocked && appProtocol === 'MCP' ? 'grid-cols-2' : 'grid-cols-1'}`}>
          <div>
            <label className="block text-[11px] text-gray-500 mb-1">Authentication Method</label>
            {authLocked ? (
              <div className="px-3 py-2 border border-border rounded-md text-sm text-gray-500 bg-gray-50">
                {appProtocol === 'MCP' ? 'HTTP Authentication' : (getAuthOptions()[0]?.label ?? 'No Authentication')}
              </div>
            ) : (
              <Dropdown options={getAuthOptions()} value={authMethod} onChange={(v) => { setAuthMethod(v); if (v === 'http_auth') setAuthScheme('basic'); else if (v === 'api_key') setAuthScheme('header'); else if (v === 'mtls_x509') setAuthScheme('x509'); else if (v === 'password_auth') setAuthScheme('password'); else if (v === 'snowflake_jwt') setAuthScheme('snowflake_jwt'); else if (v === 'oauth_client') setAuthScheme('post_body_form'); else setAuthScheme(''); }} placeholder="No Authentication" />
            )}
          </div>
          {authLocked && appProtocol === 'MCP' && (
            <div>
              <label className="block text-[11px] text-gray-500 mb-1">Authentication Scheme</label>
              <div className="px-3 py-2 border border-border rounded-md text-sm text-gray-500 bg-gray-50">Bearer</div>
            </div>
          )}
          {showAuthScheme && !authLocked && (
            <div>
              <label className="block text-[11px] text-gray-500 mb-1">Authentication Scheme</label>
              <Dropdown options={getAuthSchemeOptions()} value={authScheme} onChange={setAuthScheme} placeholder="Select scheme..." />
            </div>
          )}
        </div>

        {authMethod === 'api_key' && (
          <div className="mt-3">
            <label className="block text-[11px] text-gray-500 mb-1">{authScheme === 'query' ? 'Query Parameter' : 'Header'}</label>
            <input type="text" placeholder="" className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-gray-300" />
          </div>
        )}

        {authMethod === 'http_auth' && authScheme === 'header' && (
          <div className="mt-3">
            <label className="block text-[11px] text-gray-500 mb-1">Header</label>
            <input type="text" placeholder="" className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-gray-300" />
          </div>
        )}

        {showMcpAuthUrl && (
          <div className="mt-3">
            <label className="block text-[11px] text-gray-500 mb-1">Dipole MCP Authorization Server URL</label>
            <div className="flex items-center border border-border rounded-md overflow-hidden bg-gray-50">
              <input type="text" readOnly value={`https://${host || '<tenant>'}.mcp.ap-south-1.dipolehq.com`} className="flex-1 px-3 py-2 text-sm text-gray-500 bg-gray-50 focus:outline-none" />
              <button
                onClick={() => navigator.clipboard.writeText(`https://${host || '<tenant>'}.mcp.ap-south-1.dipolehq.com`)}
                className="px-3 py-2 border-l border-border text-gray-400 hover:text-gray-600"
                title="Copy to clipboard"
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><rect x="9" y="9" width="13" height="13" rx="2" ry="2" strokeWidth="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" strokeWidth="2"/></svg>
              </button>
            </div>
          </div>
        )}
      </div>

      {showStaticHeaders && (
        <StaticHeadersField />
      )}

      <div>
        <label className="block text-xs font-medium text-gray-700 mb-1.5">Tags</label>
        <TagsField />
      </div>
    </div>
  )
}

function StaticHeadersField() {
  const [headers, setHeaders] = useState<{ key: string; value: string }[]>([])

  function addHeader() {
    setHeaders([...headers, { key: '', value: '' }])
  }

  function updateHeader(idx: number, field: 'key' | 'value', val: string) {
    setHeaders(headers.map((h, i) => i === idx ? { ...h, [field]: val } : h))
  }

  function removeHeader(idx: number) {
    setHeaders(headers.filter((_, i) => i !== idx))
  }

  return (
    <div>
      <h4 className="text-xs font-medium text-gray-800 mb-2">Static HTTP Headers</h4>
      {headers.length > 0 && (
        <div className="space-y-2 mb-3">
          {headers.map((header, idx) => (
            <div key={idx} className="flex items-center gap-2">
              <input
                type="text"
                placeholder="Header name"
                value={header.key}
                onChange={(e) => updateHeader(idx, 'key', e.target.value)}
                className="flex-1 px-3 py-2 border border-border rounded-md text-xs focus:outline-none focus:ring-1 focus:ring-gray-300"
              />
              <input
                type="text"
                placeholder="Value"
                value={header.value}
                onChange={(e) => updateHeader(idx, 'value', e.target.value)}
                className="flex-1 px-3 py-2 border border-border rounded-md text-xs focus:outline-none focus:ring-1 focus:ring-gray-300"
              />
              <button onClick={() => removeHeader(idx)} className="text-gray-400 hover:text-red-500 text-sm">×</button>
            </div>
          ))}
        </div>
      )}
      <button onClick={addHeader} className="text-xs text-orange-500 hover:text-orange-600 font-medium">
        + Add Header
      </button>
    </div>
  )
}

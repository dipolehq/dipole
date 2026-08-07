import { type ReactNode } from 'react'
import { mcpAuthServerUrl } from '@/shared/lib/platform-urls'
import { TagsField } from '@/shared/components/TagsField'
import {
  FloatingInput,
  FloatingTextarea,
  FloatingDropdown,
} from '@/shared/components/FloatingInput'
import { APP_PROTOCOLS, TRANSPORT_PROTOCOLS, AUTH_METHODS } from '@/shared/constants/policies'
import type { ServerWorkloadFormValues, FormHeader } from '@/shared/components/forms/types'

interface ServerWorkloadFormProps {
  values: ServerWorkloadFormValues
  onChange: (values: ServerWorkloadFormValues) => void
  disabled?: boolean
  statusSlot?: ReactNode
}

export function ServerWorkloadForm({
  values,
  onChange,
  disabled = false,
  statusSlot,
}: ServerWorkloadFormProps) {
  function update(partial: Partial<ServerWorkloadFormValues>) {
    onChange({ ...values, ...partial })
  }

  function handleProtocolChange(protocol: string) {
    const updates: Partial<ServerWorkloadFormValues> = { appProtocol: protocol }
    switch (protocol) {
      case 'HTTP':
        Object.assign(updates, {
          port: '80',
          forwardToPort: '80',
          tls: false,
          forwardTls: false,
          authMethod: '',
          authScheme: '',
          urlPath: '',
        })
        break
      case 'OAuth':
        Object.assign(updates, {
          port: '443',
          forwardToPort: '443',
          tls: true,
          forwardTls: true,
          authMethod: 'oauth_client',
          authScheme: 'post_body_form',
          urlPath: '',
        })
        break
      case 'PostgreSQL':
        Object.assign(updates, {
          port: '5432',
          forwardToPort: '5432',
          tls: false,
          forwardTls: false,
          authMethod: '',
        })
        break
      case 'MySQL':
        Object.assign(updates, {
          port: '3306',
          forwardToPort: '3306',
          tls: false,
          forwardTls: false,
          authMethod: '',
        })
        break
      case 'Redis':
        Object.assign(updates, {
          port: '6379',
          forwardToPort: '6379',
          tls: false,
          forwardTls: false,
          authMethod: '',
        })
        break
      case 'Snowflake':
        Object.assign(updates, {
          port: '443',
          forwardToPort: '443',
          tls: true,
          forwardTls: true,
          authMethod: '',
        })
        break
      case 'Redshift':
        Object.assign(updates, {
          port: '5439',
          forwardToPort: '5439',
          tls: false,
          forwardTls: false,
          authMethod: '',
        })
        break
      case 'OracleDatabase':
        Object.assign(updates, {
          port: '1521',
          forwardToPort: '1521',
          tls: false,
          forwardTls: false,
          authMethod: '',
        })
        break
      case 'MCP':
        Object.assign(updates, {
          port: '5439',
          forwardToPort: '',
          tls: false,
          forwardTls: false,
          authMethod: 'http_auth',
          authScheme: 'bearer',
          urlPath: '',
        })
        break
    }
    onChange({ ...values, ...updates })
  }

  const showForwardPort = !['MCP'].includes(values.appProtocol)
  const showPortTls = ['HTTP', 'OAuth', 'MCP', 'OracleDatabase', 'Redis', 'Snowflake'].includes(
    values.appProtocol,
  )
  const showUrlPath = values.appProtocol === 'MCP' || values.appProtocol === 'OAuth'
  const showAuthScheme = values.authMethod !== '' && values.authMethod !== 'no_auth'
  const showMcpAuthUrl = values.appProtocol === 'MCP'
  const showStaticHeaders = values.appProtocol === 'HTTP' || values.appProtocol === ''
  const showForwardTlsVerification = [
    'PostgreSQL',
    'MySQL',
    'Redis',
    'Redshift',
    'OracleDatabase',
  ].includes(values.appProtocol)
  const authLocked = ['TCP', 'MCP'].includes(values.appProtocol)

  function getAuthSchemeOptions() {
    switch (values.authMethod) {
      case 'http_auth':
        return [
          { value: 'basic', label: 'Basic' },
          { value: 'bearer', label: 'Bearer' },
          { value: 'header', label: 'Header' },
          { value: 'aws_sigv4', label: 'AWS Signature v4' },
        ]
      case 'api_key':
        return [
          { value: 'header', label: 'Header' },
          { value: 'query', label: 'Query Parameter' },
        ]
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
    switch (values.appProtocol) {
      case 'PostgreSQL':
      case 'MySQL':
      case 'Redis':
      case 'Redshift':
      case 'OracleDatabase':
        return [{ value: 'password_auth', label: 'Password Authentication' }]
      case 'Snowflake':
        return [
          { value: 'snowflake_jwt', label: 'JWT Token Authentication' },
          { value: 'password_auth', label: 'Password Authentication' },
        ]
      case 'OAuth':
        return [{ value: 'oauth_client', label: 'OAuth Client Authentication' }]
      case 'MCP':
        return [{ value: 'http_auth', label: 'HTTP Authentication' }]
      case 'TCP':
        return []
      default:
        return AUTH_METHODS.map((m) => ({ value: m.value, label: m.label }))
    }
  }

  function handleTlsChange(checked: boolean) {
    const u: Partial<ServerWorkloadFormValues> = { tls: checked }
    if (['Snowflake', 'Redis', 'MCP', 'OAuth'].includes(values.appProtocol)) {
      /* port stays */
    } else if (values.appProtocol === 'OracleDatabase') {
      u.port = checked ? '2484' : '1521'
    } else {
      u.port = checked ? '443' : '80'
    }
    update(u)
  }

  function handleForwardTlsChange(checked: boolean) {
    const u: Partial<ServerWorkloadFormValues> = { forwardTls: checked }
    if (['Snowflake', 'OAuth'].includes(values.appProtocol)) {
      /* stays */
    } else if (values.appProtocol === 'OracleDatabase') {
      u.forwardToPort = checked ? '2484' : '1521'
    } else if (!showForwardTlsVerification) {
      u.forwardToPort = checked ? '443' : '80'
    }
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
        {/* Details */}
        <div className="border border-border rounded-lg p-5">
          <h4 className="text-sm font-semibold text-gray-800 mb-4">Details</h4>
          <div className="space-y-4">
            {statusSlot}
            <FloatingInput
              label="Name"
              value={values.name}
              onChange={(v) => update({ name: v })}
              disabled={disabled}
              required
            />
            <FloatingTextarea
              label="Description (Optional)"
              value={values.description}
              onChange={(v) => update({ description: v })}
              disabled={disabled}
              rows={2}
            />
          </div>
        </div>

        {/* Service Endpoint */}
        <div className="border border-border rounded-lg p-5">
          <h4 className="text-sm font-semibold text-gray-800 mb-4">Service Endpoint</h4>
          <div className="space-y-4">
            <FloatingInput
              label="Host"
              value={values.host}
              onChange={(v) => update({ host: v })}
              disabled={disabled}
            />
            <div className="grid grid-cols-2 gap-3">
              <FloatingDropdown
                label="Application Protocol"
                options={APP_PROTOCOLS.map((p) => ({ value: p.value, label: p.label }))}
                value={values.appProtocol}
                onChange={handleProtocolChange}
                disabled={disabled}
              />
              <FloatingDropdown
                label="Transport Protocol"
                options={TRANSPORT_PROTOCOLS.map((p) => ({ value: p.value, label: p.label }))}
                value={values.transportProtocol}
                onChange={(v) => update({ transportProtocol: v })}
                disabled={disabled}
              />
            </div>
            <div className={`grid gap-3 ${showForwardPort ? 'grid-cols-2' : 'grid-cols-1'}`}>
              <div>
                {showPortTls ? (
                  <div className="flex items-center border border-border rounded-md overflow-hidden h-[44px]">
                    <input
                      type="text"
                      placeholder="443"
                      value={values.port}
                      onChange={(e) => update({ port: e.target.value })}
                      className="flex-1 px-3 py-2 text-sm focus:outline-none"
                    />
                    <label className="flex items-center gap-1.5 px-3 border-l border-border cursor-pointer">
                      <input
                        type="checkbox"
                        checked={values.tls}
                        onChange={(e) => handleTlsChange(e.target.checked)}
                        className="w-3.5 h-3.5 rounded border-gray-300"
                      />
                      <span className="text-xs text-gray-600">TLS</span>
                    </label>
                  </div>
                ) : (
                  <FloatingInput
                    label="Port"
                    value={values.port}
                    onChange={(v) => update({ port: v })}
                    disabled={disabled}
                  />
                )}
              </div>
              {showForwardPort && (
                <div>
                  <div className="flex items-center border border-border rounded-md overflow-hidden h-[44px]">
                    <input
                      type="text"
                      value={values.forwardToPort}
                      onChange={(e) => update({ forwardToPort: e.target.value })}
                      className="flex-1 px-3 py-2 text-sm focus:outline-none"
                      placeholder="Forward to Port"
                    />
                    <label className="flex items-center gap-1.5 px-3 border-l border-border cursor-pointer">
                      <input
                        type="checkbox"
                        checked={values.forwardTls}
                        onChange={(e) => handleForwardTlsChange(e.target.checked)}
                        className="w-3.5 h-3.5 rounded border-gray-300"
                      />
                      <span className="text-xs text-gray-600">TLS</span>
                    </label>
                  </div>
                  {values.forwardTls && showForwardTlsVerification && (
                    <div className="mt-3">
                      <FloatingDropdown
                        label="Forward TLS Verification"
                        options={[
                          { value: 'full', label: 'Full' },
                          { value: 'none', label: 'None' },
                        ]}
                        value={values.forwardTlsVerification}
                        onChange={(v) => update({ forwardTlsVerification: v })}
                        disabled={disabled}
                      />
                    </div>
                  )}
                </div>
              )}
            </div>
            {showUrlPath && (
              <FloatingInput
                label="URL Path"
                value={values.urlPath}
                onChange={(v) => update({ urlPath: v })}
                disabled={disabled}
              />
            )}
          </div>
        </div>

        {/* Authentication */}
        <div className="border border-border rounded-lg p-5">
          <h4 className="text-sm font-semibold text-gray-800 mb-4">Authentication</h4>
          <div className="space-y-4">
            <div
              className={`grid gap-3 ${showAuthScheme && !authLocked ? 'grid-cols-2' : authLocked && values.appProtocol === 'MCP' ? 'grid-cols-2' : 'grid-cols-1'}`}
            >
              <div>
                {authLocked ? (
                  <FloatingInput
                    label="Authentication Method"
                    value={values.appProtocol === 'MCP' ? 'HTTP Authentication' : ''}
                    disabled
                  />
                ) : (
                  <FloatingDropdown
                    label="Authentication Method"
                    options={getAuthOptions()}
                    value={values.authMethod}
                    onChange={handleAuthMethodChange}
                    disabled={disabled}
                  />
                )}
              </div>
              {authLocked && values.appProtocol === 'MCP' && (
                <FloatingInput label="Authentication Scheme" value="Bearer" disabled />
              )}
              {showAuthScheme && !authLocked && (
                <FloatingDropdown
                  label="Authentication Scheme"
                  options={getAuthSchemeOptions()}
                  value={values.authScheme}
                  onChange={(v) => update({ authScheme: v })}
                  disabled={disabled}
                />
              )}
            </div>
            {values.authMethod === 'api_key' && (
              <FloatingInput
                label={values.authScheme === 'query' ? 'Query Parameter' : 'Header Name'}
                value={values.headerName}
                onChange={(v) => update({ headerName: v })}
                disabled={disabled}
              />
            )}
            {values.authMethod === 'http_auth' && values.authScheme === 'header' && (
              <FloatingInput
                label="Header Name"
                value={values.headerName}
                onChange={(v) => update({ headerName: v })}
                disabled={disabled}
              />
            )}
            {showMcpAuthUrl && (
              <FloatingInput
                label="Dipole MCP Authorization Server URL"
                value={mcpAuthServerUrl(values.host || '<tenant>')}
                disabled
              />
            )}
          </div>
        </div>

        {/* Static Headers */}
        {showStaticHeaders && (
          <StaticHeadersSection
            value={values.staticHeaders}
            onChange={(staticHeaders) => update({ staticHeaders })}
            disabled={disabled}
          />
        )}

        {/* Tags */}
        <div className="border border-border rounded-lg p-5">
          <h4 className="text-sm font-semibold text-gray-800 mb-4">Tags</h4>
          <TagsField
            value={values.tags}
            onChange={(tags) => update({ tags })}
            disabled={disabled}
          />
        </div>
      </div>
    </fieldset>
  )
}

interface StaticHeadersSectionProps {
  value: FormHeader[]
  onChange: (headers: FormHeader[]) => void
  disabled?: boolean
}

function StaticHeadersSection({ value, onChange, disabled = false }: StaticHeadersSectionProps) {
  function addHeader() {
    onChange([...value, { name: '', value: '' }])
  }

  function updateHeader(idx: number, field: 'name' | 'value', val: string) {
    onChange(value.map((h, i) => (i === idx ? { ...h, [field]: val } : h)))
  }

  function removeHeader(idx: number) {
    onChange(value.filter((_, i) => i !== idx))
  }

  return (
    <div className="border border-border rounded-lg p-5">
      <h4 className="text-sm font-semibold text-gray-800 mb-4">Static HTTP Headers</h4>
      {value.length > 0 && (
        <div className="space-y-3 mb-3">
          {value.map((header, idx) => (
            <div key={idx} className="flex items-center gap-2">
              <div className="flex-1">
                <FloatingInput
                  label="Header Name"
                  value={header.name}
                  onChange={(v) => updateHeader(idx, 'name', v)}
                  disabled={disabled}
                />
              </div>
              <div className="flex-1">
                <FloatingInput
                  label="Header Value"
                  value={header.value}
                  onChange={(v) => updateHeader(idx, 'value', v)}
                  disabled={disabled}
                />
              </div>
              <button
                type="button"
                onClick={() => removeHeader(idx)}
                className="text-orange-400 hover:text-red-500 text-sm"
              >
                ×
              </button>
            </div>
          ))}
        </div>
      )}
      <button
        type="button"
        onClick={addHeader}
        className="text-xs text-orange-500 hover:text-orange-600 font-medium"
      >
        + Add Header
      </button>
    </div>
  )
}

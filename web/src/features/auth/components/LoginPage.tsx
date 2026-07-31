import { useState, useEffect } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { apiClient } from '@/shared/lib/api-client'

interface SSOProvider {
  id: string
  name: string
  type: string
}

export function LoginPage() {
  const navigate = useNavigate()
  const [ssoProviders, setSsoProviders] = useState<SSOProvider[]>([])
  const [ssoLoading, setSsoLoading] = useState(false)
  const [showForm, setShowForm] = useState(false)
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const [mfaToken, setMfaToken] = useState('')
  const [mfaCode, setMfaCode] = useState('')

  const [tenants, setTenants] = useState<{ id: string; name: string; subdomain: string }[]>([])

  useEffect(() => {
    apiClient.get<{ providers: SSOProvider[] }>('/auth/sso/providers')
      .then((res) => setSsoProviders(res.providers ?? []))
      .catch(() => {}) // Silently fail — SSO button just won't show.
  }, [])

  async function handleSSOLogin(providerId: string) {
    setSsoLoading(true)
    setError('')
    try {
      const callbackUrl = window.location.origin + '/auth/sso/callback'
      const res = await apiClient.get<{ redirectUrl: string; state: string }>(
        `/auth/sso/authorize?provider_id=${providerId}&callback_url=${encodeURIComponent(callbackUrl)}`,
      )
      sessionStorage.setItem('dipole_sso_state', res.state)
      sessionStorage.setItem('dipole_sso_provider', providerId)
      window.location.href = res.redirectUrl
    } catch {
      setError('Failed to initiate SSO login')
      setSsoLoading(false)
    }
  }
  async function handleSubmit(e: React.FormEvent, tenantId?: string) {
    e.preventDefault()
    setError('')
    setLoading(true)

    try {
      const res = await apiClient.post<{ token?: string; mfaRequired?: boolean; mfaToken?: string; user?: { id: string; email: string; name: string }; tenants?: { id: string; name: string; subdomain: string }[] }>(
        '/auth/login',
        { email, password, tenantId },
      )

      if (res.tenants && res.tenants.length > 1) {
        setTenants(res.tenants)
      } else if (res.mfaRequired && res.mfaToken) {
        setMfaToken(res.mfaToken)
      } else if (res.token && res.user) {
        sessionStorage.setItem('dipole_token', res.token)
        sessionStorage.setItem('dipole_user', JSON.stringify(res.user))
        navigate('/')
      }
    } catch {
      setError('Invalid email or password')
    } finally {
      setLoading(false)
    }
  }

  async function handleTenantSelect(tenantId: string) {
    setError('')
    setLoading(true)
    try {
      const res = await apiClient.post<{ token?: string; mfaRequired?: boolean; mfaToken?: string; user?: { id: string; email: string; name: string } }>(
        '/auth/login',
        { email, password, tenantId },
      )

      if (res.mfaRequired && res.mfaToken) {
        setTenants([])
        setMfaToken(res.mfaToken)
      } else if (res.token && res.user) {
        sessionStorage.setItem('dipole_token', res.token)
        sessionStorage.setItem('dipole_user', JSON.stringify(res.user))
        navigate('/')
      }
    } catch {
      setError('Failed to sign in to selected tenant')
    } finally {
      setLoading(false)
    }
  }

  async function handleMFAVerify(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setLoading(true)

    try {
      const res = await apiClient.post<{ token: string; user: { id: string; email: string; name: string } }>(
        '/auth/mfa-verify',
        { mfaToken, code: mfaCode },
      )
      sessionStorage.setItem('dipole_token', res.token)
      sessionStorage.setItem('dipole_user', JSON.stringify(res.user))
      navigate('/')
    } catch {
      setError('Invalid verification code')
    } finally {
      setLoading(false)
    }
  }

  if (!showForm) {
    return (
      <div className="h-screen w-screen bg-[#050505] overflow-hidden">
        <div className="fixed inset-0 z-10 flex flex-col items-center justify-center pointer-events-none">
          <div className="flex items-center gap-6 mb-6 -mt-16 select-none">
            <img src="/dipole_logo.svg" alt="Dipole" className="h-32 w-32" draggable={false} />
            <span className="text-white text-8xl tracking-tight" style={{ fontFamily: 'Tomorrow, sans-serif', fontWeight: 700 }}>Dipole</span>
          </div>

          <p className="text-white text-lg mb-12 select-none">Identity & Access Management Fabric For Agents & Workloads</p>

          <button
            type="button"
            onClick={() => setShowForm(true)}
            className="px-8 py-3 bg-orange-500 hover:bg-orange-600 text-white text-sm font-medium rounded-md transition-colors pointer-events-auto"
          >
            Sign In with Email
          </button>
        </div>
      </div>
    )
  }

  return (
    <div className="h-screen w-screen bg-[#f5f5f5] flex items-center justify-center">
      <div className="w-[400px] bg-white rounded-xl shadow-lg p-8">
        <div className="flex flex-col items-center mb-8">
          <img src="/dipole_logo.svg" alt="Dipole" className="h-12 w-12 mb-3" />
          <h1 className="text-lg font-semibold text-gray-900">
            {tenants.length > 0 ? 'Select Tenant' : mfaToken ? 'Verification Required' : 'Sign In to Dipole'}
          </h1>
          {tenants.length > 0 && (
            <p className="text-sm text-gray-500 mt-1 text-center">
              Your account has access to multiple tenants. Choose one to continue.
            </p>
          )}
          {mfaToken && (
            <p className="text-sm text-gray-500 mt-1 text-center">
              Enter the 6-digit code from your authenticator app or a backup code.
            </p>
          )}
        </div>

        {tenants.length > 0 ? (
          <div className="space-y-2">
            {tenants.map((t) => (
              <button
                key={t.id}
                type="button"
                onClick={() => handleTenantSelect(t.id)}
                disabled={loading}
                className="w-full text-left px-4 py-3 border border-gray-200 rounded-lg hover:border-orange-400 hover:bg-orange-50 transition-colors disabled:opacity-50"
              >
                <div className="text-sm font-medium text-gray-900">{t.name}</div>
                <div className="text-xs text-gray-500">{t.subdomain}</div>
              </button>
            ))}

            {error && <p className="text-sm text-red-600 text-center mt-2">{error}</p>}

            <button
              type="button"
              onClick={() => { setTenants([]); setError('') }}
              className="block w-full text-center text-sm text-gray-500 hover:text-gray-700 mt-3"
            >
              ← Back to Sign In
            </button>
          </div>
        ) : mfaToken ? (
          <form onSubmit={handleMFAVerify} className="space-y-4">
            <div>
              <input
                type="text"
                placeholder="000000"
                value={mfaCode}
                onChange={(e) => setMfaCode(e.target.value.replace(/[^a-zA-Z0-9\-]/g, '').slice(0, 9))}
                required
                autoFocus
                className="w-full px-4 py-3 border border-gray-300 rounded-md text-sm text-center tracking-[0.3em] font-mono focus:outline-none focus:ring-2 focus:ring-orange-500 focus:border-transparent"
              />
            </div>

            {error && (
              <p className="text-sm text-red-600 text-center">{error}</p>
            )}

            <button
              type="submit"
              disabled={loading || mfaCode.length < 6}
              className="w-full py-3 bg-orange-500 hover:bg-orange-600 text-white text-sm font-medium rounded-md transition-colors disabled:opacity-50"
            >
              {loading ? 'Verifying…' : 'Verify'}
            </button>

            <button
              type="button"
              onClick={() => { setMfaToken(''); setMfaCode(''); setError('') }}
              className="block w-full text-center text-sm text-gray-500 hover:text-gray-700"
            >
              ← Back to Sign In
            </button>
          </form>
        ) : (
          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <input
                type="email"
                placeholder="Email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
                className="w-full px-4 py-3 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-orange-500 focus:border-transparent"
              />
            </div>
            <div>
              <input
                type="password"
                placeholder="Password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
                className="w-full px-4 py-3 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-orange-500 focus:border-transparent"
              />
            </div>

            {error && (
              <p className="text-sm text-red-600 text-center">{error}</p>
            )}

            <button
              type="submit"
              disabled={loading}
              className="w-full py-3 bg-orange-500 hover:bg-orange-600 text-white text-sm font-medium rounded-md transition-colors disabled:opacity-50"
            >
              {loading ? 'Signing in…' : 'Sign In'}
            </button>

            <Link
              to="/forgot-password"
              className="block text-center text-sm text-gray-500 hover:text-orange-600 transition-colors"
            >
              Forgot password?
            </Link>

            {ssoProviders.length > 0 && (
              <>
                <div className="flex items-center gap-3 mt-2">
                  <div className="flex-1 h-px bg-gray-200" />
                  <span className="text-xs text-gray-400">or</span>
                  <div className="flex-1 h-px bg-gray-200" />
                </div>
                {ssoProviders.map((provider) => (
                  <button
                    key={provider.id}
                    type="button"
                    onClick={() => handleSSOLogin(provider.id)}
                    disabled={ssoLoading}
                    className="w-full py-3 border border-gray-300 text-gray-700 text-sm font-medium rounded-md hover:bg-gray-50 transition-colors disabled:opacity-50"
                  >
                    {ssoLoading ? 'Redirecting…' : `Sign in with ${provider.name}`}
                  </button>
                ))}
              </>
            )}
          </form>
        )}
      </div>
    </div>
  )
}

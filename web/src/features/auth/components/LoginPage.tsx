import { useState, useEffect } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { apiClient } from '@/shared/lib/api-client'
import { setCurrentTenantSubdomain } from '@/shared/lib/platform-urls'

interface SSOProvider {
  id: string
  name: string
  type: string
}

export function LoginPage() {
  const navigate = useNavigate()
  const [ssoProviders, setSsoProviders] = useState<SSOProvider[]>([])
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [mfaToken, setMfaToken] = useState('')
  const [mfaCode, setMfaCode] = useState('')
  const [tenants, setTenants] = useState<{ id: string; name: string; subdomain: string }[]>([])

  useEffect(() => {
    document.title = 'Dipole | Sign In'
    apiClient.get<{ providers: SSOProvider[] }>('/auth/sso/providers')
      .then((res) => setSsoProviders(res.providers ?? []))
      .catch(() => {})
  }, [])

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')

    if (!email.trim()) {
      setError('Email is required')
      return
    }
    if (!password) {
      setError('Password is required')
      return
    }

    setLoading(true)
    try {
      const res = await apiClient.post<{
        token?: string
        mfaRequired?: boolean
        mfaToken?: string
        tenantSubdomain?: string
        user?: { id: string; email: string; name: string }
        tenants?: { id: string; name: string; subdomain: string }[]
      }>('/auth/login', { email, password })

      if (res.tenants && res.tenants.length > 1) {
        setTenants(res.tenants)
      } else if (res.mfaRequired && res.mfaToken) {
        setMfaToken(res.mfaToken)
      } else if (res.token && res.user) {
        sessionStorage.setItem('dipole_token', res.token)
        sessionStorage.setItem('dipole_user', JSON.stringify(res.user))
        setCurrentTenantSubdomain(res.tenantSubdomain ?? '')
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
      const res = await apiClient.post<{
        token?: string
        mfaRequired?: boolean
        mfaToken?: string
        tenantSubdomain?: string
        user?: { id: string; email: string; name: string }
      }>('/auth/login', { email, password, tenantId })

      if (res.mfaRequired && res.mfaToken) {
        setTenants([])
        setMfaToken(res.mfaToken)
      } else if (res.token && res.user) {
        sessionStorage.setItem('dipole_token', res.token)
        sessionStorage.setItem('dipole_user', JSON.stringify(res.user))
        setCurrentTenantSubdomain(res.tenantSubdomain ?? '')
        navigate('/')
      }
    } catch {
      setError('Failed to sign in')
    } finally {
      setLoading(false)
    }
  }

  async function handleMFAVerify(e?: React.FormEvent) {
    e?.preventDefault()
    setError('')
    setLoading(true)
    try {
      const res = await apiClient.post<{
        token: string
        tenantSubdomain?: string
        user: { id: string; email: string; name: string }
      }>('/auth/mfa-verify', { mfaToken, code: mfaCode })

      sessionStorage.setItem('dipole_token', res.token)
      sessionStorage.setItem('dipole_user', JSON.stringify(res.user))
      setCurrentTenantSubdomain(res.tenantSubdomain ?? '')
      navigate('/')
    } catch {
      setError('Invalid verification code')
    } finally {
      setLoading(false)
    }
  }

  async function handleSSOLogin(providerId: string) {
    setError('')
    try {
      const callbackUrl = window.location.origin + '/auth/sso/callback'
      const res = await apiClient.get<{ redirectUrl: string; state: string }>(
        `/auth/sso/authorize?provider_id=${providerId}&callback_url=${encodeURIComponent(callbackUrl)}`
      )
      sessionStorage.setItem('dipole_sso_state', res.state)
      sessionStorage.setItem('dipole_sso_provider', providerId)
      window.location.href = res.redirectUrl
    } catch {
      setError('Failed to initiate SSO login')
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-gray-900 via-gray-800 to-black p-4">
      <div className="w-full max-w-md">
        <div className="bg-white rounded-2xl shadow-2xl p-8">
          {/* Logo */}
          <div className="flex justify-center mb-8">
            <img src="/brand_black.svg" alt="Dipole" className="h-8" />
          </div>

          {/* Title */}
          <h1 className="text-2xl font-bold text-gray-900 text-center mb-8">
            {tenants.length > 0 ? 'Select your tenant' : mfaToken ? 'Verification required' : 'Welcome back'}
          </h1>

          {/* Tenant Selection */}
          {tenants.length > 0 ? (
            <div className="space-y-4">
              {tenants.map((t) => (
                <button
                  key={t.id}
                  type="button"
                  onClick={() => handleTenantSelect(t.id)}
                  disabled={loading}
                  className="w-full p-4 text-left border-2 border-gray-200 rounded-xl hover:border-gray-900 hover:bg-gray-50 transition-all disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  <div className="font-semibold text-gray-900">{t.name}</div>
                  <div className="text-sm text-gray-500">{t.subdomain}</div>
                </button>
              ))}
              {error && (
                <div className="p-4 bg-red-50 border border-red-200 rounded-lg text-sm text-red-600">
                  {error}
                </div>
              )}
              <button
                type="button"
                onClick={() => { setTenants([]); setError('') }}
                className="text-sm text-gray-600 hover:text-gray-900 font-medium"
              >
                ← Back
              </button>
            </div>
          ) : mfaToken ? (
            /* MFA Verification */
            <form onSubmit={(e) => handleMFAVerify(e)} className="space-y-4">
              <div>
                <input
                  type="text"
                  placeholder="Enter 6-digit code"
                  value={mfaCode}
                  onChange={(e) => setMfaCode(e.target.value.replace(/[^0-9]/g, '').slice(0, 6))}
                  maxLength={6}
                  className="w-full px-4 py-3 border-2 border-gray-200 rounded-xl focus:border-gray-900 focus:outline-none text-center text-lg tracking-widest font-mono"
                />
              </div>
              {error && (
                <div className="p-4 bg-red-50 border border-red-200 rounded-lg text-sm text-red-600">
                  {error}
                </div>
              )}
              <button
                type="submit"
                disabled={loading || mfaCode.length < 6}
                className="w-full bg-gray-900 text-white py-3 rounded-xl font-semibold hover:bg-gray-800 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {loading ? 'Verifying...' : 'Verify'}
              </button>
              <button
                type="button"
                onClick={() => { setMfaToken(''); setMfaCode(''); setError('') }}
                className="text-sm text-gray-600 hover:text-gray-900 font-medium"
              >
                ← Back
              </button>
            </form>
          ) : (
            /* Login Form */
            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <input
                  type="email"
                  placeholder="Email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  autoComplete="email"
                  className="w-full px-4 py-3 border-2 border-gray-200 rounded-xl focus:border-gray-900 focus:outline-none transition-colors"
                />
              </div>
              <div>
                <input
                  type="password"
                  placeholder="Password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  autoComplete="current-password"
                  className="w-full px-4 py-3 border-2 border-gray-200 rounded-xl focus:border-gray-900 focus:outline-none transition-colors"
                />
              </div>

              {error && (
                <div className="p-4 bg-red-50 border border-red-200 rounded-lg text-sm text-red-600">
                  {error}
                </div>
              )}

              <button
                type="submit"
                disabled={loading}
                className="w-full bg-gray-900 text-white py-3 rounded-xl font-semibold hover:bg-gray-800 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {loading ? 'Signing in...' : 'Sign In'}
              </button>

              <div className="text-center">
                <Link to="/forgot-password" className="text-sm text-gray-600 hover:text-gray-900 font-medium">
                  Forgot password?
                </Link>
              </div>

              {ssoProviders.length > 0 && (
                <>
                  <div className="relative my-6">
                    <div className="absolute inset-0 flex items-center">
                      <div className="w-full border-t border-gray-200"></div>
                    </div>
                    <div className="relative flex justify-center text-sm">
                      <span className="px-4 bg-white text-gray-500">Or continue with</span>
                    </div>
                  </div>
                  <div className="space-y-2">
                    {ssoProviders.map((provider) => (
                      <button
                        key={provider.id}
                        type="button"
                        onClick={() => handleSSOLogin(provider.id)}
                        className="w-full py-3 border-2 border-gray-200 rounded-xl font-medium text-gray-700 hover:border-gray-900 hover:bg-gray-50 transition-all"
                      >
                        {provider.name}
                      </button>
                    ))}
                  </div>
                </>
              )}

              <div className="text-center pt-4">
                <p className="text-sm text-gray-600">
                  Don't have an account?{' '}
                  <Link to="/signup" className="font-semibold text-gray-900 hover:underline">
                    Sign up
                  </Link>
                </p>
              </div>
            </form>
          )}
        </div>
      </div>
    </div>
  )
}

import { useState } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { apiClient } from '@/shared/lib/api-client'

export function LoginPage() {
  const navigate = useNavigate()
  const [showForm, setShowForm] = useState(false)
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  // MFA challenge state
  const [mfaToken, setMfaToken] = useState('')
  const [mfaCode, setMfaCode] = useState('')

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setLoading(true)

    try {
      const res = await apiClient.post<{ token?: string; mfaRequired?: boolean; mfaToken?: string; user?: { id: string; email: string; name: string } }>(
        '/auth/login',
        { email, password },
      )

      if (res.mfaRequired && res.mfaToken) {
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
            {mfaToken ? 'Verification Required' : 'Sign In to Dipole'}
          </h1>
          {mfaToken && (
            <p className="text-sm text-gray-500 mt-1 text-center">
              Enter the 6-digit code from your authenticator app or a backup code.
            </p>
          )}
        </div>

        {mfaToken ? (
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
          </form>
        )}
      </div>
    </div>
  )
}

import { useState, useEffect } from 'react'
import { Link, useSearchParams, useNavigate } from 'react-router-dom'
import { apiClient } from '@/shared/lib/api-client'

export function ResetPasswordPage() {
  const [params] = useSearchParams()
  const navigate = useNavigate()

  useEffect(() => { document.title = 'Dipole | Reset Password' }, [])

  const tokenFromUrl = params.get('token') || ''

  const [token, setToken] = useState(tokenFromUrl)
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState(false)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')

    if (!token.trim()) {
      setError('Reset token is required')
      return
    }
    if (newPassword.length < 8) {
      setError('Password must be at least 8 characters')
      return
    }
    if (newPassword !== confirmPassword) {
      setError('Passwords do not match')
      return
    }

    setLoading(true)
    try {
      await apiClient.post('/auth/reset-password', { token: token.trim(), newPassword })
      setSuccess(true)
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Something went wrong.'
      setError(msg.includes('expired') ? 'This reset link has expired. Please request a new one.' : msg)
    } finally {
      setLoading(false)
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
            {success ? 'Password reset' : 'Set new password'}
          </h1>

          {success ? (
            <div className="space-y-6">
              <div className="p-4 bg-green-50 border border-green-200 rounded-xl text-sm text-green-700">
                Your password has been reset successfully. You can now sign in with your new password.
              </div>
              <button
                type="button"
                onClick={() => navigate('/login')}
                className="w-full bg-gray-900 text-white py-3 rounded-xl font-semibold hover:bg-gray-800 transition-colors"
              >
                Go to Sign In
              </button>
            </div>
          ) : (
            <form onSubmit={handleSubmit} className="space-y-4">
              <p className="text-sm text-gray-600 text-center mb-4">
                Enter your new password below.
              </p>

              {!tokenFromUrl && (
                <div>
                  <input
                    type="text"
                    placeholder="Reset token"
                    value={token}
                    onChange={(e) => setToken(e.target.value)}
                    className="w-full px-4 py-3 border-2 border-gray-200 rounded-xl focus:border-gray-900 focus:outline-none transition-colors font-mono text-xs"
                  />
                </div>
              )}

              <div>
                <input
                  type="password"
                  placeholder="New password"
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  autoComplete="new-password"
                  className="w-full px-4 py-3 border-2 border-gray-200 rounded-xl focus:border-gray-900 focus:outline-none transition-colors"
                />
              </div>

              <div>
                <input
                  type="password"
                  placeholder="Confirm new password"
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  autoComplete="new-password"
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
                {loading ? 'Resetting...' : 'Reset Password'}
              </button>

              <Link to="/login" className="inline-block text-sm text-gray-600 hover:text-gray-900 font-medium w-full text-center">
                ← Back to Sign In
              </Link>
            </form>
          )}
        </div>
      </div>
    </div>
  )
}

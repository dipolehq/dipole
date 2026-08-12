import { useState } from 'react'
import { Link, useSearchParams, useNavigate } from 'react-router-dom'
import { apiClient } from '@/shared/lib/api-client'

export function ResetPasswordPage() {
  const [params] = useSearchParams()
  const navigate = useNavigate()
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
      setError('Reset token is required.')
      return
    }
    if (newPassword.length < 6) {
      setError('Password must be at least 6 characters.')
      return
    }
    if (newPassword !== confirmPassword) {
      setError('Passwords do not match.')
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

  if (success) {
    return (
      <div className="h-screen w-screen bg-[#f5f5f5] flex items-center justify-center">
        <div className="w-[400px] bg-white rounded-xl shadow-lg p-8">
          <div className="flex flex-col items-center mb-6">
            <img src="/logo_black.svg" alt="Dipole" className="h-12 w-12 mb-3" />
            <h1 className="text-lg font-semibold text-gray-900">Password Reset</h1>
          </div>

          <div className="bg-green-50 border border-green-200 rounded-md p-4 mb-6">
            <p className="text-sm text-green-800">
              Your password has been reset successfully. You can now sign in with your new password.
            </p>
          </div>

          <button
            type="button"
            onClick={() => navigate('/login')}
            className="w-full py-3 bg-orange-500 hover:bg-orange-600 text-white text-sm font-medium rounded-md transition-colors"
          >
            Go to Sign In
          </button>
        </div>
      </div>
    )
  }

  return (
    <div className="h-screen w-screen bg-[#f5f5f5] flex items-center justify-center">
      <div className="w-[400px] bg-white rounded-xl shadow-lg p-8">
        <div className="flex flex-col items-center mb-8">
          <img src="/logo_black.svg" alt="Dipole" className="h-12 w-12 mb-3" />
          <h1 className="text-lg font-semibold text-gray-900">Set New Password</h1>
          <p className="text-sm text-gray-500 mt-1 text-center">
            Enter your new password below.
          </p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          {!tokenFromUrl && (
            <div>
              <input
                type="text"
                placeholder="Reset token"
                value={token}
                onChange={(e) => setToken(e.target.value)}
                required
                className="w-full px-4 py-3 border border-gray-300 rounded-md text-sm font-mono focus:outline-none focus:ring-2 focus:ring-orange-500 focus:border-transparent"
              />
            </div>
          )}

          <div>
            <input
              type="password"
              placeholder="New password"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              required
              minLength={6}
              className="w-full px-4 py-3 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-orange-500 focus:border-transparent"
            />
          </div>

          <div>
            <input
              type="password"
              placeholder="Confirm new password"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              required
              minLength={6}
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
            {loading ? 'Resetting…' : 'Reset Password'}
          </button>

          <Link
            to="/login"
            className="block text-center text-sm text-gray-500 hover:text-gray-700"
          >
            ← Back to Sign In
          </Link>
        </form>
      </div>
    </div>
  )
}

import { useState } from 'react'
import { Link } from 'react-router-dom'
import { apiClient } from '@/shared/lib/api-client'

export function ForgotPasswordPage() {
  const [email, setEmail] = useState('')
  const [loading, setLoading] = useState(false)
  const [submitted, setSubmitted] = useState(false)
  const [resetToken, setResetToken] = useState('')
  const [error, setError] = useState('')

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setLoading(true)

    try {
      const res = await apiClient.post<{ token?: string; message: string }>(
        '/auth/forgot-password',
        { email },
      )
      setSubmitted(true)
      if (res.token) {
        setResetToken(res.token)
      }
    } catch {
      setError('Something went wrong. Please try again.')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="h-screen w-screen bg-[#f5f5f5] flex items-center justify-center">
      <div className="w-[400px] bg-white rounded-xl shadow-lg p-8">
        <div className="flex flex-col items-center mb-8">
          <img src="/logo_black.svg" alt="Dipole" className="h-12 w-12 mb-3" />
          <h1 className="text-lg font-semibold text-gray-900">Reset Password</h1>
          {!submitted && (
            <p className="text-sm text-gray-500 mt-1 text-center">
              Enter your email and we'll send you a reset link.
            </p>
          )}
        </div>

        {submitted ? (
          <div className="space-y-4">
            <div className="bg-green-50 border border-green-200 rounded-md p-4">
              <p className="text-sm text-green-800">
                If an account with that email exists, a password reset link has been sent.
              </p>
            </div>

            {resetToken && (
              <div className="bg-gray-50 border border-gray-200 rounded-md p-4">
                <p className="text-xs text-gray-500 mb-1 font-medium">Dev Mode — Reset Token:</p>
                <p className="text-xs text-gray-800 font-mono break-all select-all">{resetToken}</p>
                <Link
                  to={`/reset-password?token=${encodeURIComponent(resetToken)}`}
                  className="inline-block mt-3 text-xs text-gray-700 hover:text-gray-800 font-medium"
                >
                  Use this token to reset password →
                </Link>
              </div>
            )}

            <Link
              to="/login"
              className="block text-center text-sm text-gray-700 hover:text-gray-800 font-medium"
            >
              ← Back to Sign In
            </Link>
          </div>
        ) : (
          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <input
                type="email"
                placeholder="Email address"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
                className="w-full px-4 py-3 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-gray-400 focus:border-transparent"
              />
            </div>

            {error && (
              <p className="text-sm text-red-600 text-center">{error}</p>
            )}

            <button
              type="submit"
              disabled={loading}
              className="w-full py-3 bg-gray-800 hover:bg-gray-900 text-white text-sm font-medium rounded-md transition-colors disabled:opacity-50"
            >
              {loading ? 'Sending…' : 'Send Reset Link'}
            </button>

            <Link
              to="/login"
              className="block text-center text-sm text-gray-500 hover:text-gray-700"
            >
              ← Back to Sign In
            </Link>
          </form>
        )}
      </div>
    </div>
  )
}

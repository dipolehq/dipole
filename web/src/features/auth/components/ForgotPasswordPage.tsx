import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { apiClient } from '@/shared/lib/api-client'

export function ForgotPasswordPage() {
  const [email, setEmail] = useState('')
  const [loading, setLoading] = useState(false)
  const [submitted, setSubmitted] = useState(false)
  const [resetToken, setResetToken] = useState('')
  const [error, setError] = useState('')

  useEffect(() => {
    document.title = 'Dipole | Reset Password'
  }, [])

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')

    if (!email.trim()) {
      setError('Email is required')
      return
    }

    setLoading(true)
    try {
      const res = await apiClient.post<{ token?: string; message: string }>(
        '/auth/forgot-password',
        { email }
      )
      setSubmitted(true)
      if (res.token) setResetToken(res.token)
    } catch {
      setError('Something went wrong. Please try again.')
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
            {submitted ? 'Check your email' : 'Reset password'}
          </h1>

          {submitted ? (
            <div className="space-y-6">
              <div className="p-4 bg-green-50 border border-green-200 rounded-xl text-sm text-green-700">
                If an account with that email exists, a password reset link has been sent.
              </div>

              {resetToken && (
                <div className="p-4 bg-yellow-50 border border-yellow-200 rounded-xl text-sm space-y-3">
                  <p className="font-semibold text-yellow-900">Dev Mode — Reset Token:</p>
                  <p className="font-mono text-xs break-all bg-white p-3 rounded-lg border border-yellow-200">
                    {resetToken}
                  </p>
                  <Link
                    to={`/reset-password?token=${encodeURIComponent(resetToken)}`}
                    className="inline-block text-yellow-900 hover:underline font-medium"
                  >
                    Use this token to reset password →
                  </Link>
                </div>
              )}

              <Link to="/login" className="inline-block text-sm text-gray-600 hover:text-gray-900 font-medium">
                ← Back to Sign In
              </Link>
            </div>
          ) : (
            <form onSubmit={handleSubmit} className="space-y-4">
              <p className="text-sm text-gray-600 text-center">
                Enter your email and we'll send you a reset link.
              </p>

              <div>
                <input
                  type="email"
                  placeholder="Your email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  autoComplete="email"
                  autoFocus
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
                {loading ? 'Sending...' : 'Send Reset Link'}
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

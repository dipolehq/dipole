import { useState, useRef, useEffect } from 'react'
import { Link, useSearchParams, useNavigate } from 'react-router-dom'
import { apiClient } from '@/shared/lib/api-client'
import { AuthLayout } from './AuthLayout'
import { type RobotHandle } from './RobotGuard'

export function ResetPasswordPage() {
  const [params] = useSearchParams()
  const navigate = useNavigate()
  const robotRef = useRef<RobotHandle>(null)

  useEffect(() => { document.title = 'Dipole | Reset Password' }, [])

  const tokenFromUrl = params.get('token') || ''

  const [token, setToken] = useState(tokenFromUrl)
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState(false)

  function handlePassFocus() {
    robotRef.current?.setMood('shy')
    robotRef.current?.turnAway(true)
    robotRef.current?.say("New secret. I won't look.")
  }

  function handlePassBlur(e: React.FocusEvent) {
    const related = e.relatedTarget as HTMLElement | null
    if (related?.tagName === 'INPUT') return
    robotRef.current?.turnAway(false)
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')

    if (!token.trim()) { setError('Reset token is required.'); return }
    if (newPassword.length < 8) { setError('Password must be at least 8 characters.'); robotRef.current?.say("Too short!"); return }
    if (newPassword !== confirmPassword) { setError('Passwords do not match.'); robotRef.current?.say("Those don't match."); return }

    setLoading(true)
    try {
      await apiClient.post('/auth/reset-password', { token: token.trim(), newPassword })
      setSuccess(true)
      robotRef.current?.success("Password reset! You're all set.")
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Something went wrong.'
      setError(msg.includes('expired') ? 'This reset link has expired. Please request a new one.' : msg)
      robotRef.current?.say("That didn't work.")
      robotRef.current?.setMood('watching')
    } finally {
      setLoading(false)
    }
  }

  return (
    <AuthLayout robotRef={robotRef}>
      <form className="card" onSubmit={handleSubmit} noValidate>
        <span className="hand hand--l" aria-hidden="true" />
        <span className="hand hand--r" aria-hidden="true" />

        <div className="title-row">
          <img src="/logo_black.svg" alt="Dipole" className="card-logo" draggable={false} />
          <h1 className="title">{success ? 'Password reset' : 'Set new password'}</h1>
          <img src="/logo_black_mirror.svg" alt="" className="card-logo" draggable={false} />
        </div>

        {success ? (
          <div>
            <div className="success-box">
              <p>Your password has been reset successfully. You can now sign in with your new password.</p>
            </div>
            <button
              type="button"
              className="btn"
              onClick={() => navigate('/login')}
              onMouseEnter={() => robotRef.current?.hype(true)}
              onMouseLeave={() => robotRef.current?.hype(false)}
            >
              <span className="btn-label">GO TO SIGN IN</span>
            </button>
          </div>
        ) : (
          <>
            <p className="form-subtitle">Enter your new password below.</p>

            {!tokenFromUrl && (
              <label className="field">
                <svg className="field-icon" viewBox="0 0 24 24" aria-hidden="true"><path d="M12 2a5 5 0 0 0-5 5v3H6a2 2 0 0 0-2 2v8a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-8a2 2 0 0 0-2-2h-1V7a5 5 0 0 0-5-5Zm-3 8V7a3 3 0 0 1 6 0v3H9Zm3 4a2 2 0 0 1 1 3.7V19h-2v-1.3a2 2 0 0 1 1-3.7Z" /></svg>
                <input
                  type="text"
                  placeholder="Reset token"
                  value={token}
                  onChange={(e) => setToken(e.target.value)}
                  aria-label="Reset token"
                  style={{ fontFamily: 'monospace', fontSize: '12px' }}
                />
              </label>
            )}

            <label className="field">
              <svg className="field-icon" viewBox="0 0 24 24" aria-hidden="true"><path d="M12 2a5 5 0 0 0-5 5v3H6a2 2 0 0 0-2 2v8a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-8a2 2 0 0 0-2-2h-1V7a5 5 0 0 0-5-5Zm-3 8V7a3 3 0 0 1 6 0v3H9Zm3 4a2 2 0 0 1 1 3.7V19h-2v-1.3a2 2 0 0 1 1-3.7Z" /></svg>
              <input
                type="password"
                placeholder="New password"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                onFocus={handlePassFocus}
                onBlur={handlePassBlur}
                autoComplete="new-password"
                aria-label="New password"
              />
            </label>

            <label className="field">
              <svg className="field-icon" viewBox="0 0 24 24" aria-hidden="true"><path d="M12 2a5 5 0 0 0-5 5v3H6a2 2 0 0 0-2 2v8a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-8a2 2 0 0 0-2-2h-1V7a5 5 0 0 0-5-5Zm-3 8V7a3 3 0 0 1 6 0v3H9Zm3 4a2 2 0 0 1 1 3.7V19h-2v-1.3a2 2 0 0 1 1-3.7Z" /></svg>
              <input
                type="password"
                placeholder="Confirm new password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                onFocus={handlePassFocus}
                onBlur={handlePassBlur}
                autoComplete="new-password"
                aria-label="Confirm new password"
              />
            </label>

            {error && <p className="form-error">{error}</p>}

            <button
              type="submit"
              className="btn"
              disabled={loading}
              onMouseEnter={() => robotRef.current?.hype(true)}
              onMouseLeave={() => robotRef.current?.hype(false)}
              onPointerDown={() => robotRef.current?.press()}
              onPointerUp={() => robotRef.current?.release()}
            >
              <span className="btn-bolt" aria-hidden="true">&#9889;</span>
              <span className="btn-label">{loading ? 'Resetting...' : 'RESET PASSWORD'}</span>
            </button>

            <div className="form-links">
              <Link to="/login" className="form-link">&larr; Back to Sign In</Link>
            </div>
          </>
        )}

        <span className="foot foot--l" aria-hidden="true" />
        <span className="foot foot--r" aria-hidden="true" />
      </form>
    </AuthLayout>
  )
}

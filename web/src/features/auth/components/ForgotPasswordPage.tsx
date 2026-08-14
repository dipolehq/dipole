import { useState, useRef } from 'react'
import { Link } from 'react-router-dom'
import { apiClient } from '@/shared/lib/api-client'
import { AuthLayout } from './AuthLayout'
import { type RobotHandle } from './RobotGuard'

export function ForgotPasswordPage() {
  const robotRef = useRef<RobotHandle>(null)
  const [email, setEmail] = useState('')
  const [loading, setLoading] = useState(false)
  const [submitted, setSubmitted] = useState(false)
  const [resetToken, setResetToken] = useState('')
  const [error, setError] = useState('')

  function handleEmailFocus() {
    robotRef.current?.setMood('watching')
    robotRef.current?.turnAway(false)
    robotRef.current?.say("Where should I send the reset link?")
    robotRef.current?.followTyping(email)
  }

  function handleEmailChange(value: string) {
    setEmail(value)
    robotRef.current?.followTyping(value)
    const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]{2,}$/
    if (EMAIL_RE.test(value.trim())) {
      robotRef.current?.setMood('happy')
      robotRef.current?.say("Got it. Ready when you are.")
    }
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    if (!email.trim()) {
      setError('Email is required.')
      robotRef.current?.say("I need an email to help you.")
      return
    }
    setLoading(true)

    try {
      const res = await apiClient.post<{ token?: string; message: string }>(
        '/auth/forgot-password',
        { email },
      )
      setSubmitted(true)
      if (res.token) setResetToken(res.token)
      robotRef.current?.success("Done! Check your inbox.")
    } catch {
      setError('Something went wrong. Please try again.')
      robotRef.current?.say("Something broke. Try again?")
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
          <h1 className="title">{submitted ? 'Check your email' : 'Reset password'}</h1>
          <img src="/logo_black_mirror.svg" alt="" className="card-logo" draggable={false} />
        </div>

        {submitted ? (
          <div>
            <div className="success-box">
              <p>If an account with that email exists, a password reset link has been sent.</p>
            </div>

            {resetToken && (
              <div className="dev-box">
                <p className="dev-label">Dev Mode — Reset Token:</p>
                <p className="dev-token">{resetToken}</p>
                <Link
                  to={`/reset-password?token=${encodeURIComponent(resetToken)}`}
                  className="dev-link"
                >
                  Use this token to reset password &rarr;
                </Link>
              </div>
            )}

            <div className="form-links">
              <Link to="/login" className="form-link">&larr; Back to Sign In</Link>
            </div>
          </div>
        ) : (
          <>
            <p className="form-subtitle">Enter your email and we'll send you a reset link.</p>

            <label className="field">
              <svg className="field-icon" viewBox="0 0 24 24" aria-hidden="true"><path d="M4 5h16a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V7a2 2 0 0 1 2-2Zm8 7.3L4.4 7h15.2L12 12.3ZM4 9.2V17h16V9.2l-8 5.3-8-5.3Z" /></svg>
              <input
                type="email"
                placeholder="Your email"
                value={email}
                onChange={(e) => handleEmailChange(e.target.value)}
                onFocus={handleEmailFocus}
                autoComplete="email"
                autoFocus
                aria-label="Email"
              />
            </label>

            {error && <p className="form-error">{error}</p>}

            <button
              type="submit"
              className="btn"
              disabled={loading}
              onMouseEnter={() => { robotRef.current?.hype(true); robotRef.current?.setMood('excited'); robotRef.current?.say('Ooh. Do it. Press it.') }}
              onMouseLeave={() => { robotRef.current?.hype(false); robotRef.current?.setMood('idle') }}
              onPointerDown={() => { robotRef.current?.press(); robotRef.current?.say("Ahh. That's the stuff.") }}
              onPointerUp={() => robotRef.current?.release()}
            >
              <span className="btn-bolt" aria-hidden="true">&#9889;</span>
              <span className="btn-label">{loading ? 'Sending...' : 'SEND RESET LINK'}</span>
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

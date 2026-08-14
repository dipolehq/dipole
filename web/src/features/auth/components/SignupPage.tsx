import { useState, useRef } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { apiClient } from '@/shared/lib/api-client'
import { setCurrentTenantSubdomain } from '@/shared/lib/platform-urls'
import { AuthLayout } from './AuthLayout'
import { type RobotHandle } from './RobotGuard'
import { evaluatePassword } from './password-strength'

export function SignupPage() {
  const navigate = useNavigate()
  const robotRef = useRef<RobotHandle>(null)
  const [firstName, setFirstName] = useState('')
  const [lastName, setLastName] = useState('')
  const [companyName, setCompanyName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [showPass, setShowPass] = useState(false)
  const [passStrength, setPassStrength] = useState({ score: 0, label: '', isValid: false, feedback: '' })

  const pick = (arr: string[]) => arr[Math.floor(Math.random() * arr.length)]!

  function handleNameFocus() {
    robotRef.current?.turnAway(false)
    robotRef.current?.setMood('watching')
    robotRef.current?.say(pick(["Tell me your name.", "A visitor. State your identity."]))
  }

  function handleCompanyFocus() {
    robotRef.current?.turnAway(false)
    robotRef.current?.setMood('watching')
    robotRef.current?.say("What's your organization called?")
  }

  function handleEmailFocus() {
    robotRef.current?.turnAway(false)
    robotRef.current?.setMood('watching')
    robotRef.current?.say("Email next. I promise no spam.")
    robotRef.current?.followTyping(email)
  }

  function handleEmailChange(value: string) {
    setEmail(value)
    robotRef.current?.followTyping(value)
    const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]{2,}$/
    if (EMAIL_RE.test(value.trim())) {
      robotRef.current?.setMood('happy')
      robotRef.current?.say("Valid email. Nice.")
    }
  }

  function handlePassFocus() {
    robotRef.current?.setMood('shy')
    robotRef.current?.turnAway(true)
    robotRef.current?.say("Secret time. *turns around*")
  }

  function handlePassChange(value: string) {
    setPassword(value)
    const strength = evaluatePassword(value)
    setPassStrength(strength)
    robotRef.current?.setPasswordStrength(strength.score, strength.label)
  }

  function handlePassBlur(e: React.FocusEvent) {
    const related = e.relatedTarget as HTMLElement | null
    if (related?.dataset?.peek) return
    robotRef.current?.turnAway(false)
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')

    if (!firstName.trim()) { setError('First name is required.'); robotRef.current?.say("I need your name."); return }
    if (!lastName.trim()) { setError('Last name is required.'); return }
    if (!companyName.trim()) { setError('Company name is required.'); robotRef.current?.say("What company?"); return }
    if (!email.trim()) { setError('Email is required.'); robotRef.current?.say("Email, please."); return }
    if (!passStrength.isValid) { setError(passStrength.feedback || 'Password is too weak.'); robotRef.current?.say("Stronger password needed."); return }
    if (password !== confirmPassword) { setError('Passwords do not match.'); robotRef.current?.say("Those don't match."); return }

    setLoading(true)
    try {
      const res = await apiClient.post<{ token: string; user: { id: string; email: string; name: string }; subdomain: string }>(
        '/auth/signup',
        { firstName: firstName.trim(), lastName: lastName.trim(), companyName: companyName.trim(), email: email.trim(), password },
      )
      sessionStorage.setItem('dipole_token', res.token)
      sessionStorage.setItem('dipole_user', JSON.stringify(res.user))
      setCurrentTenantSubdomain(res.subdomain)
      robotRef.current?.success(`Welcome aboard, ${res.user.name || 'human'}!`)
      setTimeout(() => navigate('/'), 1500)
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Signup failed. Please try again.'
      setError(msg)
      robotRef.current?.say("Something went wrong.")
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
          <h1 className="title">Sign Up For Dipole Tenant</h1>
          <img src="/logo_black_mirror.svg" alt="" className="card-logo" draggable={false} />
        </div>

        <div className="field-row">
          <label className="field">
            <svg className="field-icon" viewBox="0 0 24 24" aria-hidden="true"><path d="M12 12a4.5 4.5 0 1 0-4.5-4.5A4.5 4.5 0 0 0 12 12Zm0 2c-3.9 0-8 2-8 5v1.5h16V19c0-3-4.1-5-8-5Z" /></svg>
            <input
              type="text"
              placeholder="First name"
              value={firstName}
              onChange={(e) => setFirstName(e.target.value)}
              onFocus={handleNameFocus}
              autoComplete="given-name"
              aria-label="First name"
            />
          </label>
          <label className="field">
            <svg className="field-icon" viewBox="0 0 24 24" aria-hidden="true"><path d="M12 12a4.5 4.5 0 1 0-4.5-4.5A4.5 4.5 0 0 0 12 12Zm0 2c-3.9 0-8 2-8 5v1.5h16V19c0-3-4.1-5-8-5Z" /></svg>
            <input
              type="text"
              placeholder="Last name"
              value={lastName}
              onChange={(e) => setLastName(e.target.value)}
              onFocus={handleNameFocus}
              autoComplete="family-name"
              aria-label="Last name"
            />
          </label>
        </div>

        <label className="field">
          <svg className="field-icon" viewBox="0 0 24 24" aria-hidden="true"><path d="M12 7V3H2v18h20V7H12ZM6 19H4v-2h2v2Zm0-4H4v-2h2v2Zm0-4H4V9h2v2Zm0-4H4V5h2v2Zm4 12H8v-2h2v2Zm0-4H8v-2h2v2Zm0-4H8V9h2v2Zm0-4H8V5h2v2Zm10 12h-8v-2h2v-2h-2v-2h2v-2h-2V9h8v10Zm-2-8h-2v2h2v-2Zm0 4h-2v2h2v-2Z" /></svg>
          <input
            type="text"
            placeholder="Company name"
            value={companyName}
            onChange={(e) => setCompanyName(e.target.value)}
            onFocus={handleCompanyFocus}
            autoComplete="organization"
            aria-label="Company name"
          />
        </label>

        <label className="field">
          <svg className="field-icon" viewBox="0 0 24 24" aria-hidden="true"><path d="M4 5h16a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V7a2 2 0 0 1 2-2Zm8 7.3L4.4 7h15.2L12 12.3ZM4 9.2V17h16V9.2l-8 5.3-8-5.3Z" /></svg>
          <input
            type="email"
            placeholder="Work email"
            value={email}
            onChange={(e) => handleEmailChange(e.target.value)}
            onFocus={handleEmailFocus}
            autoComplete="email"
            aria-label="Email"
          />
        </label>

        <label className="field">
          <svg className="field-icon" viewBox="0 0 24 24" aria-hidden="true"><path d="M12 2a5 5 0 0 0-5 5v3H6a2 2 0 0 0-2 2v8a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-8a2 2 0 0 0-2-2h-1V7a5 5 0 0 0-5-5Zm-3 8V7a3 3 0 0 1 6 0v3H9Zm3 4a2 2 0 0 1 1 3.7V19h-2v-1.3a2 2 0 0 1 1-3.7Z" /></svg>
          <input
            type={showPass ? 'text' : 'password'}
            placeholder="Password (min 8 characters)"
            value={password}
            onChange={(e) => handlePassChange(e.target.value)}
            onFocus={handlePassFocus}
            onBlur={handlePassBlur}
            autoComplete="new-password"
            aria-label="Password"
          />
          <button
            className="peek"
            type="button"
            data-peek="true"
            onClick={() => setShowPass(!showPass)}
            aria-label={showPass ? 'Hide password' : 'Show password'}
            aria-pressed={showPass}
          >
            <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 5c-5 0-9.3 3.1-11 7.5C2.7 16.9 7 20 12 20s9.3-3.1 11-7.5C21.3 8.1 17 5 12 5Zm0 12.5a5 5 0 1 1 5-5 5 5 0 0 1-5 5Zm0-8a3 3 0 1 0 3 3 3 3 0 0 0-3-3Z" /></svg>
          </button>
        </label>


        <label className="field">
          <svg className="field-icon" viewBox="0 0 24 24" aria-hidden="true"><path d="M12 2a5 5 0 0 0-5 5v3H6a2 2 0 0 0-2 2v8a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-8a2 2 0 0 0-2-2h-1V7a5 5 0 0 0-5-5Zm-3 8V7a3 3 0 0 1 6 0v3H9Zm3 4a2 2 0 0 1 1 3.7V19h-2v-1.3a2 2 0 0 1 1-3.7Z" /></svg>
          <input
            type={showPass ? 'text' : 'password'}
            placeholder="Confirm password"
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
            onFocus={handlePassFocus}
            onBlur={handlePassBlur}
            autoComplete="new-password"
            aria-label="Confirm password"
          />
        </label>

        {error && <p className="form-error">{error}</p>}

        <button
          type="submit"
          className="btn"
          disabled={loading}
          onMouseEnter={() => { robotRef.current?.hype(true); robotRef.current?.setMood('excited'); robotRef.current?.say(pick(['Ooh. Do it. Press it.', 'This is my favorite part.'])) }}
          onMouseLeave={() => { robotRef.current?.hype(false); robotRef.current?.setMood('idle') }}
          onPointerDown={() => { robotRef.current?.press(); robotRef.current?.say(pick(["Ahh. That's the stuff.", "Beep. Do that again."])) }}
          onPointerUp={() => robotRef.current?.release()}
        >
          <span className="btn-bolt" aria-hidden="true">&#9889;</span>
          <span className="btn-label">{loading ? 'Creating account...' : 'SIGN ME UP'}</span>
        </button>

        <div className="form-links">
          <Link to="/login" className="form-link">Already have an account? <strong>Sign in</strong></Link>
        </div>

        <span className="foot foot--l" aria-hidden="true" />
        <span className="foot foot--r" aria-hidden="true" />
      </form>
    </AuthLayout>
  )
}

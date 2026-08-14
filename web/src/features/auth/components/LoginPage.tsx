import { useState, useEffect, useRef } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { apiClient } from '@/shared/lib/api-client'
import { setCurrentTenantSubdomain } from '@/shared/lib/platform-urls'
import { AuthLayout } from './AuthLayout'
import { type RobotHandle } from './RobotGuard'

interface SSOProvider {
  id: string
  name: string
  type: string
}

export function LoginPage() {
  const navigate = useNavigate()
  const robotRef = useRef<RobotHandle>(null)
  const [ssoProviders, setSsoProviders] = useState<SSOProvider[]>([])
  const [ssoLoading, setSsoLoading] = useState(false)
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [showPass, setShowPass] = useState(false)
  const [shake, setShake] = useState(false)
  const [btnLabel, setBtnLabel] = useState('SIGN ME IN')
  const [done, setDone] = useState(false)

  const [mfaToken, setMfaToken] = useState('')
  const [mfaCode, setMfaCode] = useState('')
  const [tenants, setTenants] = useState<{ id: string; name: string; subdomain: string }[]>([])

  const emailRef = useRef<HTMLInputElement>(null)
  const passRef = useRef<HTMLInputElement>(null)

  const pick = (arr: string[]) => arr[Math.floor(Math.random() * arr.length)]!

  // Load SSO providers
  useEffect(() => {
    document.title = 'Dipole | Sign In'
    apiClient.get<{ providers: SSOProvider[] }>('/auth/sso/providers')
      .then((res) => setSsoProviders(res.providers ?? []))
      .catch(() => {})
  }, [])

  // Email field
  function handleEmailFocus() {
    robotRef.current?.turnAway(false)
    robotRef.current?.setMood('watching')
    robotRef.current?.say(pick(["Email, please. I don't do spam.", "Typing detected. Go on."]))
    robotRef.current?.followTyping(email)
  }

  function handleEmailChange(value: string) {
    setEmail(value)
    robotRef.current?.followTyping(value)
    const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]{2,}$/
    if (EMAIL_RE.test(value.trim())) {
      robotRef.current?.setMood('happy')
      robotRef.current?.say(pick(['Valid email. Respect.', 'That looks right.']))
    } else {
      robotRef.current?.setMood('watching')
      if (value.includes('@')) robotRef.current?.say('Close. Not yet.')
    }
  }

  // Password field
  function handlePassFocus() {
    robotRef.current?.setMood('shy')
    robotRef.current?.turnAway(true)
    robotRef.current?.say("A secret? *turns around*")
  }

  function handlePassBlur(e: React.FocusEvent) {
    const related = e.relatedTarget as HTMLElement | null
    if (related?.dataset?.peek) return
    robotRef.current?.turnAway(false)
  }

  function handlePeek() {
    setShowPass(!showPass)
    if (!showPass) robotRef.current?.say("Peeking? Good thing I'm facing away.")
    passRef.current?.focus()
  }

  // Button interactions
  function handleBtnEnter() {
    if (done) return
    robotRef.current?.hype(true)
    robotRef.current?.setMood('excited')
    robotRef.current?.say(pick(['Ooh. Do it. Press it.', 'This is my favorite part.']))
  }
  function handleBtnLeave() {
    robotRef.current?.hype(false)
    if (!done) robotRef.current?.setMood('idle')
  }
  function handleBtnDown() {
    robotRef.current?.press()
    robotRef.current?.say(pick(["Ahh. That's the stuff.", "Beep. Do that again."]))
  }
  function handleBtnUp() { robotRef.current?.release() }

  // Submit
  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (done) return
    setError('')

    if (!email.trim()) {
      robotRef.current?.say("I need your email.")
      robotRef.current?.setMood('watching')
      setShake(true); setTimeout(() => setShake(false), 400)
      emailRef.current?.focus()
      return
    }
    if (!password) {
      robotRef.current?.say("A password would help.")
      robotRef.current?.setMood('watching')
      setShake(true); setTimeout(() => setShake(false), 400)
      passRef.current?.focus()
      return
    }

    setLoading(true)
    try {
      const res = await apiClient.post<{ token?: string; mfaRequired?: boolean; mfaToken?: string; tenantSubdomain?: string; user?: { id: string; email: string; name: string }; tenants?: { id: string; name: string; subdomain: string }[] }>(
        '/auth/login',
        { email, password },
      )

      if (res.tenants && res.tenants.length > 1) {
        setTenants(res.tenants)
        robotRef.current?.turnAway(false)
        robotRef.current?.setMood('happy')
        robotRef.current?.say('Multiple tenants. Pick one.')
      } else if (res.mfaRequired && res.mfaToken) {
        setMfaToken(res.mfaToken)
        robotRef.current?.turnAway(false)
        robotRef.current?.setMood('watching')
        robotRef.current?.say('Security check. Enter the code.')
      } else if (res.token && res.user) {
        setDone(true)
        setBtnLabel('ACCESS GRANTED')
        sessionStorage.setItem('dipole_token', res.token)
        sessionStorage.setItem('dipole_user', JSON.stringify(res.user))
        setCurrentTenantSubdomain(res.tenantSubdomain ?? '')
        robotRef.current?.success(`Welcome, ${res.user.name || 'human'}.`)
        setTimeout(() => navigate('/'), 1500)
      }
    } catch {
      setError('Invalid email or password')
      robotRef.current?.say("Access denied. Try again.")
      robotRef.current?.setMood('watching')
      setShake(true); setTimeout(() => setShake(false), 400)
    } finally {
      setLoading(false)
    }
  }

  async function handleTenantSelect(tenantId: string) {
    setError('')
    setLoading(true)
    try {
      const res = await apiClient.post<{ token?: string; mfaRequired?: boolean; mfaToken?: string; tenantSubdomain?: string; user?: { id: string; email: string; name: string } }>(
        '/auth/login',
        { email, password, tenantId },
      )
      if (res.mfaRequired && res.mfaToken) {
        setTenants([])
        setMfaToken(res.mfaToken)
        robotRef.current?.say('Security check. Enter the code.')
      } else if (res.token && res.user) {
        setDone(true)
        setBtnLabel('ACCESS GRANTED')
        sessionStorage.setItem('dipole_token', res.token)
        sessionStorage.setItem('dipole_user', JSON.stringify(res.user))
        setCurrentTenantSubdomain(res.tenantSubdomain ?? '')
        robotRef.current?.success(`Welcome, ${res.user.name || 'human'}.`)
        setTimeout(() => navigate('/'), 1500)
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
      const res = await apiClient.post<{ token: string; tenantSubdomain?: string; user: { id: string; email: string; name: string } }>(
        '/auth/mfa-verify',
        { mfaToken, code: mfaCode },
      )
      setDone(true)
      sessionStorage.setItem('dipole_token', res.token)
      sessionStorage.setItem('dipole_user', JSON.stringify(res.user))
      setCurrentTenantSubdomain(res.tenantSubdomain ?? '')
      robotRef.current?.success(`Verified. Welcome, ${res.user.name || 'human'}.`)
      setTimeout(() => navigate('/'), 1500)
    } catch {
      setError('Invalid verification code')
      robotRef.current?.say("That code doesn't check out.")
    } finally {
      setLoading(false)
    }
  }

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

  return (
    <AuthLayout robotRef={robotRef}>
      <form className={`card ${shake ? 'shake' : ''}`} onSubmit={handleSubmit} noValidate>
        <span className="hand hand--l" aria-hidden="true" />
        <span className="hand hand--r" aria-hidden="true" />

        <div className="title-row">
          <img src="/logo_black.svg" alt="Dipole" className="card-logo" draggable={false} />
          <h1 className="title">
            {tenants.length > 0 ? 'Select your tenant' : mfaToken ? 'Verification required' : 'Sign in to Dipole'}
          </h1>
          <img src="/logo_black_mirror.svg" alt="" className="card-logo" draggable={false} />
        </div>

        {/* Tenant selection */}
        {tenants.length > 0 ? (
          <div className="tenant-list">
            {tenants.map((t) => (
              <button key={t.id} type="button" onClick={() => handleTenantSelect(t.id)} disabled={loading} className="tenant-btn">
                <span className="tenant-name">{t.name}</span>
                <span className="tenant-sub">{t.subdomain}</span>
              </button>
            ))}
            {error && <p className="form-error">{error}</p>}
            <button type="button" onClick={() => { setTenants([]); setError('') }} className="back-link">Back</button>
          </div>
        ) : mfaToken ? (
          /* MFA */
          <div>
            <label className="field">
              <svg className="field-icon" viewBox="0 0 24 24" aria-hidden="true"><path d="M12 2a5 5 0 0 0-5 5v3H6a2 2 0 0 0-2 2v8a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-8a2 2 0 0 0-2-2h-1V7a5 5 0 0 0-5-5Zm-3 8V7a3 3 0 0 1 6 0v3H9Zm3 4a2 2 0 0 1 1 3.7V19h-2v-1.3a2 2 0 0 1 1-3.7Z" /></svg>
              <input
                type="text"
                placeholder="6-digit code"
                value={mfaCode}
                onChange={(e) => setMfaCode(e.target.value.replace(/[^a-zA-Z0-9-]/g, '').slice(0, 9))}
                autoFocus
                aria-label="Verification code"
                style={{ textAlign: 'center', letterSpacing: '0.3em', fontFamily: 'monospace' }}
              />
            </label>
            {error && <p className="form-error">{error}</p>}
            <button type="button" className="btn" onClick={() => handleMFAVerify()} disabled={loading || mfaCode.length < 6}>
              <span className="btn-label">{loading ? 'Verifying...' : 'VERIFY'}</span>
            </button>
            <button type="button" onClick={() => { setMfaToken(''); setMfaCode(''); setError('') }} className="back-link">Back</button>
          </div>
        ) : (
          /* Email + password */
          <>
            <label className="field">
              <svg className="field-icon" viewBox="0 0 24 24" aria-hidden="true"><path d="M4 5h16a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V7a2 2 0 0 1 2-2Zm8 7.3L4.4 7h15.2L12 12.3ZM4 9.2V17h16V9.2l-8 5.3-8-5.3Z" /></svg>
              <input
                ref={emailRef}
                type="email"
                placeholder="Your email"
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
                ref={passRef}
                type={showPass ? 'text' : 'password'}
                placeholder="Password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                onFocus={handlePassFocus}
                onBlur={handlePassBlur}
                autoComplete="current-password"
                aria-label="Password"
              />
              <button className="peek" type="button" data-peek="true" onClick={handlePeek} aria-label={showPass ? 'Hide password' : 'Show password'} aria-pressed={showPass}>
                <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 5c-5 0-9.3 3.1-11 7.5C2.7 16.9 7 20 12 20s9.3-3.1 11-7.5C21.3 8.1 17 5 12 5Zm0 12.5a5 5 0 1 1 5-5 5 5 0 0 1-5 5Zm0-8a3 3 0 1 0 3 3 3 3 0 0 0-3-3Z" /></svg>
              </button>
            </label>

            {error && <p className="form-error">{error}</p>}

            <button
              type="submit"
              className={`btn ${done ? 'is-success' : ''}`}
              disabled={loading}
              onMouseEnter={handleBtnEnter}
              onMouseLeave={handleBtnLeave}
              onPointerDown={handleBtnDown}
              onPointerUp={handleBtnUp}
            >
              <span className="btn-bolt" aria-hidden="true">&#9889;</span>
              <span className="btn-label">{loading ? 'Signing in...' : btnLabel}</span>
            </button>

            <div className="form-links">
              <Link to="/forgot-password" className="form-link">Forgot password?</Link>

              {ssoProviders.length > 0 && (
                <>
                  <div className="divider"><span>or</span></div>
                  {ssoProviders.map((provider) => (
                    <button key={provider.id} type="button" onClick={() => handleSSOLogin(provider.id)} disabled={ssoLoading} className="sso-btn">
                      {ssoLoading ? 'Redirecting...' : `Continue with ${provider.name}`}
                    </button>
                  ))}
                </>
              )}

              <Link to="/signup" className="form-link signup-link">Don't have an account? <strong>Sign up</strong></Link>
            </div>
          </>
        )}

        <span className="foot foot--l" aria-hidden="true" />
        <span className="foot foot--r" aria-hidden="true" />
      </form>
    </AuthLayout>
  )
}

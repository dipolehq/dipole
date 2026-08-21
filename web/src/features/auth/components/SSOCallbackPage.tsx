import { useEffect, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { apiClient } from '@/shared/lib/api-client'

/**
 * SSOCallbackPage handles the redirect back from the Identity Provider.
 * It extracts the authorization code from the URL, sends it to the backend
 * for exchange, stores the JWT, and redirects to the dashboard.
 */
export function SSOCallbackPage() {
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const [error, setError] = useState('')

  useEffect(() => {
    const code = params.get('code')
    const state = params.get('state')

    if (!code) {
      setError('No authorization code received from identity provider.')
      return
    }

    const storedState = sessionStorage.getItem('dipole_sso_state')
    if (storedState && state !== storedState) {
      setError('Invalid state parameter. Please try signing in again.')
      sessionStorage.removeItem('dipole_sso_state')
      sessionStorage.removeItem('dipole_sso_provider')
      return
    }

    const providerId = sessionStorage.getItem('dipole_sso_provider') || ''
    const callbackUrl = window.location.origin + '/auth/sso/callback'

    sessionStorage.removeItem('dipole_sso_state')
    sessionStorage.removeItem('dipole_sso_provider')

    apiClient.post<{ token: string; user: { id: string; email: string; name: string } }>(
      '/auth/sso/callback',
      { providerId, code, callbackUrl, state },
    ).then((res) => {
      sessionStorage.setItem('dipole_token', res.token)
      sessionStorage.setItem('dipole_user', JSON.stringify(res.user))
      navigate('/', { replace: true })
    }).catch((err: unknown) => {
      const msg = err instanceof Error ? err.message : 'SSO authentication failed.'
      setError(msg)
    })
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  if (error) {
    return (
      <div className="h-screen w-screen bg-[#f5f5f5] flex items-center justify-center">
        <div className="w-[400px] bg-white rounded-xl shadow-lg p-8">
          <div className="flex flex-col items-center mb-6">
            <img src="/logo_black.svg" alt="Dipole" className="h-12 w-12 mb-3" />
            <h1 className="text-lg font-semibold text-gray-900">SSO Error</h1>
          </div>
          <div className="bg-red-50 border border-red-200 rounded-md p-4 mb-4">
            <p className="text-sm text-red-800">{error}</p>
          </div>
          <button
            type="button"
            onClick={() => navigate('/login', { replace: true })}
            className="w-full py-3 bg-gray-800 hover:bg-gray-900 text-white text-sm font-medium rounded-md transition-colors"
          >
            Back to Sign In
          </button>
        </div>
      </div>
    )
  }

  return (
    <div className="h-screen w-screen bg-[#f5f5f5] flex items-center justify-center">
      <div className="w-[400px] bg-white rounded-xl shadow-lg p-8 text-center">
        <img src="/logo_black.svg" alt="Dipole" className="h-12 w-12 mx-auto mb-4" />
        <p className="text-sm text-gray-600">Completing sign in...</p>
      </div>
    </div>
  )
}

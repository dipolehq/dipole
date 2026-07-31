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

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setLoading(true)

    try {
      const res = await apiClient.post<{ token: string; user: { id: string; email: string; name: string } }>(
        '/auth/login',
        { email, password },
      )
      sessionStorage.setItem('dipole_token', res.token)
      sessionStorage.setItem('dipole_user', JSON.stringify(res.user))
      navigate('/')
    } catch {
      setError('Invalid email or password')
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
          <h1 className="text-lg font-semibold text-gray-900">Sign In to Dipole</h1>
        </div>

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
      </div>
    </div>
  )
}

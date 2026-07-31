import { type ReactNode } from 'react'
import { Navigate } from 'react-router-dom'

interface AuthGuardProps {
  children: ReactNode
}

export function AuthGuard({ children }: AuthGuardProps) {
  // Dev mode: auto-login bypass. Stripped from production builds by Vite.
  if (import.meta.env.DEV && !sessionStorage.getItem('dipole_token')) {
    sessionStorage.setItem('dipole_token', 'dev-bypass')
    sessionStorage.setItem('dipole_user', JSON.stringify({ id: 'dev', email: 'admin@dipole.dev', name: 'Admin' }))
    // Force reload so the token is picked up on next render.
    window.location.replace('/')
    return null
  }

  const token = sessionStorage.getItem('dipole_token')
  if (!token) {
    return <Navigate to="/login" replace />
  }
  return <>{children}</>
}

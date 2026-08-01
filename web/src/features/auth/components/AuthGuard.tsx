import { type ReactNode } from 'react'
import { Navigate } from 'react-router-dom'

interface AuthGuardProps {
  children: ReactNode
}

export function AuthGuard({ children }: AuthGuardProps) {
  const token = sessionStorage.getItem('dipole_token')
  if (!token) {
    return <Navigate to="/login" replace />
  }
  return <>{children}</>
}

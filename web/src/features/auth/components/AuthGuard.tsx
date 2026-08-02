import { type ReactNode } from 'react'
import { Navigate } from 'react-router-dom'
import { useSessionRefresh } from '../hooks/useSessionRefresh'

interface AuthGuardProps {
  children: ReactNode
}

export function AuthGuard({ children }: AuthGuardProps) {
  const token = sessionStorage.getItem('dipole_token')
  useSessionRefresh()
  if (!token) {
    return <Navigate to="/login" replace />
  }
  return <>{children}</>
}

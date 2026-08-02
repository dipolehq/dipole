/**
 * Silently refreshes the session token every 5 minutes while the user is active.
 * Activity is detected via mouse, keyboard, and touch events.
 * If no activity for 30 minutes, the token expires and the user is redirected to login.
 */
import { useEffect, useRef } from 'react'
import { BASE_URL } from '@/shared/lib/api-client'

const REFRESH_INTERVAL_MS = 5 * 60 * 1000
const INACTIVITY_THRESHOLD_MS = 25 * 60 * 1000

export function useSessionRefresh() {
  const lastActivityRef = useRef(Date.now())

  useEffect(() => {
    function markActive() {
      lastActivityRef.current = Date.now()
    }

    window.addEventListener('mousemove', markActive, { passive: true })
    window.addEventListener('keydown', markActive, { passive: true })
    window.addEventListener('touchstart', markActive, { passive: true })
    window.addEventListener('click', markActive, { passive: true })

    const interval = setInterval(async () => {
      const token = sessionStorage.getItem('dipole_token')
      if (!token) return

      const expiresAt = sessionStorage.getItem('dipole_token_expires')
      if (expiresAt && new Date(expiresAt).getTime() <= Date.now()) {
        sessionStorage.removeItem('dipole_token')
        sessionStorage.removeItem('dipole_user')
        sessionStorage.removeItem('dipole_token_expires')
        window.location.href = '/login'
        return
      }

      const idleTime = Date.now() - lastActivityRef.current
      if (idleTime > INACTIVITY_THRESHOLD_MS) {
        return
      }

      try {
        const resp = await fetch(`${BASE_URL}/auth/refresh`, {
          method: 'POST',
          headers: {
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json',
          },
        })

        if (resp.ok) {
          const data = await resp.json()
          if (data.token) {
            sessionStorage.setItem('dipole_token', data.token)
            if (data.expiresAt) {
              sessionStorage.setItem('dipole_token_expires', data.expiresAt)
            }
          }
        } else if (resp.status === 401) {
          sessionStorage.removeItem('dipole_token')
          sessionStorage.removeItem('dipole_user')
          sessionStorage.removeItem('dipole_token_expires')
          window.location.href = '/login'
        }
      } catch {
        // Silent failure — next interval will retry
      }
    }, REFRESH_INTERVAL_MS)

    return () => {
      clearInterval(interval)
      window.removeEventListener('mousemove', markActive)
      window.removeEventListener('keydown', markActive)
      window.removeEventListener('touchstart', markActive)
      window.removeEventListener('click', markActive)
    }
  }, [])
}

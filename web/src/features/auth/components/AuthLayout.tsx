import { type ReactNode, type RefObject } from 'react'
import { RobotGuard, type RobotHandle } from './RobotGuard'
import './login-robot.css'

interface AuthLayoutProps {
  children: ReactNode
  robotRef?: RefObject<RobotHandle | null>
}

/**
 * Shared split layout for auth pages (login, signup, forgot password).
 * Image stays on the left, robot + form on the right.
 */
export function AuthLayout({ children, robotRef }: AuthLayoutProps) {
  return (
    <div className="login-split">
      <div className="login-left">
        <img src="/loginpage.png" alt="" className="login-illustration" draggable={false} />
      </div>
      <div className="login-right">
        <div className="scene">
          <main className="stage">
            <RobotGuard ref={robotRef} />
            {children}
          </main>
        </div>
      </div>
    </div>
  )
}

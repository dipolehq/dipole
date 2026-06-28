import { type ReactNode } from 'react'
import { Sidebar } from '@/shared/components/Sidebar'

interface LayoutProps {
  children: ReactNode
}

export function Layout({ children }: LayoutProps) {
  return (
    <div className="min-h-screen bg-surface-secondary">
      <Sidebar />
      <main className="ml-52">{children}</main>
    </div>
  )
}

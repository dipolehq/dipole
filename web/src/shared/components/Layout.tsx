import { type ReactNode } from 'react'
import { Sidebar } from '@/shared/components/Sidebar'

interface LayoutProps {
  children: ReactNode
}

export function Layout({ children }: LayoutProps) {
  return (
    <div className="h-screen bg-surface-secondary overflow-hidden">
      <Sidebar />
      <main className="ml-[204px] h-full overflow-hidden">{children}</main>
    </div>
  )
}

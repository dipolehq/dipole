import { type ReactNode } from 'react'
import { Sidebar } from '@/shared/components/Sidebar'

interface LayoutProps {
  children: ReactNode
}

export function Layout({ children }: LayoutProps) {
  return (
    <div className="h-screen bg-surface-secondary overflow-hidden flex">
      <Sidebar />
      <main className="flex-1 ml-[204px] h-screen overflow-hidden flex flex-col">{children}</main>
    </div>
  )
}

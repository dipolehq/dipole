import { PageHeader } from '@/shared/components/PageHeader'
import { GettingStartedPanel } from '@/features/help/components/GettingStartedPanel'

export function HelpPage() {
  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-4 shrink-0 h-[44px] border-b border-border bg-white">
        <PageHeader />
      </div>
      <div className="flex-1 overflow-y-auto min-h-0 px-4 py-3">
        <GettingStartedPanel />
      </div>
    </div>
  )
}

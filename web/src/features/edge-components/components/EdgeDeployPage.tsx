import { PageHeader } from '@/shared/components/PageHeader'
import { DeployEdgePanel } from '@/features/edge-components/components/DeployEdgePanel'

export function EdgeDeployPage() {
  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-4 shrink-0 h-[60px] border-b border-border bg-white">
        <PageHeader />
      </div>
      <div className="flex-1 overflow-y-auto min-h-0">
        <DeployEdgePanel />
      </div>
    </div>
  )
}

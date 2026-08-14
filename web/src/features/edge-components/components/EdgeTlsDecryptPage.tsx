import { PageHeader } from '@/shared/components/PageHeader'
import { TlsDecryptPanel } from '@/features/edge-components/components/TlsDecryptPanel'

export function EdgeTlsDecryptPage() {
  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-4 shrink-0 h-[44px] border-b border-border bg-white">
        <PageHeader />
      </div>
      <TlsDecryptPanel />
    </div>
  )
}

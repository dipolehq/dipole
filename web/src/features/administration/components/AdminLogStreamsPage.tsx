
import { Link } from 'react-router-dom'
import { useSearchParams } from 'react-router-dom'
import { Plus, Search } from 'lucide-react'
import { PageHeader } from '@/shared/components/PageHeader'
import { LogStreamsTable } from '@/features/administration/components/LogStreamsTable'
import { useLogStreams } from '@/features/administration/hooks/useLogStreams'
import { PageSizeSelect } from '@/shared/components/PageSizeSelect'

const PAGE_SIZE_OPTIONS = [10, 25, 50] as const

export function AdminLogStreamsPage() {
  const [params, setParams] = useSearchParams()
  const searchQuery = params.get('search') || ''
  const pageSize = Number(params.get('perPage')) || 10
  const currentPage = Number(params.get('page')) || 1

  function setParam(key: string, value: string) {
    setParams((prev) => { const next = new URLSearchParams(prev); if (value) next.set(key, value); else next.delete(key); next.set('page', '1'); return next }, { replace: true })
  }
  function setPage(p: number) {
    setParams((prev) => { const next = new URLSearchParams(prev); next.set('page', String(p)); return next }, { replace: true })
  }

  const { data, isLoading } = useLogStreams(currentPage, pageSize, searchQuery)
  const items = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-4 shrink-0 h-[60px] border-b border-border bg-white">
        <PageHeader />
        <div className="flex items-center gap-3">
          <div className="relative">
            <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
            <input type="text" placeholder="Search" value={searchQuery} onChange={(e) => setParam('search', e.target.value)} className="pl-9 pr-3 py-2 border border-border rounded-md text-sm w-48 focus:outline-none focus:ring-1 focus:ring-gray-300 focus:border-gray-400" />
          </div>
          <Link to="/administration/log_streams/new" className="flex items-center gap-1.5 px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white text-sm font-medium rounded-md transition-colors">
            <Plus size={14} /> New
          </Link>
        </div>
      </div>
      <div className="flex-1 overflow-y-auto min-h-0">
        <LogStreamsTable items={items} isLoading={isLoading} />
      </div>
      <div className="flex items-center justify-between px-4 py-2 border-t border-border bg-gray-50 shrink-0">
        <div className="flex items-center gap-2 text-xs text-gray-600">
          <PageSizeSelect value={pageSize} options={PAGE_SIZE_OPTIONS} onChange={(v) => setParam('perPage', String(v))} />
          <span>Showing {total === 0 ? 0 : (currentPage - 1) * pageSize + 1} - {Math.min(currentPage * pageSize, total)} of {total} Results</span>
        </div>
        <div className="flex items-center gap-1">
          <button onClick={() => setPage(Math.max(1, currentPage - 1))} disabled={currentPage === 1} className="px-2 py-1 text-xs text-gray-500 disabled:opacity-30">←</button>
          {Array.from({ length: Math.min(totalPages, 5) }, (_, i) => (
            <button key={i + 1} onClick={() => setPage(i + 1)} className={`w-7 h-7 rounded text-xs font-medium ${currentPage === i + 1 ? 'bg-orange-500 text-white' : 'text-gray-600 hover:bg-gray-100'}`}>{i + 1}</button>
          ))}
          <button onClick={() => setPage(Math.min(totalPages, currentPage + 1))} disabled={currentPage === totalPages} className="px-2 py-1 text-xs text-gray-500 disabled:opacity-30">→</button>
        </div>
      </div>
    </div>
  )
}

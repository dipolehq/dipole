import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useSearchParams } from 'react-router-dom'
import { Search, Filter } from 'lucide-react'
import { PageHeader } from '@/shared/components/PageHeader'
import { Dropdown } from '@/shared/components/Dropdown'
import { TimespanPicker } from '@/shared/components/TimespanPicker'
import { AuditLogsTable } from '@/features/reporting/components/AuditLogsTable'
import { PageSizeSelect } from '@/shared/components/PageSizeSelect'
import { fetchAuditLogs } from '@/features/reporting/api/eventsApi'
import { AUDIT_LOG_CATEGORY_OPTIONS, AUDIT_LOG_SEVERITY_OPTIONS, REPORTING_PAGE_SIZE_OPTIONS } from '@/shared/constants/reporting'

export function ReportingAuditPage() {
  const [params, setParams] = useSearchParams()

  const searchQuery = params.get('search') || ''
  const pageSize = Number(params.get('perPage')) || REPORTING_PAGE_SIZE_OPTIONS[0]
  const currentPage = Number(params.get('page')) || 1
  const timeRange = params.get('timeRange') || '1d'
  const severity = params.get('severity') || ''
  const category = params.get('category') || ''

  function setParam(key: string, value: string) {
    setParams((prev) => {
      const next = new URLSearchParams(prev)
      if (value) next.set(key, value)
      else next.delete(key)
      if (key !== 'page') next.set('page', '1')
      return next
    }, { replace: true })
  }

  const [showFilter, setShowFilter] = useState(false)

  const { data } = useQuery({
    queryKey: ['events', 'audit', currentPage, pageSize, severity, category],
    queryFn: () => fetchAuditLogs({ page: currentPage, perPage: pageSize, severity, eventType: category }),
  })

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-4 shrink-0 h-[60px] border-b border-border bg-white">
        <PageHeader />
        <div className="flex items-center gap-3">
          <div className="relative">
            <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
            <input type="text" placeholder="Search" value={searchQuery} onChange={(e) => setParam('search', e.target.value)} className="pl-9 pr-3 py-2 border border-border rounded-md text-sm w-48 focus:outline-none focus:ring-1 focus:ring-gray-300 focus:border-gray-400" />
          </div>
          <div className="relative">
            <button type="button" onClick={() => setShowFilter(!showFilter)} className={`flex items-center gap-1.5 px-4 py-2 border rounded-md text-sm font-medium transition-colors ${showFilter ? 'border-orange-400 text-orange-600 bg-orange-50' : 'border-border text-gray-600 hover:border-gray-400'}`}>
              <Filter size={14} /> Filter
            </button>
            {showFilter && (
              <div className="absolute top-full right-0 mt-1 w-64 p-3 bg-white border border-border rounded-lg shadow-lg z-50">
                <div className="space-y-3">
                  <div>
                    <label className="block text-[11px] text-gray-500 mb-1">Timespan</label>
                    <TimespanPicker value={timeRange} onChange={(v) => setParam('timeRange', v)} />
                  </div>
                  <div>
                    <label className="block text-[11px] text-gray-500 mb-1">Category</label>
                    <Dropdown options={AUDIT_LOG_CATEGORY_OPTIONS.map((c) => ({ value: c.value, label: c.label }))} value={category} onChange={(v) => setParam('category', v)} placeholder="Category" />
                  </div>
                  <div>
                    <label className="block text-[11px] text-gray-500 mb-1">Severity</label>
                    <Dropdown options={AUDIT_LOG_SEVERITY_OPTIONS.map((s) => ({ value: s.value, label: s.label }))} value={severity} onChange={(v) => setParam('severity', v)} placeholder="Severity" />
                  </div>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto min-h-0">
        <AuditLogsTable searchQuery={searchQuery} timeRange={timeRange} severity={severity} category={category} data={data?.items} />
      </div>

      <div className="flex items-center justify-between px-4 py-2 border-t border-border bg-gray-50 shrink-0">
        <div className="flex items-center gap-2 text-xs text-gray-600">
          <PageSizeSelect value={pageSize} options={REPORTING_PAGE_SIZE_OPTIONS} onChange={(v) => setParam('perPage', String(v))} />
          <span>Showing {data?.total ? (currentPage - 1) * pageSize + 1 : 0} - {Math.min(currentPage * pageSize, data?.total ?? 0)} of {data?.total ?? 0} Results</span>
        </div>
        <div className="flex items-center gap-1">
          <button onClick={() => setParam('page', String(currentPage - 1))} disabled={currentPage <= 1} className="px-2 py-1 text-xs text-gray-500 hover:text-gray-700 disabled:opacity-30 disabled:cursor-not-allowed">←</button>
          <span className="w-7 h-7 flex items-center justify-center rounded text-xs font-medium bg-orange-500 text-white">{currentPage}</span>
          <button onClick={() => setParam('page', String(currentPage + 1))} disabled={currentPage >= Math.ceil((data?.total ?? 0) / pageSize)} className="px-2 py-1 text-xs text-gray-500 hover:text-gray-700 disabled:opacity-30 disabled:cursor-not-allowed">→</button>
        </div>
      </div>
    </div>
  )
}

import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useSearchParams } from 'react-router-dom'
import { Search, Filter } from 'lucide-react'
import { PageHeader } from '@/shared/components/PageHeader'
import { Dropdown } from '@/shared/components/Dropdown'
import { TimespanPicker } from '@/shared/components/TimespanPicker'
import { WorkloadEventsTable } from '@/features/reporting/components/WorkloadEventsTable'
import { PageSizeSelect } from '@/shared/components/PageSizeSelect'
import { fetchWorkloadEvents } from '@/features/reporting/api/eventsApi'
import { SEVERITY_OPTIONS, WORKLOAD_PROTOCOL_OPTIONS, REPORTING_PAGE_SIZE_OPTIONS } from '@/shared/constants/reporting'

export function ReportingWorkloadEventsPage() {
  const [params, setParams] = useSearchParams()

  const searchQuery = params.get('search') || ''
  const pageSize = Number(params.get('perPage')) || REPORTING_PAGE_SIZE_OPTIONS[0]
  const currentPage = Number(params.get('page')) || 1
  const timeRange = params.get('timeRange') || '1d'
  const severity = params.get('severity') || ''
  const appProtocol = params.get('appProtocol') || ''
  const mcpUser = params.get('mcpUser') || ''

  function setParam(key: string, value: string) {
    setParams((prev) => {
      const next = new URLSearchParams(prev)
      if (value) next.set(key, value)
      else next.delete(key)
      next.set('page', '1')
      return next
    }, { replace: true })
  }

  const [showFilter, setShowFilter] = useState(false)

  const { data } = useQuery({
    queryKey: ['events', 'workload', currentPage, pageSize, severity, appProtocol],
    queryFn: () => fetchWorkloadEvents({ page: currentPage, perPage: pageSize, severity, eventType: appProtocol }),
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
                    <label className="block text-[11px] text-gray-500 mb-1">Severity</label>
                    <Dropdown options={SEVERITY_OPTIONS.map((s) => ({ value: s.value, label: s.label }))} value={severity} onChange={(v) => setParam('severity', v)} placeholder="Severity" />
                  </div>
                  <div>
                    <label className="block text-[11px] text-gray-500 mb-1">Application Protocol</label>
                    <Dropdown options={WORKLOAD_PROTOCOL_OPTIONS.map((p) => ({ value: p.value, label: p.label }))} value={appProtocol} onChange={(v) => setParam('appProtocol', v)} placeholder="Protocol" />
                  </div>
                  <div>
                    <label className="block text-[11px] text-gray-500 mb-1">User (MCP only)</label>
                    <input type="text" value={mcpUser} onChange={(e) => setParam('mcpUser', e.target.value)} placeholder="User email" className="w-full px-3 py-1.5 border border-border rounded-md text-xs focus:outline-none focus:ring-1 focus:ring-gray-300" />
                  </div>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto min-h-0">
        <WorkloadEventsTable searchQuery={searchQuery} timeRange={timeRange} severity={severity} appProtocol={appProtocol} mcpUser={mcpUser} data={data?.items} />
      </div>

      <div className="flex items-center justify-between px-4 py-2 border-t border-border bg-gray-50 shrink-0">
        <div className="flex items-center gap-2 text-xs text-gray-600">
          <PageSizeSelect value={pageSize} options={REPORTING_PAGE_SIZE_OPTIONS} onChange={(v) => setParam('perPage', String(v))} />
          <span>Showing {data?.total ? (currentPage - 1) * pageSize + 1 : 0} - {Math.min(currentPage * pageSize, data?.total ?? 0)} of {data?.total ?? 0} Results</span>
        </div>
      </div>
    </div>
  )
}

import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useSearchParams } from 'react-router-dom'
import { Search, FileText, Filter } from 'lucide-react'
import { PageHeader } from '@/shared/components/PageHeader'
import { Dropdown } from '@/shared/components/Dropdown'
import { TimespanPicker } from '@/shared/components/TimespanPicker'
import { AccessAuthorizationTable } from '@/features/reporting/components/AccessAuthorizationTable'
import { AuditLogsTable } from '@/features/reporting/components/AuditLogsTable'
import { GlobalPolicyComplianceTable } from '@/features/reporting/components/GlobalPolicyComplianceTable'
import { WorkloadEventsTable } from '@/features/reporting/components/WorkloadEventsTable'
import { PageSizeSelect } from '@/shared/components/PageSizeSelect'
import { fetchAuthEvents, fetchWorkloadEvents, fetchAuditLogs } from '@/features/reporting/api/eventsApi'
import { LOG_TYPES, SEVERITY_OPTIONS, EVENT_TYPE_OPTIONS, REPORTING_PAGE_SIZE_OPTIONS, AUDIT_LOG_CATEGORY_OPTIONS, AUDIT_LOG_SEVERITY_OPTIONS, COMPLIANCE_STATUS_OPTIONS, RESOURCE_SET_OPTIONS, WORKLOAD_PROTOCOL_OPTIONS } from '@/shared/constants/reporting'
import type { LogType } from '@/shared/constants/reporting'

export function ReportingPage() {
  const [params, setParams] = useSearchParams()

  const selectedLogType = (params.get('logType') || 'audit_logs') as LogType
  const searchQuery = params.get('search') || ''
  const pageSize = Number(params.get('perPage')) || REPORTING_PAGE_SIZE_OPTIONS[0]
  const currentPage = Number(params.get('page')) || 1
  const timeRange = params.get('timeRange') || '1d'
  const severity = params.get('severity') || ''
  const eventType = params.get('eventType') || ''
  const category = params.get('category') || ''
  const complianceStatus = params.get('complianceStatus') || ''
  const resourceSet = params.get('resourceSet') || ''
  const appProtocol = params.get('appProtocol') || ''
  const mcpUser = params.get('mcpUser') || ''

  function setParam(key: string, value: string) {
    setParams((prev) => {
      const next = new URLSearchParams(prev)
      if (value) next.set(key, value)
      else next.delete(key)
      return next
    }, { replace: true })
  }

  function setSelectedLogType(v: LogType) {
    setParams((prev) => {
      const next = new URLSearchParams(prev)
      next.set('logType', v)
      next.set('page', '1')
      return next
    }, { replace: true })
  }
  function setSearchQuery(v: string) {
    setParams((prev) => {
      const next = new URLSearchParams(prev)
      if (v) next.set('search', v)
      else next.delete('search')
      next.set('page', '1')
      return next
    }, { replace: true })
  }
  function setPageSize(v: number) {
    setParams((prev) => {
      const next = new URLSearchParams(prev)
      next.set('perPage', String(v))
      next.set('page', '1')
      return next
    }, { replace: true })
  }
  function setTimeRange(v: string) { setParam('timeRange', v) }
  function setSeverity(v: string) { setParam('severity', v) }
  function setEventType(v: string) { setParam('eventType', v) }
  function setCategory(v: string) { setParam('category', v) }
  function setComplianceStatus(v: string) { setParam('complianceStatus', v) }
  function setResourceSet(v: string) { setParam('resourceSet', v) }
  function setAppProtocol(v: string) { setParam('appProtocol', v) }
  function setMcpUser(v: string) { setParam('mcpUser', v) }

  // Filter state (UI only — whether the dropdown is open)
  const [showFilter, setShowFilter] = useState(false)

  void currentPage

  // Fetch event data based on selected log type
  const authEventsQuery = useQuery({
    queryKey: ['events', 'auth', currentPage, pageSize, severity, eventType],
    queryFn: () => fetchAuthEvents({ page: currentPage, perPage: pageSize, severity, eventType }),
    enabled: selectedLogType === 'access_authorization',
  })

  const workloadEventsQuery = useQuery({
    queryKey: ['events', 'workload', currentPage, pageSize, severity, appProtocol],
    queryFn: () => fetchWorkloadEvents({ page: currentPage, perPage: pageSize, severity, eventType: appProtocol }),
    enabled: selectedLogType === 'workload_events',
  })

  const auditLogsQuery = useQuery({
    queryKey: ['events', 'audit', currentPage, pageSize, severity, category],
    queryFn: () => fetchAuditLogs({ page: currentPage, perPage: pageSize, severity, eventType: category }),
    enabled: selectedLogType === 'audit_logs',
  })

  function handleResetFilters() {
    setTimeRange('1d')
    setSeverity('')
    setEventType('')
    setCategory('')
    setComplianceStatus('')
    setResourceSet('')
    setAppProtocol('')
    setMcpUser('')
  }

  return (
    <div className="flex flex-col h-full">
      {/* Header bar */}
      <div className="flex items-center justify-between px-4 shrink-0 h-[60px] border-b border-border bg-white">
        <PageHeader />
        <div className="flex items-center gap-3">
          <div className="relative">
            <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
            <input
              type="text"
              placeholder="Search"
              value={searchQuery}
              onChange={(e) => { setSearchQuery(e.target.value) }}
              className="pl-9 pr-3 py-2 border border-border rounded-md text-sm w-48 focus:outline-none focus:ring-1 focus:ring-gray-300 focus:border-gray-400"
            />
          </div>
          <div className="relative">
            <button
              type="button"
              onClick={() => setShowFilter(!showFilter)}
              className={`flex items-center gap-1.5 px-4 py-2 border rounded-md text-sm font-medium transition-colors ${
                showFilter ? 'border-gray-400 text-gray-700 bg-gray-50' : 'border-border text-gray-600 hover:border-gray-400'
              }`}
            >
              <Filter size={14} />
              Filter
            </button>
            {showFilter && (
              <div className="absolute top-full right-0 mt-1 w-64 p-3 bg-white border border-border rounded-lg shadow-lg z-50">
                <div className="flex items-center justify-between mb-3">
                  <button type="button" onClick={handleResetFilters} className="text-[11px] text-gray-600 hover:text-gray-700 font-medium">Reset All</button>
                  <div className="flex items-center gap-2">
                    <button type="button" onClick={() => setShowFilter(false)} className="text-[11px] text-gray-500 hover:text-gray-700 font-medium">Cancel</button>
                    <button type="button" onClick={() => setShowFilter(false)} className="px-2.5 py-1 bg-gray-800 hover:bg-gray-900 text-white text-[11px] font-medium rounded-md transition-colors">Apply</button>
                  </div>
                </div>
                <div className="space-y-3">
                  {selectedLogType !== 'global_policy_compliance' && (
                    <div>
                      <label className="block text-[11px] text-gray-500 mb-1">Timespan</label>
                      <TimespanPicker value={timeRange} onChange={setTimeRange} />
                    </div>
                  )}
                  {selectedLogType === 'audit_logs' && (
                    <div>
                      <label className="block text-[11px] text-gray-500 mb-1">Category</label>
                      <Dropdown options={AUDIT_LOG_CATEGORY_OPTIONS.map((c) => ({ value: c.value, label: c.label }))} value={category} onChange={setCategory} placeholder="Category" />
                    </div>
                  )}
                  {selectedLogType !== 'global_policy_compliance' && (
                    <div>
                      <label className="block text-[11px] text-gray-500 mb-1">Severity</label>
                      <Dropdown options={(selectedLogType === 'audit_logs' ? AUDIT_LOG_SEVERITY_OPTIONS : SEVERITY_OPTIONS).map((s) => ({ value: s.value, label: s.label }))} value={severity} onChange={setSeverity} placeholder="Severity" />
                    </div>
                  )}
                  {selectedLogType === 'access_authorization' && (
                    <div>
                      <label className="block text-[11px] text-gray-500 mb-1">Event Type</label>
                      <Dropdown options={EVENT_TYPE_OPTIONS.map((e) => ({ value: e.value, label: e.label }))} value={eventType} onChange={setEventType} placeholder="Event Type" />
                    </div>
                  )}
                  {selectedLogType === 'workload_events' && (
                    <div>
                      <label className="block text-[11px] text-gray-500 mb-1">Application Protocol</label>
                      <Dropdown options={WORKLOAD_PROTOCOL_OPTIONS.map((p) => ({ value: p.value, label: p.label }))} value={appProtocol} onChange={setAppProtocol} placeholder="Application Protocol" />
                    </div>
                  )}
                  {selectedLogType === 'workload_events' && (
                    <div>
                      <label className="block text-[11px] text-gray-500 mb-1">User (MCP App Protocol only)</label>
                      <input type="text" value={mcpUser} onChange={(e) => setMcpUser(e.target.value)} placeholder="User (MCP App Protocol only)" className="w-full px-3 py-1.5 border border-border rounded-md text-xs focus:outline-none focus:ring-1 focus:ring-gray-300 focus:border-gray-400" />
                    </div>
                  )}
                  {selectedLogType === 'global_policy_compliance' && (
                    <div>
                      <label className="block text-[11px] text-gray-500 mb-1">Resource Set</label>
                      <Dropdown options={RESOURCE_SET_OPTIONS.map((r) => ({ value: r.value, label: r.label }))} value={resourceSet} onChange={setResourceSet} placeholder="Resource Set" />
                    </div>
                  )}
                  {selectedLogType === 'global_policy_compliance' && (
                    <div>
                      <label className="block text-[11px] text-gray-500 mb-1">Compliance Status</label>
                      <Dropdown options={COMPLIANCE_STATUS_OPTIONS.map((c) => ({ value: c.value, label: c.label }))} value={complianceStatus} onChange={setComplianceStatus} placeholder="Compliance Status" />
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>
          <LogTypeSelector
            value={selectedLogType}
            onChange={(v) => { setSelectedLogType(v) }}
          />
        </div>
      </div>

      {/* Table — full bleed */}
      <div className="flex-1 overflow-y-auto min-h-0">
        {selectedLogType === 'access_authorization' ? (
          <AccessAuthorizationTable data={authEventsQuery.data?.items} total={authEventsQuery.data?.total ?? 0} page={1} pageSize={50} onPageChange={() => {}} />
        ) : selectedLogType === 'audit_logs' ? (
          <AuditLogsTable searchQuery={searchQuery} timeRange={timeRange} severity={severity} category={category} data={auditLogsQuery.data?.items} />
        ) : selectedLogType === 'global_policy_compliance' ? (
          <GlobalPolicyComplianceTable searchQuery={searchQuery} complianceStatus={complianceStatus} resourceSet={resourceSet} />
        ) : selectedLogType === 'workload_events' ? (
          <WorkloadEventsTable searchQuery={searchQuery} timeRange={timeRange} severity={severity} appProtocol={appProtocol} mcpUser={mcpUser} data={workloadEventsQuery.data?.items} />
        ) : (
          <div className="flex items-center justify-center h-full">
            <p className="text-sm text-gray-400">Content coming soon</p>
          </div>
        )}
      </div>

      {/* Pagination footer */}
      <div className="flex items-center justify-between px-4 py-2 border-t border-border bg-gray-50 shrink-0">
        <div className="flex items-center gap-2 text-xs text-gray-600">
          <PageSizeSelect value={pageSize} options={REPORTING_PAGE_SIZE_OPTIONS} onChange={(v) => { setPageSize(v) }} />
          <span>Showing 0 - 0 of 0 Results</span>
        </div>
        <div className="flex items-center gap-1">
          <button disabled className="px-2 py-1 text-xs text-gray-500 hover:text-gray-700 disabled:opacity-30">←</button>
          <button className="w-7 h-7 rounded text-xs font-medium bg-gray-800 text-white">1</button>
          <button disabled className="px-2 py-1 text-xs text-gray-500 hover:text-gray-700 disabled:opacity-30">→</button>
        </div>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Log Type Selector
// ---------------------------------------------------------------------------

function LogTypeSelector({ value, onChange }: { value: LogType; onChange: (v: LogType) => void }) {
  const [open, setOpen] = useState(false)
  const selectedLabel = LOG_TYPES.find((t) => t.value === value)?.label ?? 'Select Log Type'

  return (
    <div className="relative">
      <button
        type="button"
        onClick={() => setOpen(!open)}
        className="flex items-center gap-1.5 w-[200px] px-4 py-2 bg-gray-800 hover:bg-gray-900 text-white text-sm font-medium rounded-md transition-colors"
      >
        <FileText size={14} className="shrink-0" />
        <span className="truncate">{selectedLabel}</span>
        <svg className={`w-3.5 h-3.5 ml-auto shrink-0 transition-transform ${open ? 'rotate-180' : ''}`} fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
        </svg>
      </button>
      {open && (
        <div className="absolute top-full right-0 mt-1 bg-white border border-border rounded-md shadow-lg z-50 w-64 overflow-hidden">
          {LOG_TYPES.map((opt) => (
            <button
              key={opt.value}
              type="button"
              onClick={() => { onChange(opt.value as LogType); setOpen(false) }}
              className={`w-full text-left px-4 py-2.5 text-sm transition-colors ${value === opt.value ? 'bg-gray-50 text-gray-700 font-medium' : 'text-gray-600 hover:bg-gray-50'}`}
            >
              {opt.label}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}

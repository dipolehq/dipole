import { useState } from 'react'
import { Search, FileText, Filter } from 'lucide-react'
import { ResourceSetDropdown } from '@/shared/components/ResourceSetDropdown'
import { PageHeader } from '@/shared/components/PageHeader'
import { Dropdown } from '@/shared/components/Dropdown'
import { TimespanPicker } from '@/shared/components/TimespanPicker'
import { AccessAuthorizationTable } from '@/features/reporting/components/AccessAuthorizationTable'
import { AuditLogsTable } from '@/features/reporting/components/AuditLogsTable'
import { GlobalPolicyComplianceTable } from '@/features/reporting/components/GlobalPolicyComplianceTable'
import { McpAuthorizationTracingTable } from '@/features/reporting/components/McpAuthorizationTracingTable'
import { WorkloadEventsTable } from '@/features/reporting/components/WorkloadEventsTable'
import { LOG_TYPES, SEVERITY_OPTIONS, EVENT_TYPE_OPTIONS, PAGE_SIZE_OPTIONS, AUDIT_LOG_CATEGORY_OPTIONS, AUDIT_LOG_SEVERITY_OPTIONS, COMPLIANCE_STATUS_OPTIONS, RESOURCE_SET_OPTIONS, WORKLOAD_PROTOCOL_OPTIONS } from '@/data/reporting'
import type { LogType } from '@/data/reporting'

export function ReportingPage() {
  const [selectedLogType, setSelectedLogType] = useState<LogType>('audit_logs')
  const [searchQuery, setSearchQuery] = useState('')
  const [pageSize, setPageSize] = useState<number>(PAGE_SIZE_OPTIONS[0])
  const [currentPage, setCurrentPage] = useState(1)

  // Filter state
  const [showFilter, setShowFilter] = useState(false)
  const [timeRange, setTimeRange] = useState('1d')
  const [severity, setSeverity] = useState('')
  const [eventType, setEventType] = useState('')
  const [category, setCategory] = useState('')
  const [complianceStatus, setComplianceStatus] = useState('')
  const [resourceSet, setResourceSet] = useState('')
  const [appProtocol, setAppProtocol] = useState('')
  const [mcpUser, setMcpUser] = useState('')

  void currentPage

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
      <div className="flex items-center justify-between px-4 py-3 shrink-0 min-h-[36px] border-b border-border bg-white">
        <PageHeader />
        <div className="flex items-center gap-3">
          <div className="relative">
            <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
            <input
              type="text"
              placeholder="Search"
              value={searchQuery}
              onChange={(e) => { setSearchQuery(e.target.value); setCurrentPage(1) }}
              className="pl-9 pr-3 py-2 border border-border rounded-md text-sm w-48 focus:outline-none focus:ring-1 focus:ring-gray-300 focus:border-gray-400"
            />
          </div>
          <div className="relative">
            <button
              type="button"
              onClick={() => setShowFilter(!showFilter)}
              className={`flex items-center gap-1.5 px-4 py-2 border rounded-md text-sm font-medium transition-colors ${
                showFilter ? 'border-orange-400 text-orange-600 bg-orange-50' : 'border-border text-gray-600 hover:border-gray-400'
              }`}
            >
              <Filter size={14} />
              Filter
            </button>
            {showFilter && (
              <div className="absolute top-full right-0 mt-1 w-64 p-3 bg-white border border-border rounded-lg shadow-lg z-50">
                <div className="flex items-center justify-between mb-3">
                  <button type="button" onClick={handleResetFilters} className="text-[11px] text-orange-500 hover:text-orange-600 font-medium">Reset All</button>
                  <div className="flex items-center gap-2">
                    <button type="button" onClick={() => setShowFilter(false)} className="text-[11px] text-gray-500 hover:text-gray-700 font-medium">Cancel</button>
                    <button type="button" onClick={() => setShowFilter(false)} className="px-2.5 py-1 bg-orange-500 hover:bg-orange-600 text-white text-[11px] font-medium rounded-md transition-colors">Apply</button>
                  </div>
                </div>
                <div className="space-y-3">
                  {selectedLogType !== 'global_policy_compliance' && selectedLogType !== 'mcp_authorization_tracing' && (
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
            onChange={(v) => { setSelectedLogType(v); setCurrentPage(1) }}
          />
          <ResourceSetDropdown />
        </div>
      </div>

      {/* Table — full bleed */}
      <div className="flex-1 overflow-y-auto min-h-0">
        {selectedLogType === 'access_authorization' ? (
          <AccessAuthorizationTable searchQuery={searchQuery} timeRange={timeRange} severity={severity} eventType={eventType} />
        ) : selectedLogType === 'audit_logs' ? (
          <AuditLogsTable searchQuery={searchQuery} timeRange={timeRange} severity={severity} category={category} />
        ) : selectedLogType === 'global_policy_compliance' ? (
          <GlobalPolicyComplianceTable searchQuery={searchQuery} complianceStatus={complianceStatus} resourceSet={resourceSet} />
        ) : selectedLogType === 'mcp_authorization_tracing' ? (
          <McpAuthorizationTracingTable searchQuery={searchQuery} severity={severity} />
        ) : selectedLogType === 'workload_events' ? (
          <WorkloadEventsTable searchQuery={searchQuery} timeRange={timeRange} severity={severity} appProtocol={appProtocol} mcpUser={mcpUser} />
        ) : (
          <div className="flex items-center justify-center h-full">
            <p className="text-sm text-gray-400">Content coming soon</p>
          </div>
        )}
      </div>

      {/* Pagination footer */}
      <div className="flex items-center justify-between px-4 py-2 border-t border-border bg-gray-50 shrink-0">
        <div className="flex items-center gap-2 text-xs text-gray-600">
          <select value={pageSize} onChange={(e) => { setPageSize(Number(e.target.value)); setCurrentPage(1) }} className="border border-border rounded px-2 py-1 text-xs">
            {PAGE_SIZE_OPTIONS.map((size) => <option key={size} value={size}>{size}</option>)}
          </select>
          <span>Showing 0 - 0 of 0 Results</span>
        </div>
        <div className="flex items-center gap-1">
          <button disabled className="px-2 py-1 text-xs text-gray-500 hover:text-gray-700 disabled:opacity-30">←</button>
          <button className="w-7 h-7 rounded text-xs font-medium bg-orange-500 text-white">1</button>
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
        className="flex items-center gap-1.5 w-[200px] px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white text-sm font-medium rounded-md transition-colors"
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
              className={`w-full text-left px-4 py-2.5 text-sm transition-colors ${value === opt.value ? 'bg-orange-50 text-orange-600 font-medium' : 'text-gray-600 hover:bg-gray-50'}`}
            >
              {opt.label}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}

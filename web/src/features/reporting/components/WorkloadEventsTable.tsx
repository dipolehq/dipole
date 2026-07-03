import { useState } from 'react'
import { ChevronDown, RefreshCw } from 'lucide-react'
import { workloadEventsData } from '@/data/reporting/workloadEvents'
import type { WorkloadEvent } from '@/data/reporting/workloadEvents'

function formatTimestamp(iso: string): string {
  const d = new Date(iso)
  return d.toLocaleString('en-IN', {
    month: '2-digit', day: '2-digit', year: 'numeric',
    hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: true,
  })
}

function SeverityBadge({ severity }: { severity: string }) {
  const colors: Record<string, string> = {
    Info: 'text-green-600',
    Warning: 'text-orange-500',
    Error: 'text-red-500',
  }
  return (
    <span className={`text-xs font-medium ${colors[severity] ?? 'text-gray-600'}`}>
      {severity}
    </span>
  )
}

interface WorkloadEventsTableProps {
  searchQuery: string
  timeRange: string
  severity: string
  appProtocol: string
  mcpUser: string
}

export function WorkloadEventsTable({ searchQuery, severity, appProtocol, mcpUser }: WorkloadEventsTableProps) {
  const [expandedId, setExpandedId] = useState<string | null>(null)

  const filtered = workloadEventsData.filter((event) => {
    if (severity && event.meta.severity.toLowerCase() !== severity) return false
    if (appProtocol && !event.meta.eventType.startsWith(appProtocol)) return false
    if (mcpUser) {
      const mcpApp = event.application as Record<string, unknown>
      const mcp = mcpApp?.mcp as Record<string, unknown> | undefined
      const userId = (mcp?.userId as string) ?? ''
      if (!userId.toLowerCase().includes(mcpUser.toLowerCase())) return false
    }
    if (searchQuery) {
      const q = searchQuery.toLowerCase()
      const matches =
        event.meta.eventType.toLowerCase().includes(q) ||
        event.meta.connectionId.toLowerCase().includes(q) ||
        event.workload.client.name.toLowerCase().includes(q) ||
        event.workload.server.name.toLowerCase().includes(q) ||
        event.network.serverWorkloadHost.toLowerCase().includes(q) ||
        event.outcome.result.toLowerCase().includes(q) ||
        (event.outcome.reason?.toLowerCase().includes(q) ?? false)
      if (!matches) return false
    }
    return true
  })

  function toggleRow(eventId: string) {
    setExpandedId(expandedId === eventId ? null : eventId)
  }

  return (
    <table className="w-full text-xs table-fixed">
      <colgroup>
        <col className="w-[36px]" />
        <col className="w-[16%]" />
        <col className="w-[26%]" />
        <col className="w-[13%]" />
        <col className="w-[17%]" />
        <col className="w-[17%]" />
        <col className="w-[8%]" />
      </colgroup>
      <thead className="sticky top-0 bg-gray-50 z-10">
        <tr className="border-b border-border">
          <th className="px-2 py-3">
            <button type="button" className="text-gray-400 hover:text-orange-500 transition-colors" title="Refresh">
              <RefreshCw size={15} strokeWidth={2.5} />
            </button>
          </th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Timestamp</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Connection ID</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Event Type</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Client Workload</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Server Workload</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Severity</th>
        </tr>
      </thead>
      <tbody className="divide-y divide-border">
        {filtered.length === 0 && (
          <tr>
            <td colSpan={7} className="px-4 py-12 text-center text-gray-400 text-sm">
              No workload events found.
            </td>
          </tr>
        )}
        {filtered.map((event) => (
          <WorkloadEventRow
            key={event.meta.eventId}
            event={event}
            isExpanded={expandedId === event.meta.eventId}
            onToggle={() => toggleRow(event.meta.eventId)}
          />
        ))}
      </tbody>
    </table>
  )
}

function WorkloadEventRow({ event, isExpanded, onToggle }: { event: WorkloadEvent; isExpanded: boolean; onToggle: () => void }) {
  return (
    <>
      <tr
        className="hover:bg-gray-50 transition-colors cursor-pointer"
        onClick={onToggle}
      >
        <td className="px-2 py-3 text-center">
          <ChevronDown
            size={14}
            className={`text-gray-400 transition-transform ${isExpanded ? 'rotate-180' : ''}`}
          />
        </td>
        <td className="px-4 py-3 text-gray-700 whitespace-nowrap truncate">
          {formatTimestamp(event.meta.timestamp)}
        </td>
        <td className="px-4 py-3 text-gray-700 truncate">
          {event.meta.connectionId}
        </td>
        <td className="px-4 py-3 text-gray-700 truncate">
          {event.meta.eventType}
        </td>
        <td className="px-4 py-3 text-gray-700 truncate">
          {event.workload.client.name}
        </td>
        <td className="px-4 py-3 text-gray-700 truncate">
          {event.workload.server.name}
        </td>
        <td className="px-4 py-3">
          <SeverityBadge severity={event.meta.severity} />
        </td>
      </tr>
      {isExpanded && (
        <tr>
          <td colSpan={7} className="p-0">
            <ExpandedDetail event={event} />
          </td>
        </tr>
      )}
    </>
  )
}

function ExpandedDetail({ event }: { event: WorkloadEvent }) {
  const protocol = event.meta.eventType.split('.')[0] ?? ''

  return (
    <div className="grid grid-cols-[40%_60%] bg-gray-100 max-h-[550px]">
      {/* Left panel — summary */}
      <div className="p-5 border-r border-gray-200 overflow-y-auto max-h-[550px]">
        <table className="text-xs">
          <tbody>
            <tr className="h-7">
              <td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">Timestamp</td>
              <td className="text-gray-700">{event.meta.timestamp}</td>
            </tr>
            <tr className="h-7">
              <td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">Connection ID</td>
              <td className="text-gray-700 break-all">{event.meta.connectionId}</td>
            </tr>
            <tr className="h-7">
              <td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">Event ID</td>
              <td className="text-gray-700 break-all">{event.meta.eventId}</td>
            </tr>
            <tr className="h-7">
              <td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">Policy ID</td>
              <td className="text-gray-700 break-all">{event.meta.policyId || '—'}</td>
            </tr>
            <tr className="h-7">
              <td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">Event Type</td>
              <td className="text-gray-700">{event.meta.eventType}</td>
            </tr>
            <tr className="h-7">
              <td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">Application Protocol</td>
              <td className="text-gray-700">{protocol}</td>
            </tr>
            <tr className="h-8" />
            <tr className="h-7">
              <td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">Client Workload</td>
              <td className="text-gray-700">{event.workload.client.name}</td>
            </tr>
            <tr className="h-7">
              <td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">Client IP</td>
              <td className="text-gray-700">{event.network.clientWorkloadIP}</td>
            </tr>
            <tr className="h-8" />
            <tr className="h-7">
              <td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">Server Workload</td>
              <td className="text-gray-700">{event.workload.server.name}</td>
            </tr>
            <tr className="h-7">
              <td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">Server IP</td>
              <td className="text-gray-700">{event.network.serverWorkloadHost}</td>
            </tr>
            <tr className="h-8" />
            <tr className="h-7">
              <td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">Outcome</td>
              <td className="text-gray-700">{event.outcome.result}</td>
            </tr>
            {event.outcome.reason && (
              <tr className="h-7">
                <td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">Reason</td>
                <td className="text-gray-700">{event.outcome.reason}</td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
      {/* Right panel — JSON */}
      <div className="p-4 overflow-y-auto max-h-[550px]">
        <pre className="text-[11px] text-gray-700 font-mono whitespace-pre-wrap leading-relaxed">
          {JSON.stringify(event, null, 2)}
        </pre>
      </div>
    </div>
  )
}

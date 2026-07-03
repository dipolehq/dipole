import { useState } from 'react'
import { ChevronDown } from 'lucide-react'
import { accessAuthorizationEventsData } from '@/data/reporting/accessAuthorizationEvents'
import type { AccessAuthorizationEvent } from '@/data/reporting/accessAuthorizationEvents'

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

interface AccessAuthorizationTableProps {
  searchQuery: string
  timeRange: string
  severity: string
  eventType: string
}

export function AccessAuthorizationTable({ searchQuery, severity, eventType }: AccessAuthorizationTableProps) {
  const [expandedId, setExpandedId] = useState<string | null>(null)

  // Filter events
  const filtered = accessAuthorizationEventsData.filter((e) => {
    if (severity && e.meta.severity.toLowerCase() !== severity) return false
    if (eventType && e.meta.eventType !== eventType) return false
    if (searchQuery) {
      const q = searchQuery.toLowerCase()
      const matches =
        e.meta.clientIP.toLowerCase().includes(q) ||
        e.meta.contextId.toLowerCase().includes(q) ||
        (e.clientWorkload?.name.toLowerCase().includes(q) ?? false) ||
        (e.serverWorkload?.name.toLowerCase().includes(q) ?? false) ||
        e.meta.eventType.toLowerCase().includes(q)
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
        <col className="w-[17%]" />
        <col className="w-[10%]" />
        <col className="w-[28%]" />
        <col className="w-[16%]" />
        <col className="w-[18%]" />
        <col className="w-[8%]" />
      </colgroup>
      <thead className="sticky top-0 bg-gray-50 z-10">
        <tr className="border-b border-border">
          <th className="px-2 py-3" />
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Timestamp</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Client IP</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Context ID</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Client Workload</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Server Workload</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Severity</th>
        </tr>
      </thead>
      <tbody className="divide-y divide-border">
        {filtered.length === 0 && (
          <tr>
            <td colSpan={7} className="px-4 py-12 text-center text-gray-400 text-sm">
              No access authorization events found.
            </td>
          </tr>
        )}
        {filtered.map((event) => (
          <EventRow
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

function EventRow({ event, isExpanded, onToggle }: { event: AccessAuthorizationEvent; isExpanded: boolean; onToggle: () => void }) {
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
          {event.meta.clientIP}
        </td>
        <td className="px-4 py-3 text-gray-700 truncate">
          {event.meta.contextId}
        </td>
        <td className="px-4 py-3 text-gray-700 truncate">
          {event.clientWorkload?.name ?? '—'}
        </td>
        <td className="px-4 py-3 text-gray-700 truncate">
          {event.serverWorkload?.name ?? '—'}
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

function ExpandedDetail({ event }: { event: AccessAuthorizationEvent }) {
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
              <td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">Client IP</td>
              <td className="text-gray-700">{event.meta.clientIP}</td>
            </tr>
            <tr className="h-7">
              <td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">Context ID</td>
              <td className="text-gray-700 break-all">{event.meta.contextId}</td>
            </tr>
            <tr className="h-7">
              <td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">Event Type</td>
              <td className="text-gray-700">{event.meta.eventType}</td>
            </tr>
            <tr className="h-7">
              <td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">Client Workload</td>
              <td className="text-gray-700 break-all">{event.clientWorkload?.id ?? '—'}</td>
            </tr>
            <tr className="h-7">
              <td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">Server Workload</td>
              <td className="text-gray-700 break-all">{event.serverWorkload?.id ?? '—'}</td>
            </tr>
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



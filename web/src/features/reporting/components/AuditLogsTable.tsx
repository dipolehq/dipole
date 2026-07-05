import { useState } from 'react'
import { ChevronDown, RefreshCw } from 'lucide-react'
import { auditLogsData } from '@/data/reporting/auditLogs'
import type { AuditLog } from '@/data/reporting/auditLogs'

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
    Warn: 'text-orange-500',
    Alert: 'text-red-500',
  }
  return (
    <span className={`text-xs font-medium ${colors[severity] ?? 'text-gray-600'}`}>
      {severity}
    </span>
  )
}

function ResultBadge({ result }: { result: string }) {
  return (
    <span className="text-xs text-gray-700">
      {result}
    </span>
  )
}

interface AuditLogsTableProps {
  searchQuery: string
  timeRange: string
  severity: string
  category: string
}

export function AuditLogsTable({ searchQuery, severity, category }: AuditLogsTableProps) {
  const [expandedId, setExpandedId] = useState<string | null>(null)

  const filtered = auditLogsData.filter((log) => {
    if (severity && log.severity.toLowerCase() !== severity.toLowerCase()) return false
    if (category && log.category !== category) return false
    if (searchQuery) {
      const q = searchQuery.toLowerCase()
      const matches =
        log.actor.displayName.toLowerCase().includes(q) ||
        log.actor.email.toLowerCase().includes(q) ||
        log.actor.type.toLowerCase().includes(q) ||
        log.category.toLowerCase().includes(q) ||
        log.activity.toLowerCase().includes(q) ||
        log.target.toLowerCase().includes(q) ||
        log.outcome.result.toLowerCase().includes(q) ||
        log.client.ipAddress.toLowerCase().includes(q)
      if (!matches) return false
    }
    return true
  })

  function toggleRow(externalId: string) {
    setExpandedId(expandedId === externalId ? null : externalId)
  }

  return (
    <table className="w-full text-xs table-fixed">
      <colgroup>
        <col className="w-[4%]" />
        <col className="w-[17%]" />
        <col className="w-[13%]" />
        <col className="w-[12%]" />
        <col className="w-[20%]" />
        <col className="w-[19%]" />
        <col className="w-[8%]" />
        <col className="w-[8%]" />
      </colgroup>
      <thead className="sticky top-0 bg-gray-50 z-10">
        <tr className="border-b border-border">
          <th className="px-3 py-2.5 leading-none">
            <button type="button" className="text-gray-400 hover:text-orange-500 transition-colors" title="Refresh">
              <RefreshCw size={15} strokeWidth={2.5} />
            </button>
          </th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Timestamp</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Category</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Actor</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Activity</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Target</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Result</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Severity</th>
        </tr>
      </thead>
      <tbody className="divide-y divide-border">
        {filtered.length === 0 && (
          <tr>
            <td colSpan={8} className="px-4 py-12 text-center text-gray-400 text-sm">
              No audit logs found.
            </td>
          </tr>
        )}
        {filtered.map((log) => (
          <AuditLogRow
            key={log.externalId}
            log={log}
            isExpanded={expandedId === log.externalId}
            onToggle={() => toggleRow(log.externalId)}
          />
        ))}
      </tbody>
    </table>
  )
}

function AuditLogRow({ log, isExpanded, onToggle }: { log: AuditLog; isExpanded: boolean; onToggle: () => void }) {
  const actorDisplay = log.actor.type === 'System' ? 'System' : log.actor.displayName || log.actor.email || log.actor.userName

  return (
    <>
      <tr
        className="hover:bg-gray-50 transition-colors cursor-pointer"
        onClick={onToggle}
      >
        <td className="px-3 py-3 text-center">
          <ChevronDown
            size={14}
            className={`text-gray-400 transition-transform ${isExpanded ? 'rotate-180' : ''}`}
          />
        </td>
        <td className="px-4 py-3 text-gray-700 whitespace-nowrap truncate">
          {formatTimestamp(log.createdAt)}
        </td>
        <td className="px-4 py-3 text-gray-700 truncate">
          {log.category}
        </td>
        <td className="px-4 py-3 text-gray-700 truncate">
          {actorDisplay}
        </td>
        <td className="px-4 py-3 text-gray-700 truncate">
          {log.activity}
        </td>
        <td className="px-4 py-3 text-gray-700 truncate">
          {log.target}
        </td>
        <td className="px-4 py-3">
          <ResultBadge result={log.outcome.result} />
        </td>
        <td className="px-4 py-3">
          <SeverityBadge severity={log.severity} />
        </td>
      </tr>
      {isExpanded && (
        <tr>
          <td colSpan={8} className="p-0">
            <ExpandedDetail log={log} />
          </td>
        </tr>
      )}
    </>
  )
}

function ExpandedDetail({ log }: { log: AuditLog }) {
  const actorDisplay = log.actor.type === 'System' ? 'System' : log.actor.displayName || log.actor.email || log.actor.userName

  return (
    <div className="grid grid-cols-[40%_60%] bg-gray-100 max-h-[550px]">
      {/* Left panel — summary */}
      <div className="p-5 border-r border-gray-200 overflow-y-auto max-h-[550px]">
        <table className="text-xs">
          <tbody>
            <tr className="h-7">
              <td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">Timestamp</td>
              <td className="text-gray-700">{log.createdAt}</td>
            </tr>
            <tr className="h-7">
              <td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">Actor</td>
              <td className="text-gray-700">{actorDisplay}</td>
            </tr>
            <tr className="h-7">
              <td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">Category</td>
              <td className="text-gray-700">{log.category}</td>
            </tr>
            <tr className="h-7">
              <td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">Activity</td>
              <td className="text-gray-700">{log.activity}</td>
            </tr>
            <tr className="h-7">
              <td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">Target</td>
              <td className="text-gray-700 break-all">{log.target}</td>
            </tr>
            <tr className="h-7">
              <td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">Result</td>
              <td className="text-gray-700">{log.outcome.result}</td>
            </tr>
            <tr className="h-7">
              <td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">Client IP</td>
              <td className="text-gray-700">{log.client.ipAddress}</td>
            </tr>
            <tr className="h-7">
              <td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">Browser</td>
              <td className="text-gray-700">{log.client.userAgent.browser}</td>
            </tr>
            <tr className="h-7">
              <td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">Operating System</td>
              <td className="text-gray-700">{log.client.userAgent.operatingSystem}</td>
            </tr>
            <tr className="h-7">
              <td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">User Agent</td>
              <td className="text-gray-700 break-all">{log.client.userAgent.raw}</td>
            </tr>
          </tbody>
        </table>
      </div>
      {/* Right panel — JSON */}
      <div className="p-4 overflow-y-auto max-h-[550px]">
        <pre className="text-[11px] text-gray-700 font-mono whitespace-pre-wrap leading-relaxed">
          {JSON.stringify(log, null, 2)}
        </pre>
      </div>
    </div>
  )
}

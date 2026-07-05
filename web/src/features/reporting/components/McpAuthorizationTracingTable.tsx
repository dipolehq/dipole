import { Cpu, RefreshCw } from 'lucide-react'
import { mcpAuthorizationTracingData } from '@/data/reporting/mcpAuthorizationTracing'
import type { McpAuthorizationTracingEntry, McpPolicyStatus } from '@/data/reporting/mcpAuthorizationTracing'

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

interface McpAuthorizationTracingTableProps {
  searchQuery: string
  severity: string
}

export function McpAuthorizationTracingTable({ searchQuery, severity }: McpAuthorizationTracingTableProps) {
  const filtered = mcpAuthorizationTracingData.filter((entry) => {
    if (severity && entry.severity.toLowerCase() !== severity) return false
    if (searchQuery) {
      const q = searchQuery.toLowerCase()
      const matches =
        entry.clientIP.toLowerCase().includes(q) ||
        entry.accessPolicy.name.toLowerCase().includes(q) ||
        entry.clientWorkload.name.toLowerCase().includes(q) ||
        entry.serverWorkload.name.toLowerCase().includes(q) ||
        entry.redirectUri.toLowerCase().includes(q) ||
        entry.resourceServer.toLowerCase().includes(q) ||
        entry.severity.toLowerCase().includes(q)
      if (!matches) return false
    }
    return true
  })

  return (
    <table className="w-full text-xs table-fixed">
      <colgroup>
        <col className="w-[4%]" />
        <col className="w-[15%]" />
        <col className="w-[9%]" />
        <col className="w-[14%]" />
        <col className="w-[24%]" />
        <col className="w-[28%]" />
        <col className="w-[7%]" />
      </colgroup>
      <thead className="sticky top-0 bg-gray-50 z-10">
        <tr className="border-b border-border">
          <th className="px-3 py-2.5 leading-none h-[39.16px]">
            <button type="button" className="text-gray-400 hover:text-orange-500 transition-colors" title="Refresh">
              <RefreshCw size={15} strokeWidth={2.5} />
            </button>
          </th>
          <th className="px-4 py-2.5 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Timestamp</th>
          <th className="px-4 py-2.5 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Client IP</th>
          <th className="px-4 py-2.5 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Access Policy</th>
          <th className="px-4 py-2.5 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Redirect URI / Client WL</th>
          <th className="px-4 py-2.5 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Resource Server / Server WL</th>
          <th className="px-4 py-2.5 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Severity</th>
        </tr>
      </thead>
      <tbody className="divide-y divide-border">
        {filtered.length === 0 && (
          <tr>
            <td colSpan={7} className="px-4 py-12 text-center text-gray-400 text-sm">
              No MCP authorization tracing entries found.
            </td>
          </tr>
        )}
        {filtered.map((entry) => (
          <McpTracingRow key={entry.id} entry={entry} />
        ))}
      </tbody>
    </table>
  )
}

function McpTracingRow({ entry }: { entry: McpAuthorizationTracingEntry }) {
  return (
    <tr className="hover:bg-gray-50 transition-colors align-top">
      <td className="px-3 py-2 text-center">
        <Cpu size={14} className="text-gray-400" />
      </td>
      <td className="px-4 py-2 text-gray-700 whitespace-nowrap">
        {formatTimestamp(entry.timestamp)}
      </td>
      <td className="px-4 py-2 text-gray-700">
        {entry.clientIP}
      </td>
      <td className="px-4 py-2">
        <PolicyCell name={entry.accessPolicy.name} status={entry.accessPolicy.status} />
      </td>
      <td className="px-4 py-2">
        <WorkloadCell
          name={entry.clientWorkload.name}
          status={entry.clientWorkload.status}
          uri={entry.redirectUri}
        />
      </td>
      <td className="px-4 py-2">
        <WorkloadCell
          name={entry.serverWorkload.name}
          status={entry.serverWorkload.status}
          uri={entry.resourceServer}
        />
      </td>
      <td className="px-4 py-2">
        <SeverityBadge severity={entry.severity} />
      </td>
    </tr>
  )
}

function PolicyCell({ name, status }: { name: string; status: McpPolicyStatus }) {
  if (status === 'Unknown') {
    return <span className="text-xs text-orange-500 font-medium">Unknown</span>
  }
  const statusColor = status === 'Active' ? 'text-green-600' : 'text-gray-400'
  return (
    <div>
      <div className="text-xs text-gray-700 font-medium">{name}</div>
      <span className={`text-[11px] font-medium ${statusColor}`}>{status}</span>
    </div>
  )
}

function WorkloadCell({ name, status, uri }: { name: string; status: 'Identified' | 'Unknown'; uri: string }) {
  return (
    <div>
      {status === 'Identified' ? (
        <div className="text-xs text-gray-700 font-medium">{name}</div>
      ) : (
        <div className="text-xs text-orange-500 font-medium">Unknown</div>
      )}
      <div className="text-[11px] text-gray-500 break-all leading-relaxed">{uri}</div>
    </div>
  )
}

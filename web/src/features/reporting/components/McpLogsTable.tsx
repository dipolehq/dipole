import { useState } from 'react'
import { ChevronDown, RefreshCw } from 'lucide-react'

interface McpLogEvent {
  id: string
  timestamp: string
  clientIp: string
  severity: string
  contextId: string
  payload: Record<string, unknown>
  // Extracted fields for display
  user: string
  client: string
  mcpServer: string
  toolCalled: string
  outcome: string
}

function formatTimestamp(iso: string): string {
  const d = new Date(iso)
  return d.toLocaleString('en-IN', {
    month: '2-digit',
    day: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: true,
  })
}

function SeverityBadge({ severity }: { severity: string }) {
  const colors: Record<string, string> = {
    Info: 'text-green-600',
    Warning: 'text-gray-600',
    Error: 'text-red-500',
  }
  return (
    <span className={`text-xs font-medium ${colors[severity] ?? 'text-gray-600'}`}>
      {severity}
    </span>
  )
}

function OutcomeBadge({ outcome }: { outcome: string }) {
  if (outcome === 'Authorized') {
    return <span className="text-xs font-medium text-green-600">Authorized</span>
  }
  if (outcome === 'Unauthorized') {
    return <span className="text-xs font-medium text-red-500">Unauthorized</span>
  }
  return <span className="text-xs font-medium text-gray-600">{outcome || '—'}</span>
}

function extractMcpFields(item: {
  id: string
  eventType: string
  contextId: string
  severity: string
  clientIp: string
  timestamp: string
  payload: Record<string, unknown>
}): McpLogEvent {
  const payload = item.payload ?? {}
  const clientWorkload = payload.clientWorkload as Record<string, string> | undefined
  const serverWorkload = payload.serverWorkload as Record<string, string> | undefined
  const outcome = payload.outcome as Record<string, string> | undefined
  const credentialProvider = payload.credentialProvider as Record<string, string> | undefined

  // User: from payload.user.email if available, otherwise clientIP
  const userObj = payload.user as Record<string, string> | undefined
  const user = userObj?.email || item.clientIp || '—'

  // Client: clientWorkload name (Claude Desktop, Gemini CLI, etc.)
  const client = clientWorkload?.name || '—'

  // MCP Server: serverWorkload name
  const mcpServer = serverWorkload?.name || '—'

  // Tool Called: from payload.toolCalled if present, otherwise credential provider type
  const toolCalled = (payload.toolCalled as string) || credentialProvider?.type || '—'

  // Outcome: from outcome.result
  const outcomeResult = outcome?.result || '—'

  return {
    id: item.id,
    timestamp: item.timestamp,
    clientIp: item.clientIp,
    severity: item.severity,
    contextId: item.contextId,
    payload,
    user,
    client,
    mcpServer,
    toolCalled,
    outcome: outcomeResult,
  }
}

interface McpLogsTableProps {
  searchQuery: string
  severity: string
  data?: {
    id: string
    eventType: string
    contextId: string
    severity: string
    clientIp: string
    timestamp: string
    payload: Record<string, unknown>
  }[]
  onRefresh?: () => void
}

export function McpLogsTable({ searchQuery, severity, data = [], onRefresh }: McpLogsTableProps) {
  const [expandedId, setExpandedId] = useState<string | null>(null)

  const events: McpLogEvent[] = data.map(extractMcpFields)

  const filtered = events.filter((e) => {
    if (severity && e.severity.toLowerCase() !== severity.toLowerCase()) return false
    if (searchQuery) {
      const q = searchQuery.toLowerCase()
      const matches =
        e.user.toLowerCase().includes(q) ||
        e.client.toLowerCase().includes(q) ||
        e.mcpServer.toLowerCase().includes(q) ||
        e.outcome.toLowerCase().includes(q) ||
        e.clientIp.toLowerCase().includes(q) ||
        e.contextId.toLowerCase().includes(q)
      if (!matches) return false
    }
    return true
  })

  function toggleRow(id: string) {
    setExpandedId(expandedId === id ? null : id)
  }

  return (
    <table className="w-full text-xs table-fixed">
      <colgroup>
        <col className="w-[3%]" />
        <col className="w-[16%]" />
        <col className="w-[16%]" />
        <col className="w-[16%]" />
        <col className="w-[18%]" />
        <col className="w-[13%]" />
        <col className="w-[10%]" />
        <col className="w-[8%]" />
      </colgroup>
      <thead className="sticky top-0 bg-gray-50 z-10">
        <tr className="border-b border-border">
          <th className="px-3 py-2.5 leading-none">
            <button
              type="button"
              onClick={onRefresh}
              className="text-gray-400 hover:text-gray-600 transition-colors"
              title="Refresh"
            >
              <RefreshCw size={15} strokeWidth={2.5} />
            </button>
          </th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">
            Timestamp
          </th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">
            User
          </th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">
            MCP Client
          </th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">
            MCP Server
          </th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">
            Tool Called
          </th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">
            Outcome
          </th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">
            Severity
          </th>
        </tr>
      </thead>
      <tbody className="divide-y divide-border">
        {filtered.length === 0 && (
          <tr>
            <td colSpan={8} className="px-4 py-12 text-center text-gray-400 text-sm">
              No MCP access logs found.
            </td>
          </tr>
        )}
        {filtered.map((event) => (
          <McpLogRow
            key={event.id}
            event={event}
            isExpanded={expandedId === event.id}
            onToggle={() => toggleRow(event.id)}
          />
        ))}
      </tbody>
    </table>
  )
}

function McpLogRow({
  event,
  isExpanded,
  onToggle,
}: {
  event: McpLogEvent
  isExpanded: boolean
  onToggle: () => void
}) {
  return (
    <>
      <tr className="hover:bg-gray-50 transition-colors cursor-pointer" onClick={onToggle}>
        <td className="px-4 py-3 text-center">
          <ChevronDown
            size={14}
            className={`text-gray-400 transition-transform ${isExpanded ? 'rotate-180' : ''}`}
          />
        </td>
        <td className="px-4 py-3 text-gray-700 whitespace-nowrap truncate">
          {formatTimestamp(event.timestamp)}
        </td>
        <td className="px-4 py-3 text-gray-700 truncate" title={event.user}>
          {event.user}
        </td>
        <td className="px-4 py-3 text-gray-700 truncate" title={event.client}>
          {event.client}
        </td>
        <td className="px-4 py-3 text-gray-700 truncate" title={event.mcpServer}>
          {event.mcpServer}
        </td>
        <td className="px-4 py-3 text-gray-700 truncate" title={event.toolCalled}>
          {event.toolCalled}
        </td>
        <td className="px-4 py-3">
          <OutcomeBadge outcome={event.outcome} />
        </td>
        <td className="px-4 py-3">
          <SeverityBadge severity={event.severity} />
        </td>
      </tr>
      {isExpanded && (
        <tr>
          <td colSpan={8} className="p-0">
            <McpLogExpandedDetail event={event} />
          </td>
        </tr>
      )}
    </>
  )
}

function McpLogExpandedDetail({ event }: { event: McpLogEvent }) {
  const payload = event.payload
  const accessPolicy = payload.accessPolicy as Record<string, string> | undefined
  const trustProviders = payload.trustProviders as Record<string, string>[] | undefined
  const credentialProvider = payload.credentialProvider as Record<string, string> | undefined
  const outcome = payload.outcome as Record<string, string> | undefined

  return (
    <div className="grid grid-cols-[40%_60%] bg-gray-100 max-h-[550px]">
      {/* Left panel — summary */}
      <div className="p-5 border-r border-gray-200 overflow-y-auto max-h-[550px]">
        <table className="text-xs">
          <tbody>
            <tr className="h-7">
              <td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">
                Timestamp
              </td>
              <td className="text-gray-700">{event.timestamp}</td>
            </tr>
            <tr className="h-7">
              <td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">
                User
              </td>
              <td className="text-gray-700">{event.user}</td>
            </tr>
            <tr className="h-7">
              <td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">
                MCP Client
              </td>
              <td className="text-gray-700">{event.client}</td>
            </tr>
            <tr className="h-7">
              <td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">
                MCP Server
              </td>
              <td className="text-gray-700">{event.mcpServer}</td>
            </tr>
            <tr className="h-7">
              <td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">
                Client IP
              </td>
              <td className="text-gray-700">{event.clientIp}</td>
            </tr>
            <tr className="h-7">
              <td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">
                Context ID
              </td>
              <td className="text-gray-700 break-all">{event.contextId}</td>
            </tr>
            <tr className="h-7">
              <td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">
                Access Policy
              </td>
              <td className="text-gray-700">{accessPolicy?.name || '—'}</td>
            </tr>
            <tr className="h-7">
              <td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">
                Trust Provider
              </td>
              <td className="text-gray-700">
                {trustProviders?.map((tp) => tp.name).join(', ') || '—'}
              </td>
            </tr>
            <tr className="h-7">
              <td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">
                Credential Provider
              </td>
              <td className="text-gray-700">{credentialProvider?.name || '—'}</td>
            </tr>
            <tr className="h-7">
              <td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">
                Outcome
              </td>
              <td className="text-gray-700">
                {outcome?.result || '—'}
                {outcome?.reason ? ` (${outcome.reason})` : ''}
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      {/* Right panel — JSON */}
      <div className="p-4 overflow-y-auto max-h-[550px]">
        <pre className="text-[11px] text-gray-700 font-mono whitespace-pre-wrap leading-relaxed">
          {JSON.stringify(event.payload, null, 2)}
        </pre>
      </div>
    </div>
  )
}

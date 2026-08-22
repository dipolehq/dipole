import { useState, useMemo } from 'react'
import { useQuery, useQueryClient, keepPreviousData } from '@tanstack/react-query'
import { type ColumnDef } from '@tanstack/react-table'
import { LogTable } from '@/shared/components/ui/LogTable'
import { fetchAuthEvents } from '@/features/reporting/api/eventsApi'

const PAGE_SIZE = 50

interface McpLogRow {
  id: string
  timestamp: string
  user: string
  client: string
  mcpServer: string
  toolCalled: string
  outcome: string
  severity: string
  payload: Record<string, unknown>
}

function formatTimestamp(iso: string): string {
  const d = new Date(iso)
  return d.toLocaleString('en-IN', { month: '2-digit', day: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: true })
}

function SeverityBadge({ severity }: { severity: string }) {
  const colors: Record<string, string> = { Info: 'text-green-600', Warning: 'text-gray-600', Error: 'text-red-500' }
  return <span className={`text-xs font-medium ${colors[severity] ?? 'text-gray-600'}`}>{severity}</span>
}

function OutcomeBadge({ outcome }: { outcome: string }) {
  if (outcome === 'Authorized') return <span className="text-xs font-medium text-green-600">Authorized</span>
  if (outcome === 'Unauthorized') return <span className="text-xs font-medium text-red-500">Unauthorized</span>
  return <span className="text-xs font-medium text-gray-600">{outcome || '—'}</span>
}

function ExpandedDetail({ row }: { row: McpLogRow }) {
  return (
    <div className="grid grid-cols-[40%_60%] bg-gray-100 max-h-[550px]">
      <div className="p-5 border-r border-gray-200 overflow-y-auto max-h-[550px]">
        <table className="text-xs">
          <tbody>
            <tr className="h-7"><td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">Timestamp</td><td className="text-gray-700">{row.timestamp}</td></tr>
            <tr className="h-7"><td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">User</td><td className="text-gray-700">{row.user}</td></tr>
            <tr className="h-7"><td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">MCP Client</td><td className="text-gray-700">{row.client}</td></tr>
            <tr className="h-7"><td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">MCP Server</td><td className="text-gray-700">{row.mcpServer}</td></tr>
            <tr className="h-7"><td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">Tool Called</td><td className="text-gray-700">{row.toolCalled}</td></tr>
            <tr className="h-7"><td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">Outcome</td><td className="text-gray-700">{row.outcome}</td></tr>
            <tr className="h-7"><td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">Severity</td><td className="text-gray-700">{row.severity}</td></tr>
          </tbody>
        </table>
      </div>
      <div className="p-4 overflow-y-auto max-h-[550px]">
        <pre className="text-[11px] text-gray-700 font-mono whitespace-pre-wrap leading-relaxed">
          {JSON.stringify(row.payload, null, 2)}
        </pre>
      </div>
    </div>
  )
}

export function ReportingMcpLogsPage() {
  const queryClient = useQueryClient()
  const [page, setPage] = useState(1)
  const [search, setSearch] = useState('')

  const { data } = useQuery({
    queryKey: ['events', 'mcp-logs', page, PAGE_SIZE, search],
    queryFn: () => fetchAuthEvents({ page, perPage: PAGE_SIZE, search, severity: '', eventType: 'access.authorization', source: 'mcp_gateway' }),
    placeholderData: keepPreviousData,
  })

  const rows: McpLogRow[] = useMemo(() => (data?.items ?? []).map((item) => {
    const payload = item.payload ?? {}
    const userObj = payload.user as Record<string, string> | undefined
    const clientWorkload = payload.clientWorkload as Record<string, string> | undefined
    const serverWorkload = payload.serverWorkload as Record<string, string> | undefined
    const outcome = payload.outcome as Record<string, string> | undefined
    const credentialProvider = payload.credentialProvider as Record<string, string> | undefined

    return {
      id: item.id,
      timestamp: item.timestamp,
      user: userObj?.email || item.clientIp || '—',
      client: clientWorkload?.name || '—',
      mcpServer: serverWorkload?.name || '—',
      toolCalled: (payload.toolCalled as string) || credentialProvider?.type || '—',
      outcome: outcome?.result || '—',
      severity: item.severity,
      payload,
    }
  }), [data])

  const columns: ColumnDef<McpLogRow, unknown>[] = useMemo(() => [
    { accessorKey: 'timestamp', header: 'Timestamp', cell: ({ row }) => <span className="text-gray-700 whitespace-nowrap">{formatTimestamp(row.getValue('timestamp'))}</span>, size: 200 },
    { accessorKey: 'user', header: 'User', cell: ({ row }) => <span className="text-gray-700 truncate block">{row.getValue('user')}</span> },
    { accessorKey: 'client', header: 'MCP Client', cell: ({ row }) => <span className="text-gray-700 truncate block">{row.getValue('client')}</span> },
    { accessorKey: 'mcpServer', header: 'MCP Server', cell: ({ row }) => <span className="text-gray-700 truncate block">{row.getValue('mcpServer')}</span> },
    { accessorKey: 'toolCalled', header: 'Tool Called', cell: ({ row }) => <span className="text-gray-700 truncate block">{row.getValue('toolCalled')}</span> },
    { accessorKey: 'outcome', header: 'Outcome', cell: ({ row }) => <OutcomeBadge outcome={row.getValue('outcome')} /> },
    { accessorKey: 'severity', header: 'Severity', cell: ({ row }) => <SeverityBadge severity={row.getValue('severity')} />, size: 80 },
  ], [])

  return (
    <div className="flex flex-col h-full">
      <div className="flex-1 min-h-0">
        <LogTable
          columns={columns}
          data={rows}
          total={data?.total ?? 0}
          page={page}
          pageSize={PAGE_SIZE}
          onPageChange={setPage}
          onSearch={(q) => { setSearch(q); setPage(1) }}
          searchValue={search}
          searchPlaceholder="Filter by user, client, or server..."
          searchKeys={['user', 'client', 'mcpServer', 'toolCalled', 'outcome']}
          renderExpanded={(row) => <ExpandedDetail row={row} />}
          onRefresh={() => queryClient.invalidateQueries({ queryKey: ['events', 'mcp-logs'] })}
          emptyMessage="No MCP access logs found."
        />
      </div>
    </div>
  )
}

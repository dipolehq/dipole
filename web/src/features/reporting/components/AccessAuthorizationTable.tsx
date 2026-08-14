import { useMemo } from 'react'
import { type ColumnDef } from '@tanstack/react-table'
import { LogTable } from '@/shared/components/ui/LogTable'

interface AuthEventRow {
  id: string
  timestamp: string
  clientIp: string
  contextId: string
  clientWorkloadName: string
  serverWorkloadName: string
  severity: string
  eventType: string
  payload: Record<string, unknown>
}

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
    Warning: 'text-gray-600',
    Error: 'text-red-500',
  }
  return <span className={`text-xs font-medium ${colors[severity] ?? 'text-gray-600'}`}>{severity}</span>
}

function ExpandedDetail({ row }: { row: AuthEventRow }) {
  return (
    <div className="grid grid-cols-[40%_60%] bg-gray-100 max-h-[550px]">
      <div className="p-5 border-r border-gray-200 overflow-y-auto max-h-[550px]">
        <table className="text-xs">
          <tbody>
            <tr className="h-7"><td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">Timestamp</td><td className="text-gray-700">{row.timestamp}</td></tr>
            <tr className="h-7"><td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">Client IP</td><td className="text-gray-700">{row.clientIp}</td></tr>
            <tr className="h-7"><td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">Context ID</td><td className="text-gray-700 break-all">{row.contextId}</td></tr>
            <tr className="h-7"><td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">Event Type</td><td className="text-gray-700">{row.eventType}</td></tr>
            <tr className="h-7"><td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">Client Workload</td><td className="text-gray-700">{row.clientWorkloadName || '—'}</td></tr>
            <tr className="h-7"><td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">Server Workload</td><td className="text-gray-700">{row.serverWorkloadName || '—'}</td></tr>
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

interface AccessAuthorizationTableProps {
  data?: { id: string; eventType: string; contextId: string; severity: string; clientIp: string; timestamp: string; payload: Record<string, unknown>; clientWorkloadName?: string; serverWorkloadName?: string }[]
  total: number
  page: number
  pageSize: number
  onPageChange: (page: number) => void
  onSearch?: (query: string) => void
  searchValue?: string
  onRefresh?: () => void
  severity?: string
  onSeverityChange?: (value: string) => void
}

export function AccessAuthorizationTable({ data = [], total, page, pageSize, onPageChange, onSearch, searchValue, onRefresh, severity, onSeverityChange }: AccessAuthorizationTableProps) {
  const rows: AuthEventRow[] = useMemo(() => data.map((item) => ({
    id: item.id,
    timestamp: item.timestamp,
    clientIp: item.clientIp,
    contextId: item.contextId,
    clientWorkloadName: item.clientWorkloadName ?? (item.payload?.clientWorkload as Record<string, string>)?.name ?? '',
    serverWorkloadName: item.serverWorkloadName ?? (item.payload?.serverWorkload as Record<string, string>)?.name ?? '',
    severity: item.severity,
    eventType: item.eventType,
    payload: item.payload,
  })), [data])

  const columns: ColumnDef<AuthEventRow, unknown>[] = useMemo(() => [
    {
      accessorKey: 'timestamp',
      header: 'Timestamp',
      cell: ({ row }) => <span className="text-gray-700 whitespace-nowrap truncate block">{formatTimestamp(row.getValue('timestamp'))}</span>,
      size: 200,
    },
    {
      accessorKey: 'clientIp',
      header: 'Client IP',
      cell: ({ row }) => <span className="text-gray-700 truncate block">{row.getValue('clientIp')}</span>,
      
    },
    {
      accessorKey: 'contextId',
      header: 'Context ID',
      cell: ({ row }) => <span className="text-gray-700 font-mono text-[11px] whitespace-nowrap">{row.getValue('contextId')}</span>,
      size: 300,
      
    },
    {
      accessorKey: 'clientWorkloadName',
      header: 'Client Workload',
      cell: ({ row }) => <span className="text-gray-700 truncate block">{row.getValue('clientWorkloadName') || '—'}</span>,
      
    },
    {
      accessorKey: 'serverWorkloadName',
      header: 'Server Workload',
      cell: ({ row }) => <span className="text-gray-700 truncate block">{row.getValue('serverWorkloadName') || '—'}</span>,
      
    },
    {
      accessorKey: 'severity',
      header: 'Severity',
      cell: ({ row }) => <SeverityBadge severity={row.getValue('severity')} />,
      size: 80,
    },
  ], [])

  return (
    <LogTable
      columns={columns}
      data={rows}
      total={total}
      page={page}
      pageSize={pageSize}
      onPageChange={onPageChange}
      onSearch={onSearch}
      searchValue={searchValue}
      searchPlaceholder="Filter by IP, context, or workload..."
      severity={severity}
      onSeverityChange={onSeverityChange}
      renderExpanded={(row) => <ExpandedDetail row={row} />}
      onRefresh={onRefresh}
      emptyMessage="No access authorization events found."
    />
  )
}

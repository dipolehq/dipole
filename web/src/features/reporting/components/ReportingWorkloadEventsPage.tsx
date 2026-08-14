import { useState, useMemo } from 'react'
import { useQuery, useQueryClient, keepPreviousData } from '@tanstack/react-query'
import { type ColumnDef } from '@tanstack/react-table'
import { PageHeader } from '@/shared/components/PageHeader'
import { LogTable } from '@/shared/components/ui/LogTable'
import { fetchWorkloadEvents } from '@/features/reporting/api/eventsApi'

const PAGE_SIZE = 50

interface WorkloadEventRow {
  id: string
  timestamp: string
  connectionId: string
  eventType: string
  clientWorkloadName: string
  serverWorkloadName: string
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

function ExpandedDetail({ row }: { row: WorkloadEventRow }) {
  return (
    <div className="grid grid-cols-[40%_60%] bg-gray-100 max-h-[550px]">
      <div className="p-5 border-r border-gray-200 overflow-y-auto max-h-[550px]">
        <table className="text-xs">
          <tbody>
            <tr className="h-7"><td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">Timestamp</td><td className="text-gray-700">{row.timestamp}</td></tr>
            <tr className="h-7"><td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">Connection ID</td><td className="text-gray-700 break-all">{row.connectionId}</td></tr>
            <tr className="h-7"><td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">Event Type</td><td className="text-gray-700">{row.eventType}</td></tr>
            <tr className="h-7"><td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">Client Workload</td><td className="text-gray-700">{row.clientWorkloadName || '—'}</td></tr>
            <tr className="h-7"><td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">Server Workload</td><td className="text-gray-700">{row.serverWorkloadName || '—'}</td></tr>
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

export function ReportingWorkloadEventsPage() {
  const queryClient = useQueryClient()
  const [page, setPage] = useState(1)
  const [search, setSearch] = useState('')
  const [severity, setSeverity] = useState('')

  const { data } = useQuery({
    queryKey: ['events', 'workload', page, PAGE_SIZE, severity],
    queryFn: () => fetchWorkloadEvents({ page, perPage: PAGE_SIZE, severity, eventType: '' }),
    placeholderData: keepPreviousData,
  })

  const rows: WorkloadEventRow[] = useMemo(() => (data?.items ?? []).map((item) => ({
    id: item.id,
    timestamp: item.timestamp,
    connectionId: item.connectionId ?? '',
    eventType: item.eventType ?? '',
    clientWorkloadName: item.clientWorkloadName ?? '',
    serverWorkloadName: item.serverWorkloadName ?? '',
    severity: item.severity ?? '',
    payload: item.payload ?? {},
  })), [data])

  const columns: ColumnDef<WorkloadEventRow, unknown>[] = useMemo(() => [
    { accessorKey: 'timestamp', header: 'Timestamp', cell: ({ row }) => <span className="text-gray-700 whitespace-nowrap">{formatTimestamp(row.getValue('timestamp'))}</span>, size: 200 },
    { accessorKey: 'connectionId', header: 'Connection ID', cell: ({ row }) => <span className="text-gray-700 font-mono text-[11px] whitespace-nowrap">{row.getValue('connectionId')}</span>, size: 300 },
    { accessorKey: 'eventType', header: 'Event Type', cell: ({ row }) => <span className="text-gray-700 truncate block">{row.getValue('eventType')}</span> },
    { accessorKey: 'clientWorkloadName', header: 'Client Workload', cell: ({ row }) => <span className="text-gray-700 truncate block">{row.getValue('clientWorkloadName') || '—'}</span> },
    { accessorKey: 'serverWorkloadName', header: 'Server Workload', cell: ({ row }) => <span className="text-gray-700 truncate block">{row.getValue('serverWorkloadName') || '—'}</span> },
    { accessorKey: 'severity', header: 'Severity', cell: ({ row }) => <SeverityBadge severity={row.getValue('severity')} />, size: 80 },
  ], [])

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-4 shrink-0 h-[44px] border-b border-border bg-white">
        <PageHeader />
      </div>
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
          searchPlaceholder="Filter by connection or workload..."
          severity={severity}
          onSeverityChange={(v) => { setSeverity(v); setPage(1) }}
          renderExpanded={(row) => <ExpandedDetail row={row} />}
          onRefresh={() => queryClient.invalidateQueries({ queryKey: ['events', 'workload'] })}
          emptyMessage="No workload events found."
        />
      </div>
    </div>
  )
}

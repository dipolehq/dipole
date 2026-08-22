import { useState, useMemo } from 'react'
import { useQuery, useQueryClient, keepPreviousData } from '@tanstack/react-query'
import { type ColumnDef } from '@tanstack/react-table'
import { LogTable } from '@/shared/components/ui/LogTable'
import { fetchAuditLogs } from '@/features/reporting/api/eventsApi'

const PAGE_SIZE = 50

interface AuditRow {
  id: string
  timestamp: string
  category: string
  actor: string
  activity: string
  target: string
  severity: string
}

function formatTimestamp(iso: string): string {
  const d = new Date(iso)
  return d.toLocaleString('en-IN', { month: '2-digit', day: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: true })
}

function SeverityBadge({ severity }: { severity: string }) {
  const colors: Record<string, string> = { Info: 'text-green-600', Warn: 'text-gray-600', Alert: 'text-red-500' }
  return <span className={`text-xs font-medium ${colors[severity] ?? 'text-gray-600'}`}>{severity}</span>
}

function ExpandedDetail({ row }: { row: AuditRow }) {
  return (
    <div className="grid grid-cols-[40%_60%] bg-gray-100 max-h-[550px]">
      <div className="p-5 border-r border-gray-200 overflow-y-auto max-h-[550px]">
        <table className="text-xs">
          <tbody>
            <tr className="h-7"><td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">Timestamp</td><td className="text-gray-700">{row.timestamp}</td></tr>
            <tr className="h-7"><td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">Category</td><td className="text-gray-700">{row.category}</td></tr>
            <tr className="h-7"><td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">Actor</td><td className="text-gray-700">{row.actor}</td></tr>
            <tr className="h-7"><td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">Activity</td><td className="text-gray-700">{row.activity}</td></tr>
            <tr className="h-7"><td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">Target</td><td className="text-gray-700">{row.target}</td></tr>
            <tr className="h-7"><td className="font-semibold text-gray-900 pr-6 whitespace-nowrap align-middle">Severity</td><td className="text-gray-700">{row.severity}</td></tr>
          </tbody>
        </table>
      </div>
      <div className="p-4 overflow-y-auto max-h-[550px]">
        <pre className="text-[11px] text-gray-700 font-mono whitespace-pre-wrap leading-relaxed">
          {JSON.stringify(row, null, 2)}
        </pre>
      </div>
    </div>
  )
}

export function ReportingAuditPage() {
  const queryClient = useQueryClient()
  const [page, setPage] = useState(1)
  const [search, setSearch] = useState('')
  const [severity, setSeverity] = useState('')

  const { data } = useQuery({
    queryKey: ['events', 'audit', page, PAGE_SIZE, severity, search],
    queryFn: () => fetchAuditLogs({ page, perPage: PAGE_SIZE, severity, search, eventType: '' }),
    placeholderData: keepPreviousData,
  })

  const rows: AuditRow[] = useMemo(() => (data?.items ?? []).map((item) => ({
    id: item.id,
    timestamp: item.timestamp,
    category: item.category ?? '',
    actor: item.actorEmail ?? '',
    activity: item.activity ?? '',
    target: item.targetName ?? '',
    severity: item.severity ?? '',
  })), [data])

  const columns: ColumnDef<AuditRow, unknown>[] = useMemo(() => [
    { accessorKey: 'timestamp', header: 'Timestamp', cell: ({ row }) => <span className="text-gray-700 whitespace-nowrap">{formatTimestamp(row.getValue('timestamp'))}</span>, size: 200 },
    { accessorKey: 'category', header: 'Category', cell: ({ row }) => <span className="text-gray-700 truncate block">{row.getValue('category')}</span> },
    { accessorKey: 'actor', header: 'Actor', cell: ({ row }) => <span className="text-gray-700 truncate block">{row.getValue('actor')}</span> },
    { accessorKey: 'activity', header: 'Activity', cell: ({ row }) => <span className="text-gray-700 truncate block">{row.getValue('activity')}</span> },
    { accessorKey: 'target', header: 'Target', cell: ({ row }) => <span className="text-gray-700 truncate block">{row.getValue('target')}</span> },
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
          searchPlaceholder="Filter by actor or activity..."
          searchKeys={['category', 'actor', 'activity', 'target']}
          severity={severity}
          onSeverityChange={(v) => { setSeverity(v); setPage(1) }}
          renderExpanded={(row) => <ExpandedDetail row={row} />}
          onRefresh={() => queryClient.invalidateQueries({ queryKey: ['events', 'audit'] })}
          emptyMessage="No audit logs found."
        />
      </div>
    </div>
  )
}

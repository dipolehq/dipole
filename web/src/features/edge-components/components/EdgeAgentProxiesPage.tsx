import { useMemo } from 'react'
import { type ColumnDef } from '@tanstack/react-table'
import { PageHeader } from '@/shared/components/PageHeader'
import { DataTable, type FacetedFilterConfig } from '@/shared/components/ui/DataTable'
import { useAgentProxies } from '@/features/edge-components/hooks/useAgentProxies'

interface ProxyRow {
  id: string
  proxyId: string
  controllerId: string
  lastSeen: string
  registered: string
  status: string
}

const statusFilterFn = (row: { getValue: (id: string) => unknown }, _columnId: string, filterValue: string[]) => {
  if (!filterValue?.length) return true
  return filterValue.includes(row.getValue('status') as string)
}

function formatTime(iso: string): string {
  if (!iso) return '—'
  const d = new Date(iso)
  const now = new Date()
  const diffMs = now.getTime() - d.getTime()
  const diffSec = Math.floor(diffMs / 1000)
  if (diffSec < 60) return `${diffSec}s ago`
  const diffMin = Math.floor(diffSec / 60)
  if (diffMin < 60) return `${diffMin}m ago`
  const diffHr = Math.floor(diffMin / 60)
  if (diffHr < 24) return `${diffHr}h ago`
  return d.toLocaleDateString('en-GB', { day: '2-digit', month: '2-digit', year: 'numeric' })
}

function StatusBadge({ status }: { status: string }) {
  const colors: Record<string, string> = {
    Healthy: 'text-green-600',
    Registered: 'text-blue-600',
    Disconnected: 'text-red-500',
  }
  return <span className={`font-medium ${colors[status] ?? 'text-gray-600'}`}>{status}</span>
}

function getProxyStatus(status: string, lastHealthAt: string): string {
  if (!lastHealthAt) return status || 'Registered'
  const diffMs = Date.now() - new Date(lastHealthAt).getTime()
  if (diffMs < 90_000) return 'Healthy'
  if (diffMs < 300_000) return 'Registered'
  return 'Disconnected'
}

export function EdgeAgentProxiesPage() {
  const { data, isLoading } = useAgentProxies(1, 500, '')
  const items = data?.items ?? []

  const tableData: ProxyRow[] = useMemo(
    () =>
      items.map((proxy) => ({
        id: proxy.id,
        proxyId: proxy.id ?? '',
        controllerId: proxy.controllerId ?? proxy.agentControllerId ?? '',
        lastSeen: proxy.lastHealthAt ?? '',
        registered: proxy.createdAt ?? '',
        status: getProxyStatus(proxy.status ?? '', proxy.lastHealthAt ?? ''),
      })),
    [items],
  )

  const columns: ColumnDef<ProxyRow, unknown>[] = useMemo(
    () => [
      {
        accessorKey: 'proxyId',
        header: 'Proxy ID',
        cell: ({ row }) => (
          <span className="font-mono text-gray-700 text-[11px] truncate block">
            {row.getValue('proxyId')}
          </span>
        ),
        size: 280,
        enableHiding: false,
      },
      {
        accessorKey: 'controllerId',
        header: 'Controller ID',
        cell: ({ row }) => (
          <span className="font-mono text-gray-500 text-[11px] truncate block">
            {row.getValue('controllerId')}
          </span>
        ),
        size: 280,
      },
      {
        accessorKey: 'lastSeen',
        header: 'Last Seen',
        cell: ({ row }) => <span className="text-gray-700 whitespace-nowrap">{formatTime(row.getValue('lastSeen'))}</span>,
        size: 120,
      },
      {
        accessorKey: 'registered',
        header: 'Registered',
        cell: ({ row }) => <span className="text-gray-700 whitespace-nowrap">{formatTime(row.getValue('registered'))}</span>,
        size: 120,
      },
      {
        accessorKey: 'status',
        header: 'Status',
        cell: ({ row }) => <StatusBadge status={row.getValue('status') as string} />,
        size: 100,
        filterFn: statusFilterFn,
      },
    ],
    [],
  )

  const facetedFilters: FacetedFilterConfig[] = useMemo(() => [{ columnId: 'status', label: 'Status' }], [])

  if (isLoading) return <div className="flex-1" />

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-4 shrink-0 h-[44px] border-b border-border bg-white">
        <PageHeader />
      </div>
      <DataTable
        columns={columns}
        data={tableData}
        searchPlaceholder="Filter by ID..."
        searchKeys={['proxyId', 'controllerId']}
        facetedFilters={facetedFilters}
        emptyMessage="No agent proxies found."
      />
    </div>
  )
}

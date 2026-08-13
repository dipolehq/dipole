import { useMemo } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { type ColumnDef } from '@tanstack/react-table'
import { PageHeader } from '@/shared/components/PageHeader'
import { DataTable, type FacetedFilterConfig } from '@/shared/components/ui/DataTable'
import { useAgentControllers } from '@/features/edge-components/hooks/useAgentControllers'

interface ControllerRow {
  id: string
  name: string
  description: string
  controllerId: string
  modifiedAt: string
  tags: number
  status: string
}

const statusFilterFn = (row: { getValue: (id: string) => unknown }, _columnId: string, filterValue: string[]) => {
  if (!filterValue?.length) return true
  return filterValue.includes(row.getValue('status') as string)
}

function formatDate(iso: string): string {
  if (!iso) return '—'
  return new Date(iso).toLocaleDateString('en-GB', { day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit' })
}

function StatusBadge({ status }: { status: string }) {
  const colors: Record<string, string> = {
    Healthy: 'text-green-600',
    Registered: 'text-blue-600',
    NotConnected: 'text-orange-500',
    Unregistered: 'text-gray-500',
  }
  return <span className={`font-medium ${colors[status] ?? 'text-gray-600'}`}>{status}</span>
}

export function EdgeAgentControllersPage() {
  const navigate = useNavigate()

  const { data, isLoading } = useAgentControllers(1, 500, '')
  const items = data?.items ?? []

  const tableData: ControllerRow[] = useMemo(
    () =>
      items.map((item) => ({
        id: item.id,
        name: item.name ?? '',
        description: item.description ?? '',
        controllerId: item.id ?? '',
        modifiedAt: item.updatedAt ?? item.createdAt ?? '',
        tags: Array.isArray(item.tags) ? item.tags.length : (item.tags ?? 0),
        status: item.status ?? 'Unregistered',
      })),
    [items],
  )

  const columns: ColumnDef<ControllerRow, unknown>[] = useMemo(
    () => [
      {
        accessorKey: 'name',
        header: 'Name',
        cell: ({ row }) => (
          <Link
            to={`/edge-components/agent_controllers/${row.original.id}`}
            className="font-medium text-gray-900 hover:text-blue-700 transition-colors truncate block"
            onClick={(e) => e.stopPropagation()}
          >
            {row.getValue('name')}
          </Link>
        ),
        size: 180,
        enableHiding: false,
      },
      {
        accessorKey: 'description',
        header: 'Description',
        cell: ({ row }) => <span className="text-gray-700 truncate block">{row.getValue('description')}</span>,
        size: 240,
      },
      {
        accessorKey: 'controllerId',
        header: 'ID',
        cell: ({ row }) => <span className="text-gray-500 font-mono text-[11px] truncate block">{row.getValue('controllerId')}</span>,
        size: 260,
      },
      {
        accessorKey: 'modifiedAt',
        header: 'Modified At',
        cell: ({ row }) => <span className="text-gray-700 whitespace-nowrap">{formatDate(row.getValue('modifiedAt'))}</span>,
        size: 150,
      },
      {
        accessorKey: 'tags',
        header: 'Tags',
        cell: ({ row }) => <span className="text-gray-700">{row.getValue('tags')}</span>,
        size: 60,
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
        searchPlaceholder="Filter by name..."
        searchKeys={['name', 'description']}
        facetedFilters={facetedFilters}
        actionLabel="New Controller"
        onAction={() => navigate('/edge-components/agent_controllers/new')}
        onRowClick={(row) => navigate(`/edge-components/agent_controllers/${row.id}`)}
        emptyMessage="No agent controllers found."
      />
    </div>
  )
}

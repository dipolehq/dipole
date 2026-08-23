import { useMemo } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { type ColumnDef } from '@tanstack/react-table'
import { DataTable, type FacetedFilterConfig } from '@/shared/components/ui/DataTable'
import { useLogStreams } from '@/features/administration/hooks/useLogStreams'

interface LogStreamRow {
  id: string
  name: string
  description: string
  streamType: string
  destinationType: string
  status: 'Active' | 'Inactive'
}

const statusFilterFn = (row: { getValue: (id: string) => unknown }, _columnId: string, filterValue: string[]) => {
  if (!filterValue?.length) return true
  return filterValue.includes(row.getValue('status') as string)
}

export function AdminLogStreamsPage() {
  const navigate = useNavigate()

  const { data, isLoading } = useLogStreams(1, 500, '')
  const items = data?.items ?? []

  const tableData: LogStreamRow[] = useMemo(
    () =>
      items.map((item) => ({
        id: item.id,
        name: item.name ?? '',
        description: item.description ?? '',
        streamType: item.eventType ?? '',
        destinationType: item.destinationType ?? '',
        status: item.isActive !== false ? 'Active' : 'Inactive',
      })),
    [items],
  )

  const columns: ColumnDef<LogStreamRow, unknown>[] = useMemo(
    () => [
      {
        accessorKey: 'name',
        header: 'Stream Name',
        cell: ({ row }) => (
          <Link
            to={`/administration/log_streams/${row.original.id}`}
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
        size: 280,
      },
      {
        accessorKey: 'streamType',
        header: 'Stream Type',
        cell: ({ row }) => <span className="text-gray-700 truncate block">{row.getValue('streamType')}</span>,
        size: 160,
      },
      {
        accessorKey: 'destinationType',
        header: 'Destination Type',
        cell: ({ row }) => <span className="text-gray-700 truncate block">{row.getValue('destinationType')}</span>,
        size: 160,
      },
      {
        accessorKey: 'status',
        header: 'Status',
        cell: ({ row }) =>
          row.getValue('status') === 'Active' ? (
            <span className="text-green-600 font-medium">Active</span>
          ) : (
            <span className="text-red-500 font-medium">Inactive</span>
          ),
        size: 70,
        filterFn: statusFilterFn,
      },
    ],
    [],
  )

  const facetedFilters: FacetedFilterConfig[] = useMemo(() => [{ columnId: 'status', label: 'Status' }], [])

  return (
    <div className="flex flex-col h-full">
      <DataTable
        columns={columns}
        data={tableData}
        isLoading={isLoading}
        searchPlaceholder="Filter by name or type..."
        searchKeys={['name', 'streamType', 'destinationType']}
        facetedFilters={facetedFilters}
        actionLabel="New Log Stream"
        onAction={() => navigate('/administration/log_streams/new')}
        onRowClick={(row) => navigate(`/administration/log_streams/${row.id}`)}
        emptyMessage="No log streams found."
      />
    </div>
  )
}

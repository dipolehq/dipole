import { useMemo } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { type ColumnDef } from '@tanstack/react-table'
import { PageHeader } from '@/shared/components/PageHeader'
import { DataTable, type FacetedFilterConfig } from '@/shared/components/ui/DataTable'
import { useAccessConditions } from '@/features/access-conditions/hooks/useAccessConditions'

interface EntityRow {
  id: string
  name: string
  type: string
  description: string
  tags: number
  policies: number
  status: 'Active' | 'Inactive'
}

const statusFilterFn = (row: { getValue: (id: string) => unknown }, _columnId: string, filterValue: string[]) => {
  if (!filterValue?.length) return true
  return filterValue.includes(row.getValue('status') as string)
}

export function AccessConditionsPage() {
  const navigate = useNavigate()

  const { data, isLoading } = useAccessConditions(1, 500, '')
  const items = data?.items ?? []

  const tableData: EntityRow[] = useMemo(
    () =>
      items.map((item) => ({
        id: item.id,
        name: item.name,
        type: item.type ?? item.appProtocol ?? '',
        description: item.description ?? '',
        tags: item.tags ?? 0,
        policies: item.policies ?? 0,
        status: item.isActive ? 'Active' : 'Inactive',
      })),
    [items],
  )

  const columns: ColumnDef<EntityRow, unknown>[] = useMemo(
    () => [
      {
        accessorKey: 'name',
        header: 'Name',
        cell: ({ row }) => (
          <Link
            to={`/access-conditions/${row.original.id}`}
            className="font-medium text-gray-900 hover:text-blue-700 transition-colors truncate block"
            onClick={(e) => e.stopPropagation()}
          >
            {row.getValue('name')}
          </Link>
        ),
        size: 160,
        enableHiding: false,
      },
      {
        accessorKey: 'type',
        header: 'Type',
        cell: ({ row }) => <span className="text-gray-700 truncate block">{row.getValue('type')}</span>,
        size: 220,
      },
      {
        accessorKey: 'description',
        header: 'Description',
        cell: ({ row }) => <span className="text-gray-700 truncate block">{row.getValue('description')}</span>,
        size: 320,
      },
      {
        accessorKey: 'tags',
        header: 'Tags',
        cell: ({ row }) => <span className="text-gray-700">{row.getValue('tags')}</span>,
        size: 60,
      },
      {
        accessorKey: 'policies',
        header: 'Policies',
        cell: ({ row }) => <span className="text-gray-700">{row.getValue('policies')}</span>,
        size: 70,
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

  const facetedFilters: FacetedFilterConfig[] = useMemo(
    () => [{ columnId: 'status', label: 'Status' }],
    [],
  )

  if (isLoading) return <div className="flex-1" />

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-4 shrink-0 h-[44px] border-b border-border bg-white">
        <PageHeader />
      </div>
      <DataTable
        columns={columns}
        data={tableData}
        searchPlaceholder="Filter by name or type..."
        searchKeys={['name', 'type']}
        facetedFilters={facetedFilters}
        actionLabel="New Condition"
        onAction={() => navigate('/access-conditions/new')}
        onRowClick={(row) => navigate(`/access-conditions/${row.id}`)}
        emptyMessage="No access conditions found."
      />
    </div>
  )
}

import { useMemo } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { type ColumnDef } from '@tanstack/react-table'
import { DataTable, type FacetedFilterConfig } from '@/shared/components/ui/DataTable'
import { useIdentityProviders } from '@/features/administration/hooks/useIdentityProviders'

interface IdpRow {
  id: string
  name: string
  type: string
  description: string
  status: 'Active' | 'Inactive'
}

const statusFilterFn = (row: { getValue: (id: string) => unknown }, _columnId: string, filterValue: string[]) => {
  if (!filterValue?.length) return true
  return filterValue.includes(row.getValue('status') as string)
}

export function AdminIdentityProvidersPage() {
  const navigate = useNavigate()

  const { data, isLoading } = useIdentityProviders(1, 500, '')
  const items = data?.items ?? []

  const tableData: IdpRow[] = useMemo(
    () =>
      items.map((item) => ({
        id: item.id,
        name: item.name ?? '',
        type: item.type ?? '',
        description: item.description ?? '',
        status: item.isActive !== false ? 'Active' : 'Inactive',
      })),
    [items],
  )

  const columns: ColumnDef<IdpRow, unknown>[] = useMemo(
    () => [
      {
        accessorKey: 'name',
        header: 'Name',
        cell: ({ row }) => (
          <Link
            to={`/administration/identity_providers/${row.original.id}`}
            className="font-medium text-gray-900 hover:text-blue-700 transition-colors truncate block"
            onClick={(e) => e.stopPropagation()}
          >
            {row.getValue('name')}
          </Link>
        ),
        size: 200,
        enableHiding: false,
      },
      {
        accessorKey: 'type',
        header: 'Type',
        cell: ({ row }) => <span className="text-gray-700 truncate block">{row.getValue('type')}</span>,
        size: 140,
      },
      {
        accessorKey: 'description',
        header: 'Description',
        cell: ({ row }) => <span className="text-gray-700 truncate block">{row.getValue('description')}</span>,
        size: 360,
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

  if (isLoading) return <div className="flex-1" />

  return (
    <div className="flex flex-col h-full">
      <DataTable
        columns={columns}
        data={tableData}
        searchPlaceholder="Filter by name or type..."
        searchKeys={['name', 'type']}
        facetedFilters={facetedFilters}
        actionLabel="New Provider"
        onAction={() => navigate('/administration/identity_providers/new')}
        onRowClick={(row) => navigate(`/administration/identity_providers/${row.id}`)}
        emptyMessage="No identity providers found."
      />
    </div>
  )
}

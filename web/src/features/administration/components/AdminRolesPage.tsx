import { useMemo } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { type ColumnDef } from '@tanstack/react-table'
import { DataTable, type FacetedFilterConfig } from '@/shared/components/ui/DataTable'
import { useRoles } from '@/features/administration/hooks/useRoles'

interface RoleRow {
  id: string
  name: string
  description: string
  modifiedAt: string
  status: 'Active' | 'Inactive'
}

const statusFilterFn = (row: { getValue: (id: string) => unknown }, _columnId: string, filterValue: string[]) => {
  if (!filterValue?.length) return true
  return filterValue.includes(row.getValue('status') as string)
}

function formatDate(iso: string): string {
  if (!iso) return '—'
  return new Date(iso).toLocaleDateString('en-IN', { day: '2-digit', month: '2-digit', year: 'numeric' })
}

export function AdminRolesPage() {
  const navigate = useNavigate()

  const { data, isLoading } = useRoles(1, 500, '')
  const items = data?.items ?? []

  const tableData: RoleRow[] = useMemo(
    () =>
      items.map((item) => ({
        id: item.id,
        name: item.name ?? '',
        description: item.description ?? '',
        modifiedAt: item.updatedAt ?? item.createdAt ?? '',
        status: item.isActive !== false ? 'Active' : 'Inactive',
      })),
    [items],
  )

  const columns: ColumnDef<RoleRow, unknown>[] = useMemo(
    () => [
      {
        accessorKey: 'name',
        header: 'Role Name',
        cell: ({ row }) => (
          <Link
            to={`/administration/roles/${row.original.id}`}
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
        accessorKey: 'description',
        header: 'Description',
        cell: ({ row }) => <span className="text-gray-700 truncate block">{row.getValue('description')}</span>,
        size: 360,
      },
      {
        accessorKey: 'modifiedAt',
        header: 'Modified At',
        cell: ({ row }) => <span className="text-gray-700">{formatDate(row.getValue('modifiedAt'))}</span>,
        size: 120,
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
        searchPlaceholder="Filter by name..."
        searchKeys={['name', 'description']}
        facetedFilters={facetedFilters}
        actionLabel="New Role"
        onAction={() => navigate('/administration/roles/new')}
        onRowClick={(row) => navigate(`/administration/roles/${row.id}`)}
        emptyMessage="No roles found."
      />
    </div>
  )
}

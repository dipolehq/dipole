import { useMemo } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { type ColumnDef } from '@tanstack/react-table'
import { DataTable, type FacetedFilterConfig } from '@/shared/components/ui/DataTable'
import { useUsers } from '@/features/administration/hooks/useUsers'
import { apiClient } from '@/shared/lib/api-client'

interface UserRow {
  id: string
  firstName: string
  lastName: string
  email: string
  phone: string
  role: string
  status: 'Active' | 'Inactive'
}

const statusFilterFn = (row: { getValue: (id: string) => unknown }, _columnId: string, filterValue: string[]) => {
  if (!filterValue?.length) return true
  return filterValue.includes(row.getValue('status') as string)
}

export function AdminUsersPage() {
  const navigate = useNavigate()

  const { data, isLoading } = useUsers(1, 500, '')
  const items = data?.items ?? []

  const { data: rolesData } = useQuery({
    queryKey: ['roles', 'names'],
    queryFn: () => apiClient.get<{ items: { id: string; name: string }[] }>('/roles?perPage=500'),
    staleTime: 60_000,
  })
  const roleNames = useMemo(() => new Map((rolesData?.items ?? []).map((r) => [r.id, r.name])), [rolesData])

  const tableData: UserRow[] = useMemo(
    () =>
      items.map((user) => ({
        id: user.id,
        firstName: user.firstName ?? '',
        lastName: user.lastName ?? '',
        email: user.email ?? '',
        phone: user.phone ?? '—',
        role: user.roleId ? roleNames.get(user.roleId) ?? '—' : '—',
        status: user.isActive ? 'Active' : 'Inactive',
      })),
    [items, roleNames],
  )

  const columns: ColumnDef<UserRow, unknown>[] = useMemo(
    () => [
      {
        accessorKey: 'firstName',
        header: 'First Name',
        cell: ({ row }) => (
          <Link
            to={`/administration/users/${row.original.id}`}
            className="font-medium text-gray-900 hover:text-blue-700 transition-colors truncate block"
            onClick={(e) => e.stopPropagation()}
          >
            {row.getValue('firstName')}
          </Link>
        ),
        size: 140,
        enableHiding: false,
      },
      {
        accessorKey: 'lastName',
        header: 'Last Name',
        cell: ({ row }) => <span className="text-gray-700 truncate block">{row.getValue('lastName')}</span>,
        size: 140,
      },
      {
        accessorKey: 'email',
        header: 'Email',
        cell: ({ row }) => <span className="text-gray-700 truncate block">{row.getValue('email')}</span>,
        size: 260,
      },
      {
        accessorKey: 'phone',
        header: 'Phone',
        cell: ({ row }) => <span className="text-gray-700 truncate block">{row.getValue('phone')}</span>,
        size: 140,
      },
      {
        accessorKey: 'role',
        header: 'Role',
        cell: ({ row }) => <span className="text-gray-700 truncate block">{row.getValue('role')}</span>,
        size: 140,
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
        searchPlaceholder="Filter by name or email..."
        searchKeys={['firstName', 'lastName', 'email']}
        facetedFilters={facetedFilters}
        actionLabel="New User"
        onAction={() => navigate('/administration/users/new')}
        onRowClick={(row) => navigate(`/administration/users/${row.id}`)}
        emptyMessage="No users found."
      />
    </div>
  )
}

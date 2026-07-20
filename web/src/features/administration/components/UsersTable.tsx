import { User } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { apiClient } from '@/shared/lib/api-client'
import type { UserListItem } from '@/features/administration/api/usersApi'

interface UsersTableProps {
  items: UserListItem[]
  isLoading?: boolean
}

export function UsersTable({ items, isLoading }: UsersTableProps) {
  const navigate = useNavigate()

  const { data: rolesData } = useQuery({
    queryKey: ['roles', 'names'],
    queryFn: () => apiClient.get<{ items: { id: string; name: string }[] }>('/roles?perPage=100'),
    staleTime: 60_000,
  })
  const roleNames = new Map((rolesData?.items ?? []).map((r) => [r.id, r.name]))

  return (
    <table className="w-full text-xs table-fixed">
      <colgroup>
        <col className="w-[4%]" />
        <col className="w-[17%]" />
        <col className="w-[17%]" />
        <col className="w-[28%]" />
        <col className="w-[15%]" />
        <col className="w-[10%]" />
        <col className="w-[9%]" />
      </colgroup>
      <thead className="sticky top-0 bg-gray-50 z-10">
        <tr className="border-b border-border">
          <th className="px-3 py-3" />
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">First Name</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Last Name</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Company Email</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Phone</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-center">Status</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-center">Role</th>
        </tr>
      </thead>
      <tbody className="divide-y divide-border">
        {isLoading && (
          <tr>
            <td colSpan={7} className="px-4 py-12 text-center text-gray-400 text-sm" />
          </tr>
        )}
        {!isLoading && items.length === 0 && (
          <tr>
            <td colSpan={7} className="px-4 py-12 text-center text-gray-400 text-sm">
              No users found.
            </td>
          </tr>
        )}
        {!isLoading &&
          items.map((user) => (
            <tr
              key={user.id}
              className="hover:bg-gray-50 transition-colors cursor-pointer h-[42px]"
              onClick={() => navigate(`/administration/users/${user.id}`)}
            >
              <td className="px-3 py-3 text-center">
                <User size={14} className="text-gray-400" />
              </td>
              <td className="px-4 py-3 text-gray-700 truncate">{user.firstName}</td>
              <td className="px-4 py-3 text-gray-700 truncate">{user.lastName}</td>
              <td className="px-4 py-3 text-gray-700 truncate">{user.email}</td>
              <td className="px-4 py-3 text-gray-700 truncate">{user.phone || '—'}</td>
              <td className="px-4 py-3 text-center">
                {user.isActive ? (
                  <span className="text-xs font-medium text-green-600">Active</span>
                ) : (
                  <span className="text-xs font-medium text-gray-400">Inactive</span>
                )}
              </td>
              <td className="px-4 py-3 text-center text-gray-700 text-xs">
                {user.roleId ? roleNames.get(user.roleId) ?? '—' : '—'}
              </td>
            </tr>
          ))}
      </tbody>
    </table>
  )
}

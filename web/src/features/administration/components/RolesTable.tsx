import { Monitor } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import type { RoleListItem } from '@/features/administration/api/rolesApi'

interface RolesTableProps {
  items: RoleListItem[]
  isLoading?: boolean
}

export function RolesTable({ items, isLoading }: RolesTableProps) {
  const navigate = useNavigate()

  return (
    <table className="w-full text-xs table-fixed">
      <colgroup>
        <col className="w-[4%]" />
        <col className="w-[20%]" />
        <col className="w-[46%]" />
        <col className="w-[18%]" />
        <col className="w-[12%]" />
      </colgroup>
      <thead className="sticky top-0 bg-gray-50 z-10">
        <tr className="border-b border-border">
          <th className="px-3 py-3" />
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Role Name</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Description</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Modified At</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-center">Status</th>
        </tr>
      </thead>
      <tbody className="divide-y divide-border">
        {isLoading && (
          <tr>
            <td colSpan={5} className="px-4 py-12 text-center text-gray-400 text-sm" />
          </tr>
        )}
        {!isLoading && items.length === 0 && (
          <tr>
            <td colSpan={5} className="px-4 py-12 text-center text-gray-400 text-sm">
              No roles found.
            </td>
          </tr>
        )}
        {!isLoading &&
          items.map((role) => (
            <tr
              key={role.id}
              className="hover:bg-gray-50 transition-colors cursor-pointer h-[42px]"
              onClick={() => navigate(`/administration/roles/${role.id}`)}
            >
              <td className="px-3 py-3 text-center">
                <Monitor size={14} className="text-gray-400" />
              </td>
              <td className="px-4 py-3 text-gray-900 font-medium truncate">{role.name}</td>
              <td className="px-4 py-3 text-gray-700 truncate">{role.description || '—'}</td>
              <td className="px-4 py-3 text-gray-700 whitespace-nowrap">
                {new Date(role.updatedAt).toLocaleString('en-GB', { day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit', second: '2-digit' })}
              </td>
              <td className="px-4 py-3 text-center">
                {role.isActive ? (
                  <span className="text-xs font-medium text-green-600">Active</span>
                ) : (
                  <span className="text-xs font-medium text-gray-400">Inactive</span>
                )}
              </td>
            </tr>
          ))}
      </tbody>
    </table>
  )
}

import { Monitor } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { rolesData } from '@/data/administration/roles'

interface RolesTableProps {
  searchQuery: string
}

export function RolesTable({ searchQuery }: RolesTableProps) {
  const navigate = useNavigate()

  const filtered = rolesData.filter((role) => {
    if (searchQuery) {
      const q = searchQuery.toLowerCase()
      return (
        role.name.toLowerCase().includes(q) ||
        role.description.toLowerCase().includes(q)
      )
    }
    return true
  })

  return (
    <table className="w-full text-xs table-fixed">
      <colgroup>
        <col className="w-[36px]" />
        <col className="w-[18%]" />
        <col className="w-[40%]" />
        <col className="w-[11%]" />
        <col className="w-[18%]" />
        <col className="w-[10%]" />
      </colgroup>
      <thead className="sticky top-0 bg-gray-50 z-10">
        <tr className="border-b border-border">
          <th className="px-2 py-3" />
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Role Name</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Description</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-center">Users</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-center">Credential Providers</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-center">Status</th>
        </tr>
      </thead>
      <tbody className="divide-y divide-border">
        {filtered.length === 0 && (
          <tr>
            <td colSpan={6} className="px-4 py-12 text-center text-gray-400 text-sm">
              No roles found.
            </td>
          </tr>
        )}
        {filtered.map((role) => (
          <tr key={role.id} className="hover:bg-gray-50 transition-colors cursor-pointer h-[42px]" onClick={() => navigate(`/administration/roles/${role.id}`)}>
            <td className="px-2 py-3 text-center">
              <Monitor size={14} className="text-gray-400" />
            </td>
            <td className="px-4 py-3 text-gray-900 font-medium truncate">{role.name}</td>
            <td className="px-4 py-3 text-gray-700 truncate">{role.description}</td>
            <td className="px-4 py-3 text-gray-700 text-center">{role.users}</td>
            <td className="px-4 py-3 text-gray-700 text-center">{role.credentialProviders}</td>
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

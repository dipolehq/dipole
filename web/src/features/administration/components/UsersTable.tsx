import { User } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { usersData } from '@/data/administration/users'

interface UsersTableProps {
  searchQuery: string
}

export function UsersTable({ searchQuery }: UsersTableProps) {
  const navigate = useNavigate()

  const filtered = usersData.filter((user) => {
    if (searchQuery) {
      const q = searchQuery.toLowerCase()
      return (
        user.firstName.toLowerCase().includes(q) ||
        user.lastName.toLowerCase().includes(q) ||
        user.email.toLowerCase().includes(q) ||
        user.role.toLowerCase().includes(q)
      )
    }
    return true
  })

  return (
    <table className="w-full text-xs table-fixed">
      <colgroup>
        <col className="w-[4%]" />
        <col className="w-[17%]" />
        <col className="w-[17%]" />
        <col className="w-[28%]" />
        <col className="w-[15%]" />
        <col className="w-[20%]" />
      </colgroup>
      <thead className="sticky top-0 bg-gray-50 z-10">
        <tr className="border-b border-border">
          <th className="px-3 py-3" />
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">First Name</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Last Name</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Company Email</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Role</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Multi Factor Authorization</th>
        </tr>
      </thead>
      <tbody className="divide-y divide-border">
        {filtered.length === 0 && (
          <tr>
            <td colSpan={6} className="px-4 py-12 text-center text-gray-400 text-sm">
              No users found.
            </td>
          </tr>
        )}
        {filtered.map((user) => (
          <tr key={user.id} className="hover:bg-gray-50 transition-colors cursor-pointer h-[42px]" onClick={() => navigate(`/administration/users/${user.id}`)}>
            <td className="px-3 py-3 text-center">
              <User size={14} className="text-gray-400" />
            </td>
            <td className="px-4 py-3 text-gray-700 truncate">{user.firstName}</td>
            <td className="px-4 py-3 text-gray-700 truncate">{user.lastName}</td>
            <td className="px-4 py-3 text-gray-700 truncate">{user.email}</td>
            <td className="px-4 py-3 text-gray-700">{user.role}</td>
            <td className="px-4 py-3">
              {user.mfaEnabled ? (
                <span className="text-xs font-medium text-green-600">Enabled</span>
              ) : (
                <span className="text-xs font-medium text-gray-400">Disabled</span>
              )}
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}

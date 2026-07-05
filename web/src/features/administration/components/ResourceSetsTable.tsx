import { Monitor, ArrowLeftRight, MonitorSmartphone, CheckCircle, Clock, Key, Server } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { resourceSetsData } from '@/data/administration/resourceSets'
import type { ResourceSetResource } from '@/data/administration/resourceSets'

interface ResourceSetsTableProps {
  searchQuery: string
}

export function ResourceSetsTable({ searchQuery }: ResourceSetsTableProps) {
  const navigate = useNavigate()

  const filtered = resourceSetsData.filter((rs) => {
    if (searchQuery) {
      return rs.name.toLowerCase().includes(searchQuery.toLowerCase())
    }
    return true
  })

  return (
    <table className="w-full text-xs table-fixed">
      <colgroup>
        <col className="w-[4%]" />
        <col className="w-[14%]" />
        <col className="w-[30%]" />
        <col className="w-[35%]" />
        <col className="w-[9%]" />
        <col className="w-[9%]" />
      </colgroup>
      <thead className="sticky top-0 bg-gray-50 z-10">
        <tr className="border-b border-border">
          <th className="px-3 py-3" />
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Name</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">ID</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Resources</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-center">Roles</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-center">Users</th>
        </tr>
      </thead>
      <tbody className="divide-y divide-border">
        {filtered.length === 0 && (
          <tr>
            <td colSpan={6} className="px-4 py-12 text-center text-gray-400 text-sm">
              No resource sets found.
            </td>
          </tr>
        )}
        {filtered.map((rs) => (
          <tr key={rs.id} className="hover:bg-gray-50 transition-colors cursor-pointer h-[41px]" onClick={() => navigate(`/administration/resource_sets/${rs.id}`)}>
            <td className="px-3 py-3 text-center">
              <Monitor size={14} className="text-gray-400" />
            </td>
            <td className="px-4 py-3 text-gray-900 font-medium truncate">{rs.name}</td>
            <td className="px-4 py-3 text-gray-500 truncate text-[11px]">{rs.id}</td>
            <td className="px-4 py-3">
              <ResourceIcons resources={rs.resources} />
            </td>
            <td className="px-4 py-3 text-gray-700 text-center">{rs.roles}</td>
            <td className="px-4 py-3 text-gray-700 text-center">{rs.users}</td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}

function ResourceIcons({ resources }: { resources: ResourceSetResource }) {
  const items = [
    { icon: ArrowLeftRight, count: resources.accessPolicies },
    { icon: MonitorSmartphone, count: resources.clientWorkloads },
    { icon: CheckCircle, count: resources.trustProviders },
    { icon: Clock, count: resources.accessConditions },
    { icon: Key, count: resources.credentialProviders },
    { icon: Server, count: resources.serverWorkloads },
  ]

  return (
    <div className="flex items-center gap-5">
      {items.map((item, i) => (
        <div key={i} className="flex items-center gap-1 text-gray-500">
          <item.icon size={15} strokeWidth={2.5} />
          <span className="text-[11px]">{item.count}</span>
        </div>
      ))}
    </div>
  )
}

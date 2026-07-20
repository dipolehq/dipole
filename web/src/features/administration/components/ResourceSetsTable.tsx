import { Monitor, ArrowLeftRight, MonitorSmartphone, CheckCircle, Clock, Key, Server } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import type { ResourceSetListItem } from '@/features/administration/api/resourceSetsApi'

interface ResourceSetsTableProps {
  items: ResourceSetListItem[]
  isLoading?: boolean
}

export function ResourceSetsTable({ items, isLoading }: ResourceSetsTableProps) {
  const navigate = useNavigate()

  return (
    <table className="w-full text-xs table-fixed">
      <colgroup>
        <col className="w-[4%]" />
        <col className="w-[12%]" />
        <col className="w-[18%]" />
        <col className="w-[22%]" />
        <col className="w-[30%]" />
        <col className="w-[14%]" />
      </colgroup>
      <thead className="sticky top-0 bg-gray-50 z-10">
        <tr className="border-b border-border">
          <th className="px-3 py-3" />
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Name</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Description</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">ID</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Resources</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Modified At</th>
        </tr>
      </thead>
      <tbody className="divide-y divide-border">
        {isLoading && (
          <tr>
            <td colSpan={6} className="px-4 py-12 text-center text-gray-400 text-sm" />
          </tr>
        )}
        {!isLoading && items.length === 0 && (
          <tr>
            <td colSpan={6} className="px-4 py-12 text-center text-gray-400 text-sm">
              No resource sets found.
            </td>
          </tr>
        )}
        {!isLoading &&
          items.map((rs) => (
            <tr
              key={rs.id}
              className="hover:bg-gray-50 transition-colors cursor-pointer h-[41px]"
              onClick={() => navigate(`/administration/resource_sets/${rs.id}`)}
            >
              <td className="px-3 py-3 text-center">
                <Monitor size={14} className="text-gray-400" />
              </td>
              <td className="px-4 py-3 text-gray-900 font-medium truncate">{rs.name}</td>
              <td className="px-4 py-3 text-gray-700 truncate">{rs.description || '—'}</td>
              <td className="px-4 py-3 text-gray-500 truncate">{rs.id}</td>
              <td className="px-4 py-3">
                <ResourceIcons />
              </td>
              <td className="px-4 py-3 text-gray-700 whitespace-nowrap">
                {new Date(rs.updatedAt).toLocaleString('en-GB', { day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit', second: '2-digit' })}
              </td>
            </tr>
          ))}
      </tbody>
    </table>
  )
}

function ResourceIcons() {
  // TODO: Once the backend provides per-resource-set counts, pass them as props.
  // For now, show icons with 0 counts as placeholder.
  const items = [
    { icon: ArrowLeftRight, count: 0 },
    { icon: MonitorSmartphone, count: 0 },
    { icon: CheckCircle, count: 0 },
    { icon: Clock, count: 0 },
    { icon: Key, count: 0 },
    { icon: Server, count: 0 },
  ]

  return (
    <div className="flex items-center gap-7">
      {items.map((item, i) => (
        <div key={i} className="flex items-center gap-1 text-gray-500">
          <item.icon size={15} strokeWidth={2.5} />
          <span className="text-[11px]">{item.count}</span>
        </div>
      ))}
    </div>
  )
}

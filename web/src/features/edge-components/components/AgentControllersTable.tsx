import { Settings, RefreshCw } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import type { AgentControllerListItem } from '@/features/edge-components/api/agentControllersApi'

interface AgentControllersTableProps {
  items: AgentControllerListItem[]
  isLoading?: boolean
}

export function AgentControllersTable({ items, isLoading }: AgentControllersTableProps) {
  const navigate = useNavigate()

  return (
    <table className="w-full text-xs table-fixed">
      <colgroup>
        <col className="w-[4%]" />
        <col className="w-[18%]" />
        <col className="w-[22%]" />
        <col className="w-[26%]" />
        <col className="w-[14%]" />
        <col className="w-[6%]" />
        <col className="w-[10%]" />
      </colgroup>
      <thead className="sticky top-0 bg-gray-50 z-10">
        <tr className="border-b border-border">
          <th className="px-3 py-2.5 leading-none">
            <button type="button" className="text-gray-400 hover:text-gray-600 transition-colors" title="Refresh">
              <RefreshCw size={15} strokeWidth={2.5} />
            </button>
          </th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Name</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Description</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">ID</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left whitespace-nowrap">Modified At</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-center">Tags</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-center">Status</th>
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
              No agent controllers found.
            </td>
          </tr>
        )}
        {!isLoading &&
          items.map((ac) => (
            <tr
              key={ac.id}
              className="hover:bg-gray-50 transition-colors cursor-pointer h-[41px]"
              onClick={() => navigate(`/edge-components/agent_controllers/${ac.id}`)}
            >
              <td className="px-4 py-3 text-center">
                <Settings size={14} className="text-gray-400" />
              </td>
              <td className="px-4 py-3 text-gray-900 font-medium truncate">{ac.name}</td>
              <td className="px-4 py-3 text-gray-700 truncate">{ac.description || '—'}</td>
              <td className="px-4 py-3 text-gray-500 truncate">{ac.id}</td>
              <td className="px-4 py-3 text-gray-700 whitespace-nowrap">
                {ac.updatedAt ? new Date(ac.updatedAt).toLocaleString('en-GB', { day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit', second: '2-digit' }) : '—'}
              </td>
              <td className="px-4 py-3 text-center text-gray-700">
                {ac.tags?.length ?? 0}
              </td>
              <td className="px-4 py-3 text-center whitespace-nowrap">
                <StatusBadge status={ac.status} />
              </td>
            </tr>
          ))}
      </tbody>
    </table>
  )
}

function StatusBadge({ status }: { status: string }) {
  const config: Record<string, { label: string; color: string }> = {
    healthy: { label: 'Healthy', color: 'text-green-600' },
    registered: { label: 'Registered', color: 'text-blue-600' },
    unregistered: { label: 'Unregistered', color: 'text-gray-400' },
    not_connected: { label: 'Not Connected', color: 'text-gray-600' },
  }
  const c = config[status] ?? { label: status, color: 'text-gray-500' }
  return <span className={`text-xs font-medium ${c.color}`}>{c.label}</span>
}

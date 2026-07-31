import { RefreshCw } from 'lucide-react'
import type { AgentProxyDTO } from '@/features/edge-components/api/agentProxiesApi'

interface AgentProxiesTableProps {
  items: AgentProxyDTO[]
  isLoading?: boolean
}

function getStatusColor(status: string, lastHealthAt?: string): { label: string; color: string } {
  if (!lastHealthAt) return { label: 'Registered', color: 'text-yellow-600' }

  const lastSeen = new Date(lastHealthAt).getTime()
  const now = Date.now()
  const diffSec = (now - lastSeen) / 1000

  if (status === 'healthy' || diffSec < 90) return { label: 'Healthy', color: 'text-green-600' }
  if (diffSec < 300) return { label: 'Registered', color: 'text-yellow-600' }
  return { label: 'Disconnected', color: 'text-red-500' }
}

function formatTime(iso?: string): string {
  if (!iso) return '—'
  const d = new Date(iso)
  return d.toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' })
}

export function AgentProxiesTable({ items, isLoading }: AgentProxiesTableProps) {
  return (
    <table className="w-full text-xs table-fixed">
      <colgroup>
        <col className="w-[4%]" />
        <col className="w-[30%]" />
        <col className="w-[22%]" />
        <col className="w-[22%]" />
        <col className="w-[12%]" />
        <col className="w-[10%]" />
      </colgroup>
      <thead className="sticky top-0 bg-gray-50 z-10">
        <tr className="border-b border-border">
          <th className="px-3 py-2.5 leading-none">
            <RefreshCw size={15} strokeWidth={2.5} className="text-gray-400" />
          </th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Proxy ID</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Controller ID</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Last Seen</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Registered</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-center">Status</th>
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
              No agent proxies registered yet.
            </td>
          </tr>
        )}
        {!isLoading && items.map((proxy, idx) => {
          const { label, color } = getStatusColor(proxy.status, proxy.lastHealthAt)
          return (
            <tr key={proxy.id} className="hover:bg-gray-50 transition-colors">
              <td className="px-3 py-3 text-gray-400 text-center">{idx + 1}</td>
              <td className="px-4 py-3 font-mono text-gray-700 truncate" title={proxy.id}>
                {proxy.id.slice(0, 8)}…
              </td>
              <td className="px-4 py-3 font-mono text-gray-500 truncate" title={proxy.agentControllerId}>
                {proxy.agentControllerId ? proxy.agentControllerId.slice(0, 8) + '…' : '—'}
              </td>
              <td className="px-4 py-3 text-gray-600">
                {formatTime(proxy.lastHealthAt)}
              </td>
              <td className="px-4 py-3 text-gray-600">
                {formatTime(proxy.registeredAt)}
              </td>
              <td className={`px-4 py-3 text-center font-medium ${color}`}>
                {label}
              </td>
            </tr>
          )
        })}
      </tbody>
    </table>
  )
}

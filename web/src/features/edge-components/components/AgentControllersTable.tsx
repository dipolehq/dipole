import { useState } from 'react'
import { Settings, RefreshCw } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { agentControllersData } from '@/data/edge-components/agentControllers'
import type { AgentController, AgentControllerStatus, TlsStatus } from '@/data/edge-components/agentControllers'

interface AgentControllersTableProps {
  searchQuery: string
}

export function AgentControllersTable({ searchQuery }: AgentControllersTableProps) {
  const navigate = useNavigate()

  const filtered = agentControllersData.filter((ac) => {
    if (searchQuery) {
      const q = searchQuery.toLowerCase()
      return (
        ac.name.toLowerCase().includes(q) ||
        ac.description.toLowerCase().includes(q) ||
        ac.id.toLowerCase().includes(q)
      )
    }
    return true
  })

  return (
    <table className="w-full text-xs table-fixed">
      <colgroup>
        <col className="w-[4%]" />
        <col className="w-[14%]" />
        <col className="w-[19%]" />
        <col className="w-[24%]" />
        <col className="w-[7%]" />
        <col className="w-[14%]" />
        <col className="w-[5%]" />
        <col className="w-[4%]" />
        <col className="w-[9%]" />
      </colgroup>
      <thead className="sticky top-0 bg-gray-50 z-10">
        <tr className="border-b border-border">
          <th className="px-3 py-2.5 leading-none">
            <button type="button" className="text-gray-400 hover:text-orange-500 transition-colors" title="Refresh">
              <RefreshCw size={15} strokeWidth={2.5} />
            </button>
          </th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Name</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Description</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">ID</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Version</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left whitespace-nowrap">Modified At</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-center">Tags</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-center">TLS</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-center">Status</th>
        </tr>
      </thead>
      <tbody className="divide-y divide-border">
        {filtered.length === 0 && (
          <tr>
            <td colSpan={9} className="px-4 py-12 text-center text-gray-400 text-sm">
              No agent controllers found.
            </td>
          </tr>
        )}
        {filtered.map((ac) => (
          <tr
            key={ac.id}
            className="hover:bg-gray-50 transition-colors cursor-pointer h-[41px]"
            onClick={() => navigate(`/edge-components/agent_controllers/${ac.id}`)}
          >
            <td className="px-4 py-3 text-center">
              <Settings size={14} className="text-gray-400" />
            </td>
            <td className="px-4 py-3 text-gray-900 font-medium truncate">{ac.name}</td>
            <td className="px-4 py-3 text-gray-700 truncate">{ac.description}</td>
            <td className="px-4 py-3 text-gray-500 truncate text-[11px] font-mono">{ac.id}</td>
            <td className="px-4 py-3 text-gray-700 whitespace-nowrap">{ac.version}</td>
            <td className="px-4 py-3 text-gray-700 whitespace-nowrap">{ac.modifiedAt}</td>
            <td className="px-4 py-3 text-center text-gray-700">{ac.tags}</td>
            <td className="px-4 py-3 text-center">
              <TlsIcon controller={ac} />
            </td>
            <td className="px-4 py-3 text-center whitespace-nowrap">
              <StatusText status={ac.status} lastReportedUptime={ac.lastReportedUptime} />
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}

function StatusText({ status, lastReportedUptime }: { status: AgentControllerStatus; lastReportedUptime?: number }) {
  const [showTooltip, setShowTooltip] = useState(false)

  const config: Record<AgentControllerStatus, { label: string; color: string }> = {
    healthy: { label: 'Healthy', color: 'text-green-600' },
    registered: { label: 'Registered', color: 'text-blue-600' },
    unregistered: { label: 'Unregistered', color: 'text-gray-400' },
    not_connected: { label: 'Not Connected', color: 'text-orange-500' },
  }
  const c = config[status]

  function getUptimeDisplay() {
    if (!lastReportedUptime) return null
    const date = new Date(Date.now() - lastReportedUptime * 1000)
    return `Last Reported Uptime: ${date.toLocaleDateString('en-GB', { day: '2-digit', month: '2-digit', year: 'numeric' })}, ${date.toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit', second: '2-digit' })}`
  }

  const uptimeText = getUptimeDisplay()

  return (
    <div
      className="relative inline-flex justify-center"
      onMouseEnter={() => setShowTooltip(true)}
      onMouseLeave={() => setShowTooltip(false)}
    >
      <span className={`text-xs font-medium ${c.color}`}>{c.label}</span>
      {showTooltip && uptimeText && (
        <div className="absolute bottom-full left-1/2 -translate-x-1/2 mb-2 z-50">
          <div className="bg-[#28282B] text-white text-xs rounded-lg px-3 py-2 whitespace-nowrap shadow-lg">
            {uptimeText}
          </div>
        </div>
      )}
    </div>
  )
}

function TlsIcon({ controller }: { controller: AgentController }) {
  const [showTooltip, setShowTooltip] = useState(false)

  const colorMap: Record<TlsStatus, string> = {
    valid: 'text-green-500',
    managed: 'text-blue-500',
    expiring_soon: 'text-yellow-500',
    expired: 'text-red-500',
    not_configured: 'text-gray-400',
  }

  const color = colorMap[controller.tlsStatus]

  // Build tooltip content based on TLS cert info
  function getTooltipContent() {
    if (!controller.tlsCert) return null
    const expDate = new Date(controller.tlsCert.notAfter)
    const isExpired = expDate < new Date()
    return (
      <div className="absolute bottom-full left-1/2 -translate-x-1/2 mb-2 z-50">
        <div className="bg-[#28282B] text-white text-xs rounded-lg px-3 py-2 whitespace-nowrap shadow-lg">
          <div>S/N: {controller.tlsCert.serialNumber}</div>
          <div>{isExpired ? 'Expired on:' : 'Expires on:'}</div>
          <div>{expDate.toLocaleDateString('en-GB', { day: '2-digit', month: '2-digit', year: 'numeric' })}, {expDate.toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit', second: '2-digit' })}</div>
        </div>
      </div>
    )
  }

  return (
    <div
      className="relative inline-flex justify-center"
      onMouseEnter={() => setShowTooltip(true)}
      onMouseLeave={() => setShowTooltip(false)}
    >
      {controller.tlsStatus === 'expired' || controller.tlsStatus === 'expiring_soon' ? (
        <svg className={`w-4 h-4 ${color}`} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
          <circle cx="12" cy="12" r="10" />
          <line x1="12" y1="8" x2="12" y2="12" />
          <line x1="12" y1="16" x2="12.01" y2="16" />
        </svg>
      ) : (
        <svg className={`w-4 h-4 ${color}`} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
          <circle cx="12" cy="12" r="10" />
          <line x1="8" y1="12" x2="16" y2="12" />
        </svg>
      )}
      {showTooltip && getTooltipContent()}
    </div>
  )
}

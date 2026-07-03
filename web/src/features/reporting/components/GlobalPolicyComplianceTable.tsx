import { ExternalLink, RefreshCw } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { globalPolicyComplianceData } from '@/data/reporting/globalPolicyCompliance'
import type { ComplianceLevel } from '@/data/reporting/globalPolicyCompliance'

function ComplianceBadge({ level }: { level: ComplianceLevel }) {
  const colors: Record<ComplianceLevel, string> = {
    Required: 'text-red-500',
    Recommended: 'text-orange-500',
    Optional: 'text-blue-500',
  }
  return (
    <span className={`text-xs font-medium ${colors[level]}`}>
      {level}
    </span>
  )
}

interface GlobalPolicyComplianceTableProps {
  searchQuery: string
  complianceStatus: string
  resourceSet: string
}

export function GlobalPolicyComplianceTable({ searchQuery, complianceStatus, resourceSet }: GlobalPolicyComplianceTableProps) {
  const navigate = useNavigate()

  const filtered = globalPolicyComplianceData.filter((entry) => {
    if (resourceSet && entry.resourceSet !== resourceSet) return false
    if (complianceStatus) {
      if (complianceStatus === 'Compliant') {
        if (entry.trustProvider === 'Required' || entry.accessCondition === 'Required' ||
            entry.trustProvider === 'Recommended' || entry.accessCondition === 'Recommended') return false
      } else if (complianceStatus === 'Missing Required') {
        if (entry.trustProvider !== 'Required' && entry.accessCondition !== 'Required') return false
      } else if (complianceStatus === 'Missing Recommended') {
        if (entry.trustProvider !== 'Recommended' && entry.accessCondition !== 'Recommended') return false
      }
    }
    if (searchQuery) {
      const q = searchQuery.toLowerCase()
      const matches =
        entry.policyName.toLowerCase().includes(q) ||
        entry.clientWorkload.toLowerCase().includes(q) ||
        entry.serverWorkload.toLowerCase().includes(q) ||
        entry.credentialProvider.toLowerCase().includes(q) ||
        entry.trustProvider.toLowerCase().includes(q) ||
        entry.accessCondition.toLowerCase().includes(q) ||
        entry.resourceSet.toLowerCase().includes(q)
      if (!matches) return false
    }
    return true
  })

  return (
    <table className="w-full text-xs table-fixed">
      <colgroup>
        <col className="w-[36px]" />
        <col className="w-[15%]" />
        <col className="w-[13%]" />
        <col className="w-[12%]" />
        <col className="w-[14%]" />
        <col className="w-[16%]" />
        <col className="w-[15%]" />
        <col className="w-[12%]" />
      </colgroup>
      <thead className="sticky top-0 bg-gray-50 z-10">
        <tr className="border-b border-border">
          <th className="px-2 py-3">
            <button type="button" className="text-gray-400 hover:text-orange-500 transition-colors" title="Refresh">
              <RefreshCw size={15} strokeWidth={2.5} />
            </button>
          </th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Policy Name</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Client Workload</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Trust Provider</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Access Condition</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Credential Provider</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Server Workload</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Resource Set</th>
        </tr>
      </thead>
      <tbody className="divide-y divide-border">
        {filtered.length === 0 && (
          <tr>
            <td colSpan={8} className="px-4 py-12 text-center text-gray-400 text-sm">
              No policies found.
            </td>
          </tr>
        )}
        {filtered.map((entry) => (
          <tr key={entry.id} className="hover:bg-gray-50 transition-colors">
            <td className="px-2 py-3 text-center">
              <button
                type="button"
                onClick={() => navigate(`/access-policies/${entry.id}`)}
                className="text-gray-400 hover:text-orange-500 transition-colors"
              >
                <ExternalLink size={14} />
              </button>
            </td>
            <td className="px-4 py-3 text-gray-700 truncate">{entry.policyName}</td>
            <td className="px-4 py-3 text-gray-700 truncate">{entry.clientWorkload}</td>
            <td className="px-4 py-3">
              <ComplianceBadge level={entry.trustProvider} />
            </td>
            <td className="px-4 py-3">
              <ComplianceBadge level={entry.accessCondition} />
            </td>
            <td className="px-4 py-3 text-gray-700 truncate">{entry.credentialProvider}</td>
            <td className="px-4 py-3 text-gray-700 truncate">{entry.serverWorkload}</td>
            <td className="px-4 py-3 text-gray-700 truncate">{entry.resourceSet}</td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}

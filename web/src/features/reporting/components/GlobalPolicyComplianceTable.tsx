import { ExternalLink, RefreshCw } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '@/shared/lib/api-client'

type ComplianceLevel = 'required' | 'recommended' | 'optional' | 'compliant'

interface GlobalPolicyComplianceEntry {
  id: string
  policyName: string
  clientWorkload: string
  trustProvider: ComplianceLevel
  accessCondition: ComplianceLevel
  credentialProvider: string
  serverWorkload: string
  resourceSet: string
}

async function fetchComplianceReport(): Promise<GlobalPolicyComplianceEntry[]> {
  const res = await apiClient.get<{ items: GlobalPolicyComplianceEntry[]; total: number }>('/global-policy-compliance/report')
  return res.items ?? []
}

function ComplianceBadge({ level }: { level: ComplianceLevel }) {
  const config: Record<ComplianceLevel, { label: string; color: string }> = {
    required: { label: 'Required', color: 'text-red-500' },
    recommended: { label: 'Recommended', color: 'text-gray-600' },
    optional: { label: 'Optional', color: 'text-blue-500' },
    compliant: { label: '', color: '' },
  }
  const c = config[level] ?? config.optional
  if (level === 'compliant') return null
  return (
    <span className={`text-xs font-medium ${c.color}`}>
      {c.label}
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
  const qc = useQueryClient()

  const { data: items = [] } = useQuery({
    queryKey: ['compliance-report'],
    queryFn: fetchComplianceReport,
  })

  function handleRefresh() {
    qc.invalidateQueries({ queryKey: ['compliance-report'] })
  }

  const filtered = items.filter((entry) => {
    if (resourceSet && entry.resourceSet !== resourceSet) return false
    if (complianceStatus) {
      if (complianceStatus === 'Compliant') {
        if (entry.trustProvider !== 'compliant' || entry.accessCondition !== 'compliant') return false
      } else if (complianceStatus === 'Missing Required') {
        if (entry.trustProvider !== 'required' && entry.accessCondition !== 'required') return false
      } else if (complianceStatus === 'Missing Recommended') {
        if (entry.trustProvider !== 'recommended' && entry.accessCondition !== 'recommended') return false
      }
    }
    if (searchQuery) {
      const q = searchQuery.toLowerCase()
      const matches =
        entry.policyName.toLowerCase().includes(q) ||
        entry.clientWorkload.toLowerCase().includes(q) ||
        entry.serverWorkload.toLowerCase().includes(q) ||
        entry.credentialProvider.toLowerCase().includes(q) ||
        entry.resourceSet.toLowerCase().includes(q)
      if (!matches) return false
    }
    return true
  })

  return (
    <table className="w-full text-xs table-fixed">
      <colgroup>
        <col className="w-[4%]" />
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
          <th className="px-3 py-2.5 leading-none">
            <button type="button" onClick={handleRefresh} className="text-gray-400 hover:text-gray-600 transition-colors" title="Refresh">
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
          <tr key={entry.id} className="hover:bg-gray-50 transition-colors h-[41px]">
            <td className="px-2 py-[10px] text-center">
              <button
                type="button"
                onClick={() => navigate(`/access-policies/${entry.id}`)}
                className="text-gray-400 hover:text-gray-600 transition-colors"
              >
                <ExternalLink size={14} />
              </button>
            </td>
            <td className="px-4 py-[10px] text-gray-700 truncate">{entry.policyName}</td>
            <td className="px-4 py-[10px] text-gray-700 truncate">{entry.clientWorkload}</td>
            <td className="px-4 py-[10px]">
              <ComplianceBadge level={entry.trustProvider} />
            </td>
            <td className="px-4 py-[10px]">
              <ComplianceBadge level={entry.accessCondition} />
            </td>
            <td className="px-4 py-[10px] text-gray-700 truncate">{entry.credentialProvider}</td>
            <td className="px-4 py-[10px] text-gray-700 truncate">{entry.serverWorkload}</td>
            <td className="px-4 py-[10px] text-gray-700 truncate">{entry.resourceSet}</td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}

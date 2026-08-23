import { useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { type ColumnDef } from '@tanstack/react-table'
import { ExternalLink, RefreshCw } from 'lucide-react'
import { DataTable } from '@/shared/components/ui/DataTable'
import { apiClient } from '@/shared/lib/api-client'

type ComplianceLevel = 'required' | 'recommended' | 'optional' | 'compliant'

interface ComplianceRow {
  id: string
  policyName: string
  clientWorkload: string
  trustProvider: ComplianceLevel
  accessCondition: ComplianceLevel
  credentialProvider: string
  serverWorkload: string
  resourceSet: string
  /** Derived field for faceted filtering */
  status: string
}

async function fetchComplianceReport(): Promise<ComplianceRow[]> {
  const res = await apiClient.get<{ items: ComplianceRow[]; total: number }>('/global-policy-compliance/report')
  return (res.items ?? []).map((item) => ({
    ...item,
    status: deriveStatus(item.trustProvider, item.accessCondition),
  }))
}

function deriveStatus(tp: ComplianceLevel, ac: ComplianceLevel): string {
  if (tp === 'required' || ac === 'required') return 'Missing Required'
  if (tp === 'recommended' || ac === 'recommended') return 'Missing Recommended'
  return 'Compliant'
}

function ComplianceBadge({ level }: { level: ComplianceLevel }) {
  const config: Record<ComplianceLevel, { label: string; color: string }> = {
    required: { label: 'Required', color: 'text-red-500' },
    recommended: { label: 'Recommended', color: 'text-gray-600' },
    optional: { label: 'Optional', color: 'text-blue-500' },
    compliant: { label: '—', color: 'text-gray-400' },
  }
  const c = config[level] ?? config.optional
  return <span className={`text-xs font-medium ${c.color}`}>{c.label}</span>
}

export function ReportingCompliancePage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  const { data: items = [] } = useQuery({
    queryKey: ['compliance-report'],
    queryFn: fetchComplianceReport,
  })

  const columns: ColumnDef<ComplianceRow, unknown>[] = useMemo(() => [
    {
      id: '_action',
      header: () => (
        <button
          type="button"
          onClick={() => queryClient.invalidateQueries({ queryKey: ['compliance-report'] })}
          className="text-gray-400 hover:text-gray-600 transition-colors"
          title="Refresh"
        >
          <RefreshCw size={15} strokeWidth={2.5} />
        </button>
      ),
      cell: ({ row }) => (
        <button
          type="button"
          onClick={(e) => { e.stopPropagation(); navigate(`/access-policies/${row.original.id}`) }}
          className="text-gray-400 hover:text-gray-600 transition-colors"
          title="View policy"
        >
          <ExternalLink size={14} />
        </button>
      ),
      size: 40,
      enableSorting: false,
      enableHiding: false,
    },
    {
      accessorKey: 'policyName',
      header: 'Policy Name',
      cell: ({ row }) => <span className="text-gray-700 truncate block">{row.getValue('policyName')}</span>,
    },
    {
      accessorKey: 'clientWorkload',
      header: 'Client Workload',
      cell: ({ row }) => <span className="text-gray-700 truncate block">{row.getValue('clientWorkload')}</span>,
    },
    {
      accessorKey: 'trustProvider',
      header: 'Trust Provider',
      cell: ({ row }) => <ComplianceBadge level={row.getValue('trustProvider')} />,
      size: 130,
    },
    {
      accessorKey: 'accessCondition',
      header: 'Access Condition',
      cell: ({ row }) => <ComplianceBadge level={row.getValue('accessCondition')} />,
      size: 140,
    },
    {
      accessorKey: 'credentialProvider',
      header: 'Credential Provider',
      cell: ({ row }) => <span className="text-gray-700 truncate block">{row.getValue('credentialProvider')}</span>,
    },
    {
      accessorKey: 'serverWorkload',
      header: 'Server Workload',
      cell: ({ row }) => <span className="text-gray-700 truncate block">{row.getValue('serverWorkload')}</span>,
    },
    {
      accessorKey: 'resourceSet',
      header: 'Resource Set',
      cell: ({ row }) => <span className="text-gray-700 truncate block">{row.getValue('resourceSet')}</span>,
      size: 120,
    },
    {
      accessorKey: 'status',
      header: 'Status',
      cell: ({ row }) => {
        const status = row.getValue('status') as string
        const colors: Record<string, string> = {
          'Compliant': 'text-green-600',
          'Missing Required': 'text-red-500',
          'Missing Recommended': 'text-gray-600',
        }
        return <span className={`text-xs font-medium ${colors[status] ?? 'text-gray-600'}`}>{status}</span>
      },
      size: 140,
    },
  ], [queryClient, navigate])

  return (
    <div className="flex flex-col h-full">
      <div className="flex-1 min-h-0">
        <DataTable
          columns={columns}
          data={items}
          searchPlaceholder="Filter by policy, workload, or provider..."
          searchKeys={['policyName', 'clientWorkload', 'serverWorkload', 'credentialProvider', 'resourceSet']}
          facetedFilters={[{ columnId: 'status', label: 'Status' }]}
          onRowClick={(row) => navigate(`/access-policies/${row.id}`)}
          emptyMessage="No policies found."
        />
      </div>
    </div>
  )
}

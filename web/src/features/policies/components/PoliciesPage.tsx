import { useMemo } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { type ColumnDef } from '@tanstack/react-table'
import { PageHeader } from '@/shared/components/PageHeader'
import { DataTable, type FacetedFilterConfig } from '@/shared/components/ui/DataTable'
import { usePolicies } from '@/features/policies/hooks/usePolicy'
import { fetchClientWorkloads, fetchServerWorkloads, fetchTrustProviders, fetchAccessConditions, fetchCredentialProviders } from '@/features/policies/api/entityApi'

interface PolicyRow {
  id: string
  name: string
  clientWorkload: string
  trustProvider: string
  accessCondition: string
  credentialProvider: string
  serverWorkload: string
  status: 'Active' | 'Inactive'
}

const statusFilterFn = (row: { getValue: (id: string) => unknown }, _columnId: string, filterValue: string[]) => {
  if (!filterValue?.length) return true
  return filterValue.includes(row.getValue('status') as string)
}

export function PoliciesPage() {
  const navigate = useNavigate()

  const { data, isLoading } = usePolicies(1, 500, '')
  const items = data?.items ?? []

  // Fetch entity names for display
  const { data: cwData } = useQuery({ queryKey: ['client-workloads', 'names'], queryFn: () => fetchClientWorkloads({ perPage: 500 }), staleTime: 60_000 })
  const { data: swData } = useQuery({ queryKey: ['server-workloads', 'names'], queryFn: () => fetchServerWorkloads({ perPage: 500 }), staleTime: 60_000 })
  const { data: tpData } = useQuery({ queryKey: ['trust-providers', 'names'], queryFn: () => fetchTrustProviders({ perPage: 500 }), staleTime: 60_000 })
  const { data: acData } = useQuery({ queryKey: ['access-conditions', 'names'], queryFn: () => fetchAccessConditions({ perPage: 500 }), staleTime: 60_000 })
  const { data: cpData } = useQuery({ queryKey: ['credential-providers', 'names'], queryFn: () => fetchCredentialProviders({ perPage: 500 }), staleTime: 60_000 })

  const cwNames = useMemo(() => new Map((cwData?.items ?? []).map((i) => [i.id, i.name])), [cwData])
  const swNames = useMemo(() => new Map((swData?.items ?? []).map((i) => [i.id, i.name])), [swData])
  const tpNames = useMemo(() => new Map((tpData?.items ?? []).map((i) => [i.id, i.name])), [tpData])
  const acNames = useMemo(() => new Map((acData?.items ?? []).map((i) => [i.id, i.name])), [acData])
  const cpNames = useMemo(() => new Map((cpData?.items ?? []).map((i) => [i.id, i.name])), [cpData])

  function resolveName(map: Map<string, string>, id: string | undefined): string {
    if (!id) return '—'
    return map.get(id) ?? '—'
  }

  function resolveNames(map: Map<string, string>, ids: string[]): string {
    if (!ids || ids.length === 0) return '—'
    const names = ids.map((id) => map.get(id)).filter(Boolean)
    return names.length > 0 ? names.join(', ') : '—'
  }

  const tableData: PolicyRow[] = useMemo(
    () =>
      items.map((policy) => ({
        id: policy.id,
        name: policy.name,
        clientWorkload: resolveName(cwNames, policy.clientWorkloadId),
        trustProvider: resolveNames(tpNames, policy.trustProviderIds),
        accessCondition:
          policy.accessConditionIds.length > 0
            ? resolveNames(acNames, policy.accessConditionIds)
            : '',
        credentialProvider: resolveNames(cpNames, policy.credentialProviderIds),
        serverWorkload: resolveName(swNames, policy.serverWorkloadId),
        status: policy.isActive ? 'Active' : 'Inactive',
      })),
    [items, cwNames, swNames, tpNames, acNames, cpNames],
  )

  const columns: ColumnDef<PolicyRow, unknown>[] = useMemo(
    () => [
      {
        accessorKey: 'name',
        header: 'Access Policy Name',
        cell: ({ row }) => (
          <Link
            to={`/access-policies/${row.original.id}`}
            className="font-medium text-gray-900 hover:text-blue-700 transition-colors"
            onClick={(e) => e.stopPropagation()}
          >
            {row.getValue('name')}
          </Link>
        ),
        size: 200,
        enableHiding: false,
      },
      {
        accessorKey: 'clientWorkload',
        header: 'Client Workload',
        cell: ({ row }) => <span className="text-gray-700">{row.getValue('clientWorkload')}</span>,
      },
      {
        accessorKey: 'trustProvider',
        header: 'Trust Provider',
        cell: ({ row }) => <span className="text-gray-700">{row.getValue('trustProvider')}</span>,
      },
      {
        accessorKey: 'accessCondition',
        header: 'Access Condition',
        cell: ({ row }) => {
          const val = row.getValue('accessCondition') as string
          if (!val) {
            return (
              <span className="px-2 py-0.5 bg-gray-100 text-gray-500 text-[11px] rounded font-medium">
                Optional
              </span>
            )
          }
          return <span className="text-gray-700">{val}</span>
        },
      },
      {
        accessorKey: 'credentialProvider',
        header: 'Credential Provider',
        cell: ({ row }) => <span className="text-gray-700">{row.getValue('credentialProvider')}</span>,
      },
      {
        accessorKey: 'serverWorkload',
        header: 'Server Workload',
        cell: ({ row }) => <span className="text-gray-700">{row.getValue('serverWorkload')}</span>,
      },
      {
        accessorKey: 'status',
        header: 'Status',
        cell: ({ row }) =>
          row.getValue('status') === 'Active' ? (
            <span className="text-green-600 font-medium">Active</span>
          ) : (
            <span className="text-red-500 font-medium">Inactive</span>
          ),
        size: 100,
        filterFn: statusFilterFn,
      },
    ],
    [],
  )

  const facetedFilters: FacetedFilterConfig[] = useMemo(
    () => [{ columnId: 'status', label: 'Status' }],
    [],
  )

  if (isLoading) return <div className="flex-1" />

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-4 shrink-0 h-[44px] border-b border-border bg-white">
        <PageHeader />
      </div>
      <DataTable
      columns={columns}
      data={tableData}
      searchPlaceholder="Filter by name or workload..."
      searchKeys={['name', 'clientWorkload', 'serverWorkload', 'trustProvider', 'credentialProvider']}
      facetedFilters={facetedFilters}
      actionLabel="New Policy"
      onAction={() => navigate('/access-policies/new')}
      onRowClick={(row) => navigate(`/access-policies/${row.id}`)}
      emptyMessage="No access policies found."
    />
    </div>
  )
}

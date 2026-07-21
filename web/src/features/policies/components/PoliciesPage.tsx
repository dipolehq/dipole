import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { Plus, Search } from 'lucide-react'
import { useQuery } from '@tanstack/react-query'
import { PageHeader } from '@/shared/components/PageHeader'
import { PageSizeSelect } from '@/shared/components/PageSizeSelect'
import { usePolicies } from '@/features/policies/hooks/usePolicy'
import { fetchClientWorkloads, fetchServerWorkloads, fetchTrustProviders, fetchAccessConditions, fetchCredentialProviders } from '@/features/policies/api/entityApi'

const PAGE_SIZE_OPTIONS = [15, 25, 50] as const

export function PoliciesPage() {
  const navigate = useNavigate()
  const [searchQuery, setSearchQuery] = useState('')
  const [pageSize, setPageSize] = useState<number>(PAGE_SIZE_OPTIONS[0])
  const [currentPage, setCurrentPage] = useState(1)

  const { data, isLoading } = usePolicies(currentPage, pageSize, searchQuery)
  const items = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  // Fetch entity names for display in columns
  const { data: cwData } = useQuery({
    queryKey: ['client-workloads', 'names'],
    queryFn: () => fetchClientWorkloads({ perPage: 500 }),
    staleTime: 60_000,
  })
  const { data: swData } = useQuery({
    queryKey: ['server-workloads', 'names'],
    queryFn: () => fetchServerWorkloads({ perPage: 500 }),
    staleTime: 60_000,
  })
  const { data: tpData } = useQuery({
    queryKey: ['trust-providers', 'names'],
    queryFn: () => fetchTrustProviders({ perPage: 500 }),
    staleTime: 60_000,
  })
  const { data: acData } = useQuery({
    queryKey: ['access-conditions', 'names'],
    queryFn: () => fetchAccessConditions({ perPage: 500 }),
    staleTime: 60_000,
  })
  const { data: cpData } = useQuery({
    queryKey: ['credential-providers', 'names'],
    queryFn: () => fetchCredentialProviders({ perPage: 500 }),
    staleTime: 60_000,
  })

  // Build name lookup maps
  const cwNames = new Map((cwData?.items ?? []).map((i) => [i.id, i.name]))
  const swNames = new Map((swData?.items ?? []).map((i) => [i.id, i.name]))
  const tpNames = new Map((tpData?.items ?? []).map((i) => [i.id, i.name]))
  const acNames = new Map((acData?.items ?? []).map((i) => [i.id, i.name]))
  const cpNames = new Map((cpData?.items ?? []).map((i) => [i.id, i.name]))

  function resolveName(map: Map<string, string>, id: string | undefined): string {
    if (!id) return '—'
    return map.get(id) ?? '—'
  }

  function resolveNames(map: Map<string, string>, ids: string[]): string {
    if (!ids || ids.length === 0) return '—'
    const names = ids.map((id) => map.get(id)).filter(Boolean)
    if (names.length === 0) return '—'
    return names.join(', ')
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-4 shrink-0 h-[60px] border-b border-border bg-white">
        <PageHeader />
        <div className="flex items-center gap-3">
          <div className="relative">
            <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
            <input
              type="text"
              placeholder="Search"
              value={searchQuery}
              onChange={(e) => {
                setSearchQuery(e.target.value)
                setCurrentPage(1)
              }}
              className="pl-9 pr-3 py-2 border border-border rounded-md text-sm w-48 focus:outline-none focus:ring-1 focus:ring-gray-300 focus:border-gray-400"
            />
          </div>
          <Link
            to="/access-policies/new"
            className="flex items-center gap-1.5 px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white text-sm font-medium rounded-md transition-colors"
          >
            <Plus size={14} /> New
          </Link>
        </div>
      </div>

      {isLoading ? (
        <div className="flex-1" />
      ) : (
        <>
          <div className="flex-1 overflow-y-auto min-h-0">
            <table className="w-full text-xs table-fixed">
              <thead className="sticky top-0 bg-gray-50 z-10">
                <tr className="border-b border-border">
                  <th className="px-3 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-center w-[4%]">#</th>
                  <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Access Policy Name</th>
                  <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Client Workload</th>
                  <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Trust Provider</th>
                  <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Access Condition</th>
                  <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Credential Provider</th>
                  <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Server Workload</th>
                  <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-center w-[8%]">Status</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border bg-white">
                {items.length === 0 && (
                  <tr>
                    <td colSpan={8} className="px-4 py-12 text-center text-gray-400 text-sm">
                      No access policies found.
                    </td>
                  </tr>
                )}
                {items.map((policy, idx) => (
                  <tr
                    key={policy.id}
                    className="hover:bg-gray-50 transition-colors cursor-pointer"
                    onClick={() => navigate(`/access-policies/${policy.id}`)}
                  >
                    <td className="px-3 py-3 text-gray-400 text-center w-10">
                      {(currentPage - 1) * pageSize + idx + 1}
                    </td>
                    <td className="px-4 py-3 font-medium text-gray-900 truncate max-w-[220px]">
                      <Link
                        to={`/access-policies/${policy.id}`}
                        className="hover:text-blue-700 transition-colors"
                        onClick={(e) => e.stopPropagation()}
                      >
                        {policy.name}
                      </Link>
                    </td>
                    <td className="px-4 py-3 text-gray-700 truncate max-w-[160px]">
                      {resolveName(cwNames, policy.clientWorkloadId)}
                    </td>
                    <td className="px-4 py-3 text-gray-700 truncate max-w-[160px]">
                      {resolveNames(tpNames, policy.trustProviderIds)}
                    </td>
                    <td className="px-4 py-3 truncate max-w-[160px]">
                      {policy.accessConditionIds.length > 0 ? (
                        <span className="text-gray-700">{resolveNames(acNames, policy.accessConditionIds)}</span>
                      ) : (
                        <span className="px-2 py-0.5 bg-gray-100 text-gray-500 text-xs rounded font-medium">Optional</span>
                      )}
                    </td>
                    <td className="px-4 py-3 text-gray-700 truncate max-w-[160px]">
                      {resolveNames(cpNames, policy.credentialProviderIds)}
                    </td>
                    <td className="px-4 py-3 text-gray-700 truncate max-w-[160px]">
                      {resolveName(swNames, policy.serverWorkloadId)}
                    </td>
                    <td className="px-3 py-3 text-xs font-medium text-center">
                      {policy.isActive ? (
                        <span className="text-green-600">Active</span>
                      ) : (
                        <span className="text-red-500">Inactive</span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          <div className="flex items-center justify-between px-4 py-2 border-t border-border bg-gray-50 shrink-0">
            <div className="flex items-center gap-2 text-xs text-gray-600">
              <PageSizeSelect value={pageSize} options={PAGE_SIZE_OPTIONS} onChange={(v) => { setPageSize(v); setCurrentPage(1) }} />
              <span>
                Showing {total === 0 ? 0 : (currentPage - 1) * pageSize + 1} -{' '}
                {Math.min(currentPage * pageSize, total)} of {total} Results
              </span>
            </div>
            <div className="flex items-center gap-1">
              <button
                onClick={() => setCurrentPage((p) => Math.max(1, p - 1))}
                disabled={currentPage === 1}
                className="px-2 py-1 text-xs text-gray-500 hover:text-gray-700 disabled:opacity-30"
              >
                ←
              </button>
              {Array.from({ length: totalPages }, (_, i) => (
                <button
                  key={i + 1}
                  onClick={() => setCurrentPage(i + 1)}
                  className={`w-7 h-7 rounded text-xs font-medium transition-colors ${currentPage === i + 1 ? 'bg-orange-500 text-white' : 'text-gray-600 hover:bg-gray-100'}`}
                >
                  {i + 1}
                </button>
              ))}
              <button
                onClick={() => setCurrentPage((p) => Math.min(totalPages, p + 1))}
                disabled={currentPage === totalPages}
                className="px-2 py-1 text-xs text-gray-500 hover:text-gray-700 disabled:opacity-30"
              >
                →
              </button>
            </div>
          </div>
        </>
      )}
    </div>
  )
}

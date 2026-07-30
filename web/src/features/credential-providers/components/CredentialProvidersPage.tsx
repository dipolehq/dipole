import { useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { Plus, Search } from 'lucide-react'
import { PageHeader } from '@/shared/components/PageHeader'
import { useCredentialProviders } from '@/features/credential-providers/hooks/useCredentialProviders'
import { CREDENTIAL_PROVIDER_TABLE_COLUMNS, CREDENTIAL_PROVIDER_PAGE_SIZE_OPTIONS, ENTITY_TABLE_COL_WIDTHS } from '@/shared/constants'
import { PageSizeSelect } from '@/shared/components/PageSizeSelect'

export function CredentialProvidersPage() {
  const navigate = useNavigate()
  const [params, setParams] = useSearchParams()

  const [searchQuery, setSearchQuery] = useState(params.get('search') || '')
  const [pageSize, setPageSizeState] = useState<number>(Number(params.get('perPage')) || CREDENTIAL_PROVIDER_PAGE_SIZE_OPTIONS[0])
  const [currentPage, setCurrentPageState] = useState(Number(params.get('page')) || 1)

  function setCurrentPage(p: number) { setCurrentPageState(p); setParams((prev) => { const next = new URLSearchParams(prev); next.set('page', String(p)); return next }, { replace: true }) }
  function setPageSize(pp: number) { setPageSizeState(pp); setCurrentPageState(1); setParams((prev) => { const next = new URLSearchParams(prev); next.set('perPage', String(pp)); next.set('page', '1'); return next }, { replace: true }) }

  const { data, isLoading } = useCredentialProviders(currentPage, pageSize, searchQuery)

  const items = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  return (
    <div className="flex flex-col h-full">
      {/* Header bar */}
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
                setCurrentPageState(1)
                setParams((prev) => { const next = new URLSearchParams(prev); if (e.target.value) next.set('search', e.target.value); else next.delete('search'); next.set('page', '1'); return next }, { replace: true })
              }}
              className="pl-9 pr-3 py-2 border border-border rounded-md text-sm w-48 focus:outline-none focus:ring-1 focus:ring-gray-300 focus:border-gray-400"
            />
          </div>
          <Link
            to="/credential-providers/new"
            className="flex items-center gap-1.5 px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white text-sm font-medium rounded-md transition-colors"
          >
            <Plus size={14} />
            New
          </Link>
        </div>
      </div>

      {/* Content */}
      {isLoading ? (
        <div className="flex-1 flex items-center justify-center">
          
        </div>
      ) : (
        <>
          {/* Table — full bleed */}
          <div className="flex-1 overflow-y-auto min-h-0">
            <table className="w-full text-xs table-fixed">
              <thead className="sticky top-0 bg-gray-50 z-10">
                <tr className="border-b border-border">
                  {CREDENTIAL_PROVIDER_TABLE_COLUMNS.map((col) => (
                    <th
                      key={col.key}
                      className={`px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider ${ENTITY_TABLE_COL_WIDTHS[col.key] ?? ''} ${
                        col.key === 'tags' ||
                        col.key === 'policies' ||
                        col.key === 'status' ||
                        col.key === 'number'
                          ? 'text-center'
                          : 'text-left'
                      }`}
                    >
                      {col.label}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody className="divide-y divide-border bg-white">
                {items.length === 0 && (
                  <tr>
                    <td
                      colSpan={CREDENTIAL_PROVIDER_TABLE_COLUMNS.length}
                      className="px-4 py-12 text-center text-gray-400 text-sm"
                    >
                      No credential providers found.
                    </td>
                  </tr>
                )}
                {items.map((provider, idx) => (
                  <tr
                    key={provider.id}
                    className="hover:bg-gray-50 transition-colors cursor-pointer"
                    onClick={() => navigate(`/credential-providers/${provider.id}`)}
                  >
                    <td className="px-3 py-3 text-gray-400 text-center w-10">
                      {(currentPage - 1) * pageSize + idx + 1}
                    </td>
                    <td className="px-4 py-3 font-medium text-gray-900 truncate max-w-[160px]">
                      <Link
                        to={`/credential-providers/${provider.id}`}
                        className="hover:text-blue-700 transition-colors"
                        onClick={(e) => e.stopPropagation()}
                      >
                        {provider.name}
                      </Link>
                    </td>
                    <td className="px-4 py-3 text-gray-700 truncate max-w-[160px]">
                      {provider.type}
                    </td>
                    <td className="px-4 py-3 text-gray-700 truncate max-w-[260px]">
                      {provider.description}
                    </td>
                    <td className="px-4 py-3 text-gray-700 text-center">{provider.tags}</td>
                    <td className="px-4 py-3 text-gray-700 text-center">{provider.policies}</td>
                    <td className="px-3 py-3 text-xs font-medium text-center">
                      {provider.isActive ? (
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

          {/* Pagination footer */}
          <div className="flex items-center justify-between px-4 py-2 border-t border-border bg-gray-50 shrink-0">
            <div className="flex items-center gap-2 text-xs text-gray-600">
              <PageSizeSelect value={pageSize} options={CREDENTIAL_PROVIDER_PAGE_SIZE_OPTIONS} onChange={(v) => { setPageSize(v) }} />
              <span>
                Showing {total === 0 ? 0 : (currentPage - 1) * pageSize + 1} -{' '}
                {Math.min(currentPage * pageSize, total)} of {total} Results
              </span>
            </div>
            <div className="flex items-center gap-1">
              <button
                onClick={() => setCurrentPage(Math.max(1, currentPage - 1))}
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
                onClick={() => setCurrentPage(Math.min(totalPages, currentPage + 1))}
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

import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { Plus, Search } from 'lucide-react'
import { ResourceSetDropdown } from '@/shared/components/ResourceSetDropdown'
import { PageHeader } from '@/shared/components/PageHeader'
import { useClientWorkloads } from '@/features/client-workloads/hooks/useClientWorkloads'
import { CLIENT_WORKLOAD_TABLE_COLUMNS, PAGE_SIZE_OPTIONS } from '@/data/client-workloads'
import { ENTITY_TABLE_COL_WIDTHS } from '@/data/shared/tableWidths'

export function ClientWorkloadsPage() {
  const navigate = useNavigate()
  const { data, isLoading } = useClientWorkloads()

  const [searchQuery, setSearchQuery] = useState('')
  const [pageSize, setPageSize] = useState<number>(PAGE_SIZE_OPTIONS[0])
  const [currentPage, setCurrentPage] = useState(1)

  const items = data?.items ?? []

  const filtered = items.filter((w) =>
    w.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
    w.type.toLowerCase().includes(searchQuery.toLowerCase()) ||
    w.description.toLowerCase().includes(searchQuery.toLowerCase()),
  )

  const totalPages = Math.max(1, Math.ceil(filtered.length / pageSize))
  const paginated = filtered.slice((currentPage - 1) * pageSize, currentPage * pageSize)

  return (
    <div className="px-4 py-3 flex flex-col h-full">
      <div className="flex items-center justify-between mb-4 shrink-0 min-h-[36px]">
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
            to="/client-workloads/new"
            className="flex items-center gap-1.5 px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white text-sm font-medium rounded-md transition-colors"
          >
            <Plus size={14} />
            New
          </Link>
          <ResourceSetDropdown />
        </div>
      </div>

      {isLoading ? (
        <div className="flex-1 flex items-center justify-center">
          <p className="text-sm text-gray-400">Loading client workloads…</p>
        </div>
      ) : (
        <>
          <div className="bg-white rounded-t-lg border border-b-0 border-border flex-1 overflow-y-auto min-h-0">
            <table className="w-full text-xs table-fixed">
              <thead className="sticky top-0 bg-gray-50 z-10">
                <tr className="border-b border-border">
                  {CLIENT_WORKLOAD_TABLE_COLUMNS.map((col) => (
                    <th
                      key={col.key}
                      className={`px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider ${ENTITY_TABLE_COL_WIDTHS[col.key] ?? ''} ${
                        col.key === 'tags' || col.key === 'policies' || col.key === 'status' || col.key === 'number' ? 'text-center' : 'text-left'
                      }`}
                    >
                      {col.label}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {paginated.length === 0 && (
                  <tr>
                    <td colSpan={CLIENT_WORKLOAD_TABLE_COLUMNS.length} className="px-4 py-12 text-center text-gray-400 text-sm">
                      No client workloads found.
                    </td>
                  </tr>
                )}
                {paginated.map((workload, idx) => (
                  <tr
                    key={workload.id}
                    className="hover:bg-gray-50 transition-colors cursor-pointer"
                    onClick={() => navigate(`/client-workloads/${workload.id}`)}
                  >
                    <td className="px-3 py-3 text-gray-400 text-center w-10">{(currentPage - 1) * pageSize + idx + 1}</td>
                    <td className="px-4 py-3 font-medium text-gray-900 truncate max-w-[160px]">
                      <Link
                        to={`/client-workloads/${workload.id}`}
                        className="hover:text-blue-700 transition-colors"
                        onClick={(e) => e.stopPropagation()}
                      >
                        {workload.name}
                      </Link>
                    </td>
                    <td className="px-4 py-3 text-gray-700 truncate max-w-[160px]">
                      {workload.type}
                    </td>
                    <td className="px-4 py-3 text-gray-700 truncate max-w-[260px]">
                      {workload.description}
                    </td>
                    <td className="px-4 py-3 text-gray-700 text-center">
                      {workload.tags}
                    </td>
                    <td className="px-4 py-3 text-gray-700 text-center">
                      {workload.policies}
                    </td>
                    <td className="px-3 py-3 text-xs font-medium text-center">
                      {workload.isActive ? (
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

          <div className="flex items-center justify-between px-4 py-2 border border-border rounded-b-lg bg-gray-50 shrink-0">
            <div className="flex items-center gap-2 text-xs text-gray-600">
              <select
                value={pageSize}
                onChange={(e) => {
                  setPageSize(Number(e.target.value))
                  setCurrentPage(1)
                }}
                className="border border-border rounded px-2 py-1 text-xs"
              >
                {PAGE_SIZE_OPTIONS.map((size) => (
                  <option key={size} value={size}>{size}</option>
                ))}
              </select>
              <span>
                Showing {filtered.length === 0 ? 0 : (currentPage - 1) * pageSize + 1} - {Math.min(currentPage * pageSize, filtered.length)} of {filtered.length} Results
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
                  className={`w-7 h-7 rounded text-xs font-medium transition-colors ${
                    currentPage === i + 1
                      ? 'bg-orange-500 text-white'
                      : 'text-gray-600 hover:bg-gray-100'
                  }`}
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

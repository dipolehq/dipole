import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { fetchClientWorkloads } from '@/features/policies/api/entityApi'

interface ExistingWorkloadSelectorProps {
  selectedId: string
  onSelect: (id: string) => void
  onCancel: () => void
}

export function ExistingWorkloadSelector({
  selectedId,
  onSelect,
  onCancel,
}: ExistingWorkloadSelectorProps) {
  const [search, setSearch] = useState('')
  const [selected, setSelected] = useState(selectedId)
  const [page, setPage] = useState(1)
  const pageSize = 10

  const { data, isLoading } = useQuery({
    queryKey: ['client-workloads', 'selector', page, pageSize, search],
    queryFn: () => fetchClientWorkloads({ page, perPage: pageSize, search }),
  })

  const items = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  return (
    <div className="flex flex-col" style={{ height: 'calc(100vh - 260px)' }}>
      <div className="flex items-center justify-between mb-4 shrink-0">
        <input
          type="text"
          placeholder="Search..."
          value={search}
          onChange={(e) => {
            setSearch(e.target.value)
            setPage(1)
          }}
          className="px-3 py-1.5 border border-border rounded-md text-xs w-64 focus:outline-none focus:ring-1 focus:ring-gray-300"
        />
        <div className="flex items-center gap-3">
          <button
            onClick={onCancel}
            className="text-xs text-gray-500 hover:text-gray-700 font-medium"
          >
            Cancel
          </button>
          <button
            onClick={() => {
              if (selected) onSelect(selected)
            }}
            className="px-3 py-1.5 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors"
          >
            Use Selected
          </button>
        </div>
      </div>

      <div className="border border-border rounded-t-md flex-1 overflow-y-auto">
        <table className="w-full text-xs table-fixed">
          <thead className="sticky top-0 bg-gray-50 z-10">
            <tr className="border-b border-border">
              <th className="w-8 px-1 py-2" />
              <th className="w-[25%] px-3 py-2 text-left font-semibold text-gray-500 uppercase tracking-wide">
                Name
              </th>
              <th className="w-[20%] px-3 py-2 text-left font-semibold text-gray-500 uppercase tracking-wide">
                Type
              </th>
              <th className="w-[30%] px-3 py-2 text-left font-semibold text-gray-500 uppercase tracking-wide">
                Description
              </th>
              <th className="w-[10%] px-3 py-2 text-center font-semibold text-gray-500 uppercase tracking-wide">
                Policies
              </th>
              <th className="w-[10%] px-3 py-2 text-center font-semibold text-gray-500 uppercase tracking-wide">
                Status
              </th>
            </tr>
          </thead>
          <tbody className="divide-y divide-border">
            {isLoading && (
              <tr>
                <td colSpan={6} className="px-3 py-6 text-center text-gray-400 text-xs">
                  Loading…
                </td>
              </tr>
            )}
            {!isLoading &&
              items.map((w) => (
                <tr
                  key={w.id}
                  onClick={() => setSelected(w.id)}
                  className={`cursor-pointer transition-colors ${selected === w.id ? 'bg-orange-50' : 'hover:bg-gray-50'}`}
                >
                  <td className="px-1 py-2.5 text-center">
                    <input
                      type="radio"
                      checked={selected === w.id}
                      onChange={() => setSelected(w.id)}
                      className="w-3.5 h-3.5 text-orange-500 focus:ring-orange-200"
                    />
                  </td>
                  <td className="px-3 py-2.5 text-gray-900 font-medium truncate">{w.name}</td>
                  <td className="px-3 py-2.5 text-gray-600 truncate">
                    {w.identifiers?.[0]?.type || '—'}
                  </td>
                  <td className="px-3 py-2.5 text-gray-500 truncate">{w.description || '—'}</td>
                  <td className="px-3 py-2.5 text-center text-gray-600">{w.policyCount}</td>
                  <td className="px-3 py-2.5 text-center">
                    <span
                      className={`text-[10px] font-medium ${w.isActive ? 'text-green-600' : 'text-red-500'}`}
                    >
                      {w.isActive ? 'Active' : 'Inactive'}
                    </span>
                  </td>
                </tr>
              ))}
            {!isLoading && items.length === 0 && (
              <tr>
                <td colSpan={6} className="px-3 py-6 text-center text-gray-400 text-xs">
                  No matching workloads found.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      <div className="flex items-center justify-between px-3 py-2 border border-t-0 border-border rounded-b-md bg-gray-50 shrink-0">
        <span className="text-[10px] text-gray-500">
          Showing {total === 0 ? 0 : (page - 1) * pageSize + 1}–
          {Math.min(page * pageSize, total)} of {total}
        </span>
        <div className="flex items-center gap-1">
          <button
            onClick={() => setPage((p) => Math.max(1, p - 1))}
            disabled={page === 1}
            className="px-1.5 py-0.5 text-[10px] text-gray-500 disabled:opacity-30"
          >
            ←
          </button>
          {Array.from({ length: Math.min(totalPages, 5) }, (_, i) => (
            <button
              key={i + 1}
              onClick={() => setPage(i + 1)}
              className={`w-5 h-5 rounded text-[10px] font-medium ${page === i + 1 ? 'bg-orange-500 text-white' : 'text-gray-600 hover:bg-gray-100'}`}
            >
              {i + 1}
            </button>
          ))}
          <button
            onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
            disabled={page === totalPages}
            className="px-1.5 py-0.5 text-[10px] text-gray-500 disabled:opacity-30"
          >
            →
          </button>
        </div>
      </div>
    </div>
  )
}

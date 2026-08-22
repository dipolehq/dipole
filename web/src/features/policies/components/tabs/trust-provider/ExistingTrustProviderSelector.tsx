import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { fetchTrustProviders } from '@/features/policies/api/entityApi'

interface ExistingTrustProviderSelectorProps {
  selectedIds: string[]
  onToggle: (id: string) => void
  onCancel: () => void
}

export function ExistingTrustProviderSelector({
  selectedIds,
  onToggle,
  onCancel,
}: ExistingTrustProviderSelectorProps) {
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const [localSelected, setLocalSelected] = useState<Set<string>>(new Set(selectedIds))
  const pageSize = 10

  const { data, isLoading } = useQuery({
    queryKey: ['trust-providers', 'selector', page, pageSize, search],
    queryFn: () => fetchTrustProviders({ page, perPage: pageSize, search }),
  })

  const items = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  function toggleLocal(id: string) {
    setLocalSelected((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  function handleUseSelected() {
    // onToggle flips membership (adds if absent, removes if present), so
    // propagate every id whose selection state actually changed — both
    // newly checked ids AND ids the user unchecked. Only calling onToggle
    // for additions silently dropped removals, leaving stale trust
    // provider ids attached to the policy after save.
    const changedIds = [
      ...Array.from(localSelected).filter((id) => !selectedIds.includes(id)),
      ...selectedIds.filter((id) => !localSelected.has(id)),
    ]
    for (const id of changedIds) {
      onToggle(id)
    }
    onCancel()
  }

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
            onClick={handleUseSelected}
            className="px-3 py-1.5 bg-gray-800 hover:bg-gray-900 text-white text-xs font-medium rounded-md transition-colors"
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
              <th className="w-[25%] px-3 py-2 text-left font-semibold text-gray-500 uppercase tracking-wide">
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
              items.map((tp) => {
                const isSelected = localSelected.has(tp.id)
                return (
                  <tr
                    key={tp.id}
                    onClick={() => toggleLocal(tp.id)}
                    className={`cursor-pointer transition-colors ${isSelected ? 'bg-gray-50' : 'hover:bg-gray-50'}`}
                  >
                    <td className="px-1 py-2.5 text-center">
                      <input
                        type="checkbox"
                        checked={isSelected}
                        onChange={() => toggleLocal(tp.id)}
                        className="w-3.5 h-3.5 text-gray-600 focus:ring-gray-200"
                      />
                    </td>
                    <td className="px-3 py-2.5 text-gray-900 font-medium truncate">{tp.name}</td>
                    <td className="px-3 py-2.5 text-gray-600 truncate">{tp.type || '—'}</td>
                    <td className="px-3 py-2.5 text-gray-500 truncate">
                      {tp.description || '—'}
                    </td>
                    <td className="px-3 py-2.5 text-center text-gray-600">{tp.policyCount}</td>
                    <td className="px-3 py-2.5 text-center">
                      <span
                        className={`text-[10px] font-medium ${tp.isActive ? 'text-green-600' : 'text-red-500'}`}
                      >
                        {tp.isActive ? 'Active' : 'Inactive'}
                      </span>
                    </td>
                  </tr>
                )
              })}
            {!isLoading && items.length === 0 && (
              <tr>
                <td colSpan={6} className="px-3 py-6 text-center text-gray-400 text-xs">
                  No matching trust providers found.
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
              className={`w-5 h-5 rounded text-[10px] font-medium ${page === i + 1 ? 'bg-gray-800 text-white' : 'text-gray-600 hover:bg-gray-100'}`}
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

import { useState, useRef } from 'react'
import { LayoutGrid, ChevronDown } from 'lucide-react'
import { useClickOutside } from '@/shared/hooks/useClickOutside'

const RESOURCE_SETS = [
  { id: 'ffffffff-ffff-ffff-ffff-ffffffffffff', name: 'Default' },
  { id: 'a1b2c3d4-0000-0000-0000-000000000001', name: 'Production' },
  { id: 'a1b2c3d4-0000-0000-0000-000000000002', name: 'Staging' },
]

export function ResourceSetDropdown() {
  const [selected, setSelected] = useState(RESOURCE_SETS[0]!)
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useClickOutside(ref, () => setOpen(false))

  return (
    <div ref={ref} className="relative">
      <button
        onClick={() => setOpen(!open)}
        className="flex items-center justify-between gap-2 px-4 py-2 bg-[#28282B] text-white text-sm font-medium rounded-md hover:bg-[#333336] transition-colors w-[160px]"
      >
        <LayoutGrid size={14} />
        <span>{selected.name}</span>
        <ChevronDown size={14} />
      </button>

      {open && (
        <div className="absolute top-full right-0 mt-1 bg-white border border-border rounded-md shadow-lg py-1 z-50 min-w-[160px]">
          {RESOURCE_SETS.map((rs) => (
            <button
              key={rs.id}
              onClick={() => {
                setSelected(rs)
                setOpen(false)
              }}
              className={`w-full text-left px-3 py-2 text-sm transition-colors ${
                selected.id === rs.id
                  ? 'bg-gray-50 text-gray-900 font-medium'
                  : 'text-gray-600 hover:bg-gray-50'
              }`}
            >
              {rs.name}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}

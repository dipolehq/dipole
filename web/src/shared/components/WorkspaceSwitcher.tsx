import { useState, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { ChevronsUpDown, Check, Plus } from 'lucide-react'
import { useClickOutside } from '@/shared/hooks/useClickOutside'
import { useResourceSetContext } from '@/shared/context/ResourceSetContext'

export function WorkspaceSwitcher() {
  const { current, all, setCurrent } = useResourceSetContext()
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)
  const navigate = useNavigate()

  useClickOutside(ref, () => setOpen(false))

  return (
    <div ref={ref} className="relative">
      <button
        onClick={() => setOpen(!open)}
        className="flex items-center gap-3 w-full px-3 py-2 rounded-md text-[13px] font-medium text-gray-600 hover:bg-gray-50 hover:text-gray-900 transition-colors"
      >
        <svg className="w-[15px] h-[15px] shrink-0" fill="none" stroke="currentColor" strokeWidth="1.75" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" d="M2 9.5L12 4l10 5.5M2 9.5v5l10 5.5m-10-10.5L12 15m10-5.5v5L12 20m10-10.5L12 15" />
        </svg>
        <span className="flex-1 text-left truncate">{current.name}</span>
        <ChevronsUpDown size={14} className="text-gray-400 shrink-0" />
      </button>

      {open && (
        <div className="absolute left-0 right-0 top-full mt-1 bg-white rounded-lg shadow-xl border border-gray-200 py-1 z-50 max-h-64 overflow-y-auto">
          {all.map((rs) => (
            <button
              key={rs.id}
              onClick={() => {
                setCurrent(rs)
                setOpen(false)
              }}
              className="flex items-center gap-2.5 w-full px-3 py-2 text-left hover:bg-gray-50 transition-colors"
            >
              <span className="flex-1 text-[12px] text-gray-700 truncate">{rs.name}</span>
              {current.id === rs.id && (
                <Check size={13} className="text-orange-500 shrink-0" />
              )}
            </button>
          ))}

          <div className="border-t border-gray-200 mt-1 pt-1">
            <button
              onClick={() => { setOpen(false); navigate('/administration/resource_sets/new') }}
              className="flex items-center gap-2.5 w-full px-3 py-2 text-left hover:bg-gray-50 transition-colors"
            >
              <Plus size={13} className="text-gray-400" />
              <span className="text-[12px] text-gray-400">New Resource Set</span>
            </button>
          </div>
        </div>
      )}
    </div>
  )
}

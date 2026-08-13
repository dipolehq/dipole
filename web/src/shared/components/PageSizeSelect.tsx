import { useState, useRef } from 'react'
import { ChevronDown } from 'lucide-react'
import { useClickOutside } from '@/shared/hooks/useClickOutside'

interface PageSizeSelectProps {
  value: number
  options: readonly number[]
  onChange: (value: number) => void
}

export function PageSizeSelect({ value, options, onChange }: PageSizeSelectProps) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useClickOutside(ref, () => setOpen(false))

  return (
    <div ref={ref} className="relative">
      <button
        type="button"
        onClick={() => setOpen(!open)}
        className="flex items-center gap-1.5 px-2.5 py-1 border border-border rounded-md text-xs text-gray-700 hover:border-gray-400 transition-colors"
      >
        <span>{value}</span>
        <ChevronDown size={12} className={`text-gray-400 transition-transform ${open ? 'rotate-180' : ''}`} />
      </button>

      {open && (
        <div className="absolute bottom-full left-0 mb-1 bg-white border border-border rounded-md shadow-lg py-1 z-50 min-w-[48px]">
          {options.map((size) => (
            <button
              key={size}
              type="button"
              onClick={() => {
                onChange(size)
                setOpen(false)
              }}
              className={`w-full text-left px-3 py-1.5 text-xs transition-colors ${
                value === size
                  ? 'text-gray-700 font-medium bg-gray-50'
                  : 'text-gray-600 hover:bg-gray-50'
              }`}
            >
              {size}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}

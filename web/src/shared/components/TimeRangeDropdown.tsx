import { useState, useRef } from 'react'
import { ChevronDown } from 'lucide-react'
import { useClickOutside } from '@/shared/hooks/useClickOutside'

const TIME_OPTIONS = [
  { label: 'Last 1 Hour', value: '1h' },
  { label: 'Last 6 Hours', value: '6h' },
  { label: 'Last 24 Hours', value: '24h' },
  { label: 'Last 7 Days', value: '7d' },
  { label: 'Last 30 Days', value: '30d' },
] as const

interface TimeRangeDropdownProps {
  /** Current selected value (controlled) */
  defaultValue?: string
  onChange?: (value: string) => void
}

export function TimeRangeDropdown({ defaultValue = '6h', onChange }: TimeRangeDropdownProps) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useClickOutside(ref, () => setOpen(false))

  const selectedLabel = TIME_OPTIONS.find((o) => o.value === defaultValue)?.label ?? 'Last 6 Hours'

  return (
    <div ref={ref} className="relative inline-block">
      <button
        onClick={() => setOpen(!open)}
        className="flex items-center gap-1 text-xs text-gray-600 hover:text-gray-700 transition-colors"
      >
        <span>{selectedLabel}</span>
        <ChevronDown size={12} />
      </button>

      {open && (
        <div className="absolute top-full left-0 mt-1 bg-white border border-border rounded-md shadow-lg py-1 z-50 min-w-[140px]">
          {TIME_OPTIONS.map((option) => (
            <button
              key={option.value}
              onClick={() => {
                onChange?.(option.value)
                setOpen(false)
              }}
              className={`w-full text-left px-3 py-1.5 text-xs transition-colors ${
                defaultValue === option.value
                  ? 'bg-gray-50 text-gray-900 font-medium'
                  : 'text-gray-600 hover:bg-gray-50'
              }`}
            >
              {option.label}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}

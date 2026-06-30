import { useState, useRef } from 'react'
import { ChevronDown } from 'lucide-react'
import { useClickOutside } from '@/shared/hooks/useClickOutside'

interface DropdownOption {
  value: string
  label: string
}

interface DropdownProps {
  options: DropdownOption[]
  value: string
  onChange: (value: string) => void
  placeholder?: string
  className?: string
}

export function Dropdown({ options, value, onChange, placeholder = 'Please select one', className = '' }: DropdownProps) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useClickOutside(ref, () => setOpen(false))

  const selectedLabel = options.find((o) => o.value === value)?.label

  return (
    <div ref={ref} className={`relative ${className}`}>
      <button
        onClick={() => setOpen(!open)}
        className={`w-full flex items-center justify-between px-3 py-2 border rounded-md text-sm transition-colors ${
          open ? 'border-orange-400 ring-2 ring-orange-100' : 'border-border'
        } ${selectedLabel ? 'text-gray-900' : 'text-gray-400'}`}
      >
        <span className="truncate">{selectedLabel ?? placeholder}</span>
        <ChevronDown size={14} className={`shrink-0 text-orange-500 transition-transform ${open ? 'rotate-180' : ''}`} />
      </button>

      {open && (
        <div className="absolute top-full left-0 right-0 mt-1 bg-white border border-border rounded-md shadow-lg z-50 max-h-56 overflow-y-auto">
          {options.map((opt) => (
            <button
              key={opt.value}
              onClick={() => {
                onChange(opt.value)
                setOpen(false)
              }}
              className={`w-full text-left px-4 py-2.5 text-sm transition-colors ${
                value === opt.value
                  ? 'bg-gray-100 text-orange-600 font-medium'
                  : 'text-gray-600 hover:bg-gray-50'
              }`}
            >
              {opt.label}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}

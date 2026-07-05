import { useState, useId, useRef } from 'react'
import { ChevronDown } from 'lucide-react'
import { useClickOutside } from '@/shared/hooks/useClickOutside'

interface FloatingInputProps {
  label: string
  value: string
  onChange?: (value: string) => void
  disabled?: boolean
  type?: string
  required?: boolean
}

export function FloatingInput({
  label,
  value,
  onChange,
  disabled = false,
  type = 'text',
  required = false,
}: FloatingInputProps) {
  const [focused, setFocused] = useState(false)
  const id = useId()
  const hasValue = value.length > 0
  const isFloating = focused || hasValue

  return (
    <div className="relative">
      <input
        id={id}
        type={type}
        value={value}
        onChange={(e) => onChange?.(e.target.value)}
        disabled={disabled}
        onFocus={() => setFocused(true)}
        onBlur={() => setFocused(false)}
        className={`peer w-full px-3 pt-4 pb-1.5 text-sm border rounded-md outline-none transition-colors ${
          focused ? 'border-orange-500 ring-1 ring-orange-500' : 'border-border'
        } ${disabled ? 'bg-gray-50 text-gray-600 cursor-default' : 'bg-white text-gray-900'}`}
        placeholder=" "
      />
      <label
        htmlFor={id}
        className={`absolute left-3 transition-all pointer-events-none ${
          isFloating ? '-top-2 text-[10px] bg-white px-1' : 'top-2.5 text-sm'
        } ${focused ? 'text-orange-500' : 'text-gray-400'}`}
      >
        {label}{required && <span className="text-red-500 ml-0.5">*</span>}
      </label>
    </div>
  )
}

interface FloatingTextareaProps {
  label: string
  value: string
  onChange?: (value: string) => void
  disabled?: boolean
  rows?: number
  required?: boolean
}

export function FloatingTextarea({
  label,
  value,
  onChange,
  disabled = false,
  rows = 3,
  required = false,
}: FloatingTextareaProps) {
  const [focused, setFocused] = useState(false)
  const id = useId()
  const hasValue = value.length > 0
  const isFloating = focused || hasValue

  return (
    <div className="relative">
      <textarea
        id={id}
        value={value}
        onChange={(e) => onChange?.(e.target.value)}
        disabled={disabled}
        onFocus={() => setFocused(true)}
        onBlur={() => setFocused(false)}
        rows={rows}
        className={`peer w-full px-3 pt-4 pb-1.5 text-sm border rounded-md outline-none transition-colors resize-y ${
          focused ? 'border-orange-500 ring-1 ring-orange-500' : 'border-border'
        } ${disabled ? 'bg-gray-50 text-gray-600 cursor-default' : 'bg-white text-gray-900'}`}
        placeholder=" "
      />
      <label
        htmlFor={id}
        className={`absolute left-3 transition-all pointer-events-none ${
          isFloating ? '-top-2 text-[10px] bg-white px-1' : 'top-2.5 text-sm'
        } ${focused ? 'text-orange-500' : 'text-gray-400'}`}
      >
        {label}{required && <span className="text-red-500 ml-0.5">*</span>}
      </label>
    </div>
  )
}

interface FloatingDropdownProps {
  label: string
  options: { value: string; label: string }[]
  value: string
  onChange: (value: string) => void
  disabled?: boolean
  required?: boolean
  className?: string
}

export function FloatingDropdown({
  label,
  options,
  value,
  onChange,
  disabled = false,
  required = false,
  className = '',
}: FloatingDropdownProps) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)
  useClickOutside(ref, () => setOpen(false))

  const selectedLabel = options.find((o) => o.value === value)?.label
  const hasValue = value.length > 0
  const isFloating = open || hasValue

  return (
    <div ref={ref} className={`relative ${className}`}>
      <button
        type="button"
        onClick={() => !disabled && setOpen(!open)}
        disabled={disabled}
        className={`w-full h-[44px] flex items-center justify-between px-3 text-sm border rounded-md transition-colors text-left ${
          open ? 'border-orange-500 ring-1 ring-orange-500' : 'border-border'
        } ${disabled ? 'bg-gray-50 text-gray-600 cursor-default' : 'bg-white text-gray-900'}`}
      >
        <span className={`truncate leading-[24px] ${selectedLabel ? 'text-gray-900' : 'text-transparent'}`}>{selectedLabel ?? 'x'}</span>
        <ChevronDown size={14} className={`shrink-0 -mt-2 transition-transform ${open ? 'rotate-180 text-orange-500' : 'text-gray-400'}`} />
      </button>
      <label
        className={`absolute left-3 transition-all pointer-events-none ${
          isFloating ? '-top-2 text-[10px] bg-white px-1' : 'top-2.5 text-sm'
        } ${open ? 'text-orange-500' : 'text-gray-400'}`}
      >
        {label}{required && <span className="text-red-500 ml-0.5">*</span>}
      </label>

      {open && (
        <div className="absolute top-full left-0 right-0 mt-1 bg-white border border-border rounded-md shadow-lg z-50 max-h-56 overflow-y-auto">
          {options.map((opt) => (
            <button
              key={opt.value}
              type="button"
              onClick={() => {
                onChange(opt.value)
                setOpen(false)
              }}
              className={`w-full text-left px-4 py-2.5 text-sm transition-colors ${
                value === opt.value ? 'bg-gray-100 text-orange-600 font-medium' : 'text-gray-600 hover:bg-gray-50'
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

import { Dropdown } from '@/shared/components/Dropdown'

interface SelectTabProps {
  title: string
  placeholder: string
  options: { id: string; name: string }[]
  value: string
  onChange: (value: string) => void
}

export function SelectTab({ title, placeholder, options, value, onChange }: SelectTabProps) {
  return (
    <div className="space-y-6">
      <h3 className="text-sm font-semibold text-gray-900">{title}</h3>
      <div>
        <label className="block text-xs font-medium text-gray-700 mb-1.5">{title}</label>
        <Dropdown
          options={options.map((o) => ({ value: o.id, label: o.name }))}
          value={value}
          onChange={onChange}
          placeholder={placeholder}
        />
        {value && (
          <p className="mt-2 text-xs text-green-600">
            ✓ {options.find((o) => o.id === value)?.name} selected
          </p>
        )}
      </div>
    </div>
  )
}

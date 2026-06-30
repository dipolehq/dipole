interface MultiSelectTabProps {
  title: string
  options: { id: string; name: string }[]
  selected: string[]
  onToggle: (id: string) => void
}

export function MultiSelectTab({ title, options, selected, onToggle }: MultiSelectTabProps) {
  return (
    <div className="space-y-6">
      <h3 className="text-sm font-semibold text-gray-900">{title}</h3>
      <div>
        <label className="block text-xs font-medium text-gray-700 mb-1.5">{title}</label>
        {selected.length > 0 && (
          <div className="flex flex-wrap gap-1.5 mb-3">
            {selected.map((id) => {
              const opt = options.find((o) => o.id === id)
              return (
                <span key={id} className="inline-flex items-center gap-1 px-2.5 py-1 bg-brand-50 text-brand-700 text-xs rounded-md">
                  {opt?.name}
                  <button onClick={() => onToggle(id)} className="text-brand-400 hover:text-brand-700 ml-0.5">×</button>
                </span>
              )
            })}
          </div>
        )}
        <div className="border border-border rounded-md divide-y divide-border">
          {options.map((opt) => (
            <label key={opt.id} className="flex items-center gap-3 px-4 py-2.5 hover:bg-gray-50 cursor-pointer transition-colors">
              <input type="checkbox" checked={selected.includes(opt.id)} onChange={() => onToggle(opt.id)} className="w-3.5 h-3.5 rounded border-gray-300 text-brand-500 focus:ring-gray-300" />
              <span className="text-sm text-gray-700">{opt.name}</span>
            </label>
          ))}
        </div>
      </div>
    </div>
  )
}

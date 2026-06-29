import { TimeRangeDropdown } from '@/shared/components/TimeRangeDropdown'

interface ConnectionsChartProps {
  title: string
  subtitle: string
  data: { name: string; value: number }[]
  barColor?: string
}

export function ConnectionsChart({ title, subtitle, data, barColor = 'bg-sky-100' }: ConnectionsChartProps) {
  const max = Math.max(...data.map((d) => d.value))

  return (
    <div className="bg-white rounded-lg border border-border p-5">
      <div className="mb-4">
        <h3 className="text-sm text-gray-600">
          {title} / <span className="font-bold text-gray-900">{subtitle}</span>
        </h3>
        <TimeRangeDropdown defaultValue="24h" />
      </div>
      <div className="space-y-1.5">
        {data.map((item) => (
          <div key={item.name} className="relative h-7 flex items-center">
            <div
              className={`absolute inset-y-0 left-0 ${barColor} rounded-sm`}
              style={{ width: `${(item.value / max) * 100}%` }}
            />
            <span className="relative z-10 text-xs text-gray-700 pl-2 truncate">
              {item.name}
            </span>
            <span className="relative z-10 text-xs font-medium text-gray-700 ml-auto pr-2">
              {item.value}
            </span>
          </div>
        ))}
      </div>
    </div>
  )
}

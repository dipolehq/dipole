import { TimeRangeDropdown } from '@/shared/components/TimeRangeDropdown'

interface ConnectionsChartProps {
  title: string
  subtitle: string
  data: { name: string; value: number }[]
  barColor?: string
  timeRange?: string
  onTimeRangeChange?: (range: string) => void
}

export function ConnectionsChart({ title, subtitle, data, barColor = 'bg-sky-100', timeRange = '24h', onTimeRangeChange }: ConnectionsChartProps) {
  const max = Math.max(1, ...data.map((d) => d.value))
  const hasData = data.length > 0

  return (
    <div className="bg-white rounded-lg border border-border p-5 h-[340px] flex flex-col">
      <div className="mb-4 shrink-0">
        <h3 className="text-sm text-gray-600">
          {title} / <span className="font-bold text-gray-900">{subtitle}</span>
        </h3>
        <TimeRangeDropdown defaultValue={timeRange} onChange={onTimeRangeChange} />
      </div>
      <div className="flex-1 relative overflow-hidden">
        {!hasData && (
          <>
            {/* Skeleton bars */}
            <div className="space-y-2 opacity-30">
              {Array.from({ length: 7 }, (_, i) => (
                <div key={i} className="relative h-7 flex items-center">
                  <div className="absolute inset-y-0 left-0 right-0 bg-gray-200 rounded-sm" style={{ width: `${100 - i * 5}%` }} />
                </div>
              ))}
            </div>
            {/* Overlay */}
            <div className="absolute inset-0 flex items-center justify-center">
              <span className="text-[13px] text-gray-300 font-semibold select-none pointer-events-none">No Data Available</span>
            </div>
          </>
        )}
        {hasData && (
          <div className="space-y-1.5 overflow-y-auto h-full">
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
        )}
      </div>
    </div>
  )
}

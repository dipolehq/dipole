import { useMemo } from 'react'
import ReactECharts from 'echarts-for-react'
import { RotateCw } from 'lucide-react'
import { TimeRangeDropdown } from '@/shared/components/TimeRangeDropdown'

interface WorkloadEventsChartProps {
  data?: { info: number[]; warning: number[]; error: number[]; timestamps?: string[] }
  timeRange?: string
  onTimeRangeChange?: (range: string) => void
  onRefresh?: () => void
}

function formatLabel(iso: string, timeRange: string): string {
  const d = new Date(iso)
  if (timeRange === '7d' || timeRange === '30d') {
    return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
  }
  return d.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit', hour12: true })
}

export function WorkloadEventsChart({ data, timeRange = '6h', onTimeRangeChange, onRefresh }: WorkloadEventsChartProps) {
  const timestamps = data?.timestamps ?? []
  const hasData = timestamps.length > 0

  const option = useMemo(() => ({
    animation: true,
    animationDuration: 300,
    grid: { top: 20, right: 16, bottom: 30, left: 40 },
    tooltip: {
      trigger: 'axis' as const,
      backgroundColor: '#fff',
      borderColor: '#e5e7eb',
      borderWidth: 1,
      textStyle: { fontSize: 12, color: '#374151' },
    },
    xAxis: {
      type: 'category' as const,
      data: hasData ? timestamps.map((t) => formatLabel(t, timeRange)) : [],
      axisLine: { show: false },
      axisTick: { show: false },
      axisLabel: { fontSize: 10, color: '#9ca3af', interval: 'auto' },
    },
    yAxis: {
      type: 'value' as const,
      axisLine: { show: false },
      axisTick: { show: false },
      axisLabel: { fontSize: 10, color: '#9ca3af' },
      splitLine: { lineStyle: { color: '#f3f4f6' } },
    },
    series: [
      {
        name: 'Info',
        type: 'line',
        data: hasData ? data?.info ?? [] : [],
        smooth: true,
        showSymbol: false,
        lineStyle: { width: 2, color: '#4dabf7' },
        areaStyle: { color: 'rgba(77, 171, 247, 0.08)' },
      },
      {
        name: 'Warning',
        type: 'line',
        data: hasData ? data?.warning ?? [] : [],
        smooth: true,
        showSymbol: false,
        lineStyle: { width: 2, color: '#fcc419' },
        areaStyle: { color: 'rgba(252, 196, 25, 0.08)' },
      },
      {
        name: 'Error',
        type: 'line',
        data: hasData ? data?.error ?? [] : [],
        smooth: true,
        showSymbol: false,
        lineStyle: { width: 2, color: '#ff6b6b' },
        areaStyle: { color: 'rgba(255, 107, 107, 0.08)' },
      },
    ],
  }), [data, timestamps, hasData, timeRange])

  return (
    <div className="bg-white rounded-lg border border-border p-5 relative">
      <div className="flex items-center justify-between mb-2">
        <div>
          <h3 className="text-sm font-semibold text-gray-800">Workload Events</h3>
          <TimeRangeDropdown defaultValue={timeRange} onChange={onTimeRangeChange} />
        </div>
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-4 text-xs">
            <span className="flex items-center gap-1.5"><span className="w-2 h-2 rounded-full bg-[#4dabf7]" />Info</span>
            <span className="flex items-center gap-1.5"><span className="w-2 h-2 rounded-full bg-[#fcc419]" />Warning</span>
            <span className="flex items-center gap-1.5"><span className="w-2 h-2 rounded-full bg-[#ff6b6b]" />Error</span>
          </div>
          <button onClick={onRefresh} className="p-2 rounded-md bg-gray-800 hover:bg-gray-900 text-white transition-colors">
            <RotateCw size={14} />
          </button>
        </div>
      </div>
      {hasData ? (
        <ReactECharts option={option} style={{ height: 220 }} opts={{ renderer: 'canvas' }} />
      ) : (
        <div className="h-[220px] flex items-center justify-center">
          <span className="text-[13px] text-gray-300 font-semibold">No Data Available</span>
        </div>
      )}
    </div>
  )
}

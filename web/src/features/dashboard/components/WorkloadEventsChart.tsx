import { RotateCw } from 'lucide-react'
import Chart from 'react-apexcharts'
import type { ApexOptions } from 'apexcharts'
import { TimeRangeDropdown } from '@/shared/components/TimeRangeDropdown'

function formatTimeLabel(iso: string): string {
  const d = new Date(iso)
  return d.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit', hour12: true })
}

function generateTimeLabels(range: string): string[] {
  const now = new Date()
  switch (range) {
    case '1h':
      return Array.from({ length: 6 }, (_, i) => {
        const t = new Date(now.getTime() - (5 - i) * 10 * 60 * 1000)
        return t.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit', hour12: true })
      })
    case '6h':
      return Array.from({ length: 7 }, (_, i) => {
        const t = new Date(now.getTime() - (6 - i) * 60 * 60 * 1000)
        return t.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit', hour12: true })
      })
    case '24h':
      return Array.from({ length: 9 }, (_, i) => {
        const t = new Date(now.getTime() - (8 - i) * 3 * 60 * 60 * 1000)
        return t.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit', hour12: true })
      })
    case '7d':
      return Array.from({ length: 7 }, (_, i) => {
        const t = new Date(now.getTime() - (6 - i) * 24 * 60 * 60 * 1000)
        return t.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
      })
    case '30d':
      return Array.from({ length: 10 }, (_, i) => {
        const t = new Date(now.getTime() - (9 - i) * 3 * 24 * 60 * 60 * 1000)
        return t.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
      })
    default:
      return Array.from({ length: 7 }, (_, i) => {
        const t = new Date(now.getTime() - (6 - i) * 60 * 60 * 1000)
        return t.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit', hour12: true })
      })
  }
}

const timeLabels = Array.from({ length: 72 }, (_, i) => {
  const now = new Date()
  const offset = (71 - i) * 5 * 60 * 1000
  const time = new Date(now.getTime() - offset)
  return time.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit', hour12: true })
})

const options: ApexOptions = {
  chart: {
    type: 'line',
    toolbar: { show: false },
    zoom: { enabled: false },
    fontFamily: 'Inter, system-ui, sans-serif',
    animations: { enabled: true, easing: 'easeinout', speed: 300, dynamicAnimation: { enabled: true, speed: 200 } },
  },
  stroke: { width: 1.5, curve: 'straight' },
  dataLabels: { enabled: false },
  colors: ['#4dabf7', '#fcc419', '#ff6b6b'],
  legend: { show: false },
  xaxis: {
    categories: timeLabels,
    labels: {
      show: true,
      style: { fontSize: '10px', colors: '#868e96' },
      rotate: 0,
      hideOverlappingLabels: true,
      showDuplicates: false,
    },
    axisBorder: { show: false },
    axisTicks: { show: false },
    tickAmount: 6,
  },
  yaxis: {
    min: 0,
    max: 100,
    tickAmount: 5,
    labels: {
      style: { fontSize: '10px', colors: '#868e96' },
      formatter: (val: number) => Math.round(val).toString(),
    },
    forceNiceScale: true,
  },
  grid: {
    borderColor: '#f1f3f5',
    xaxis: { lines: { show: false } },
    yaxis: { lines: { show: true } },
  },
  tooltip: { enabled: true },
  fill: { type: 'solid', opacity: [0.1, 0.1, 0.1] },
}

interface WorkloadEventsChartProps {
  data?: { info: number[]; warning: number[]; error: number[]; timestamps?: string[] }
  timeRange?: string
  onTimeRangeChange?: (range: string) => void
  onRefresh?: () => void
}

export function WorkloadEventsChart({ data, timeRange, onTimeRangeChange, onRefresh }: WorkloadEventsChartProps) {
  const timestamps = data?.timestamps ?? []
  const hasData = timestamps.length > 0

  const labels = hasData
    ? timestamps.map((t) => formatTimeLabel(t))
    : generateTimeLabels(timeRange ?? '6h')

  const chartOptions: ApexOptions = { ...options, xaxis: { ...options.xaxis, categories: labels }, tooltip: { enabled: hasData } }

  const emptyFill = Array(labels.length).fill(0) as number[]
  const series = [
    { name: 'Info', data: hasData ? (data?.info ?? []) : emptyFill },
    { name: 'Warning', data: hasData ? (data?.warning ?? []) : emptyFill },
    { name: 'Error', data: hasData ? (data?.error ?? []) : emptyFill },
  ]

  return (
    <div className="bg-white rounded-lg border border-border p-5 relative">
      <div className="flex items-center justify-between mb-2">
        <div>
          <h3 className="text-sm font-semibold text-gray-800">Workload Events</h3>
          <TimeRangeDropdown defaultValue={timeRange ?? '6h'} onChange={onTimeRangeChange} />
        </div>
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-4 text-xs">
            <span className="flex items-center gap-1.5"><span className="w-2 h-2 rounded-full bg-[#4dabf7]" />Info</span>
            <span className="flex items-center gap-1.5"><span className="w-2 h-2 rounded-full bg-[#fcc419]" />Warning</span>
            <span className="flex items-center gap-1.5"><span className="w-2 h-2 rounded-full bg-[#ff6b6b]" />Error</span>
          </div>
          <button onClick={onRefresh} className="p-2 rounded-md bg-orange-500 hover:bg-orange-600 text-white transition-colors">
            <RotateCw size={14} />
          </button>
        </div>
      </div>
      <div className={hasData ? '' : 'opacity-30'}>
        <Chart options={chartOptions} series={series} type="area" height={220} />
      </div>
      {!hasData && (
        <div className="absolute left-0 right-0 bottom-0 top-[60px] flex items-center justify-center pointer-events-none">
          <span className="text-[13px] text-gray-300 font-semibold select-none">No Data Available</span>
        </div>
      )}
    </div>
  )
}

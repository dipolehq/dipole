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

const hourLabels = Array.from({ length: 24 }, (_, i) => {
  const hour = (new Date().getHours() - 23 + i + 24) % 24
  return `${hour.toString().padStart(2, '0')}:00`
})

const options: ApexOptions = {
  chart: {
    type: 'area',
    toolbar: { show: false },
    zoom: { enabled: false },
    fontFamily: 'Inter, system-ui, sans-serif',
    animations: { enabled: true, easing: 'easeinout', speed: 300, dynamicAnimation: { enabled: true, speed: 200 } },
  },
  stroke: { width: 2, curve: 'smooth' },
  dataLabels: { enabled: false },
  colors: ['#69db7c', '#ff6b6b'],
  fill: {
    type: 'gradient',
    gradient: { opacityFrom: 0.25, opacityTo: 0.05 },
  },
  legend: { show: false },
  xaxis: {
    categories: hourLabels,
    labels: {
      style: { fontSize: '10px', colors: '#868e96' },
      rotate: 0,
      hideOverlappingLabels: true,
    },
    axisBorder: { show: false },
    axisTicks: { show: false },
    tickAmount: 8,
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
    strokeDashArray: 3,
  },
  tooltip: { enabled: true },
}

interface AccessDecisionsChartProps {
  data?: { authorized: number[]; denied: number[]; timestamps?: string[] }
  timeRange?: string
  onTimeRangeChange?: (range: string) => void
  onRefresh?: () => void
}

export function AccessDecisionsChart({ data, timeRange, onTimeRangeChange, onRefresh }: AccessDecisionsChartProps) {
  const timestamps = data?.timestamps ?? []
  const hasData = timestamps.length > 0

  const labels = hasData
    ? timestamps.map((t) => formatTimeLabel(t))
    : generateTimeLabels(timeRange ?? '24h')

  const chartOptions: ApexOptions = { ...options, xaxis: { ...options.xaxis, categories: labels }, tooltip: { enabled: hasData } }

  const emptyFill = Array(labels.length).fill(0) as number[]
  const series = [
    { name: 'Authorized', data: hasData ? (data?.authorized ?? []) : emptyFill },
    { name: 'Denied', data: hasData ? (data?.denied ?? []) : emptyFill },
  ]

  return (
    <div className="bg-white rounded-lg border border-border p-5 relative">
      <div className="flex items-center justify-between mb-2">
        <div>
          <h3 className="text-sm font-semibold text-gray-800">
            Access Decisions / <span className="font-normal text-gray-500">Authorized vs Denied</span>
          </h3>
          <TimeRangeDropdown defaultValue={timeRange ?? '24h'} onChange={onTimeRangeChange} />
        </div>
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-4 text-xs">
            <span className="flex items-center gap-1.5"><span className="w-2 h-2 rounded-full bg-[#69db7c]" />Authorized</span>
            <span className="flex items-center gap-1.5"><span className="w-2 h-2 rounded-full bg-[#ff6b6b]" />Denied</span>
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

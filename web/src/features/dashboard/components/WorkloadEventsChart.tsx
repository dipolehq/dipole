import { RotateCw } from 'lucide-react'
import Chart from 'react-apexcharts'
import type { ApexOptions } from 'apexcharts'
import { TimeRangeDropdown } from '@/shared/components/TimeRangeDropdown'
import { dashboardData } from '@/features/dashboard/data'

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
    max: 60,
    tickAmount: 6,
    labels: { style: { fontSize: '10px', colors: '#868e96' } },
  },
  grid: {
    borderColor: '#f1f3f5',
    xaxis: { lines: { show: false } },
    yaxis: { lines: { show: true } },
  },
  tooltip: { enabled: true },
  fill: { type: 'solid', opacity: [0.1, 0.1, 0.1] },
}

const series = [
  { name: 'Info', data: dashboardData.workloadEvents.info },
  { name: 'Warning', data: dashboardData.workloadEvents.warning },
  { name: 'Error', data: dashboardData.workloadEvents.error },
]

export function WorkloadEventsChart() {
  return (
    <div className="bg-white rounded-lg border border-border p-5">
      <div className="flex items-center justify-between mb-2">
        <div>
          <h3 className="text-sm font-semibold text-gray-800">Workload Events</h3>
          <TimeRangeDropdown defaultValue="6h" />
        </div>
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-4 text-xs">
            <span className="flex items-center gap-1.5"><span className="w-2 h-2 rounded-full bg-[#4dabf7]" />Info</span>
            <span className="flex items-center gap-1.5"><span className="w-2 h-2 rounded-full bg-[#fcc419]" />Warning</span>
            <span className="flex items-center gap-1.5"><span className="w-2 h-2 rounded-full bg-[#ff6b6b]" />Error</span>
          </div>
          <button className="p-2 rounded-md bg-orange-500 hover:bg-orange-600 text-white transition-colors">
            <RotateCw size={14} />
          </button>
        </div>
      </div>
      <Chart options={options} series={series} type="area" height={220} />
    </div>
  )
}

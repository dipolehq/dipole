import { RotateCw } from 'lucide-react'
import Chart from 'react-apexcharts'
import type { ApexOptions } from 'apexcharts'
import { TimeRangeDropdown } from '@/shared/components/TimeRangeDropdown'
import { dashboardData } from '@/data/dashboard'

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
    labels: { style: { fontSize: '10px', colors: '#868e96' } },
  },
  grid: {
    borderColor: '#f1f3f5',
    strokeDashArray: 3,
  },
  tooltip: { enabled: true },
}

const series = [
  { name: 'Authorized', data: dashboardData.accessDecisions.authorized },
  { name: 'Denied', data: dashboardData.accessDecisions.denied },
]

export function AccessDecisionsChart() {
  return (
    <div className="bg-white rounded-lg border border-border p-5">
      <div className="flex items-center justify-between mb-2">
        <div>
          <h3 className="text-sm font-semibold text-gray-800">
            Access Decisions / <span className="font-normal text-gray-500">Authorized vs Denied</span>
          </h3>
          <TimeRangeDropdown defaultValue="24h" />
        </div>
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-4 text-xs">
            <span className="flex items-center gap-1.5"><span className="w-2 h-2 rounded-full bg-[#69db7c]" />Authorized</span>
            <span className="flex items-center gap-1.5"><span className="w-2 h-2 rounded-full bg-[#ff6b6b]" />Denied</span>
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

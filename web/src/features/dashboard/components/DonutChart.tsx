import Chart from 'react-apexcharts'
import type { ApexOptions } from 'apexcharts'
import { TimeRangeDropdown } from '@/shared/components/TimeRangeDropdown'

interface DonutChartProps {
  title: string
  subtitle: string
  total: number
  labels: string[]
  series: number[]
  colors: string[]
}

export function DonutChart({ title, subtitle, total, labels, series, colors }: DonutChartProps) {
  const options: ApexOptions = {
    chart: {
      type: 'donut',
      fontFamily: 'Inter, system-ui, sans-serif',
      selection: { enabled: false },
    },
    labels,
    colors,
    legend: {
      position: 'right',
      fontSize: '12px',
      markers: { size: 6, shape: 'circle' },
      offsetY: 0,
      itemMargin: { vertical: 4 },
      onItemHover: { highlightDataSeries: false },
      onItemClick: { toggleDataSeries: false },
    },
    dataLabels: { enabled: false },
    plotOptions: {
      pie: {
        expandOnClick: false,
        donut: {
          size: '70%',
          labels: {
            show: true,
            name: { show: false },
            value: {
              show: true,
              fontSize: '24px',
              fontWeight: '400',
              color: '#495057',
              offsetY: 8,
              formatter: () => total.toLocaleString(),
            },
            total: {
              show: true,
              showAlways: true,
              label: 'Total',
              fontSize: '11px',
              fontWeight: '400',
              color: '#868e96',
              formatter: () => total.toLocaleString(),
            },
          },
        },
      },
    },
    states: {
      hover: { filter: { type: 'none' } },
      active: { filter: { type: 'none' } },
    },
    stroke: { width: 2, colors: ['#fff'] },
    tooltip: {
      enabled: true,
      y: { formatter: (val: number) => String(val) },
    },
  }

  return (
    <div className="bg-white rounded-lg border border-border p-5 h-[340px]">
      <div className="mb-2">
        <h3 className="text-sm text-gray-600">
          {title} / <span className="font-bold text-gray-900">{subtitle}</span>
        </h3>
        <TimeRangeDropdown defaultValue="24h" />
      </div>
      <Chart options={options} series={series} type="donut" height={270} />
    </div>
  )
}

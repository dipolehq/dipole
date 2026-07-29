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
  const hasData = total > 0

  const options: ApexOptions = {
    chart: {
      type: 'donut',
      fontFamily: 'Inter, system-ui, sans-serif',
      selection: { enabled: false },
    },
    labels,
    colors: hasData ? colors : colors.map(() => '#e9ecef'),
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
              fontSize: hasData ? '24px' : '13px',
              fontWeight: hasData ? '400' : '600',
              color: hasData ? '#495057' : '#6b7280',
              offsetY: 8,
              formatter: () => hasData ? total.toLocaleString() : 'No Data Available',
            },
            total: {
              show: true,
              showAlways: true,
              label: 'Total',
              fontSize: '11px',
              fontWeight: '400',
              color: '#868e96',
              formatter: () => hasData ? total.toLocaleString() : 'No Data Available',
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
      enabled: hasData,
      y: { formatter: (val: number) => String(val) },
    },
  }

  const chartSeries = hasData ? series : labels.map(() => 1)

  return (
    <div className="bg-white rounded-lg border border-border p-5 h-[340px] relative">
      <div className="mb-2">
        <h3 className="text-sm text-gray-600">
          {title} / <span className="font-bold text-gray-900">{subtitle}</span>
        </h3>
        <TimeRangeDropdown defaultValue="24h" />
      </div>
      <div className={hasData ? '' : 'opacity-30'}>
        <Chart options={options} series={chartSeries} type="donut" height={270} />
      </div>
    </div>
  )
}

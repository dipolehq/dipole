import { useMemo } from 'react'
import ReactECharts from 'echarts-for-react'
import { TimeRangeDropdown } from '@/shared/components/TimeRangeDropdown'

const CHART_COLORS = ['#ffd8a8', '#a5d8ff', '#ffec99', '#b2f2bb', '#d0bfff', '#fcc2d7', '#99e9f2', '#e9ecef']

interface DonutChartProps {
  title: string
  subtitle: string
  data: { label: string; value: number }[]
  timeRange?: string
  onTimeRangeChange?: (range: string) => void
}

export function DonutChart({ title, subtitle, data, timeRange = '24h', onTimeRangeChange }: DonutChartProps) {
  const total = data.reduce((sum, d) => sum + d.value, 0)
  const hasData = total > 0

  const chartData = useMemo(() => {
    if (!hasData) return [{ name: 'No Data', value: 1 }]
    return data.map((d) => ({ name: d.label, value: d.value }))
  }, [data, hasData])

  const colors = useMemo(() => {
    if (!hasData) return ['#e5e7eb']
    return data.map((_, i) => CHART_COLORS[i % CHART_COLORS.length])
  }, [data, hasData])

  const option = useMemo(() => ({
    animation: true,
    animationDuration: 300,
    tooltip: hasData
      ? {
          trigger: 'item' as const,
          backgroundColor: '#fff',
          borderColor: '#e5e7eb',
          borderWidth: 1,
          textStyle: { fontSize: 12, color: '#374151' },
          formatter: '{b}: {c}',
        }
      : { show: false },
    legend: {
      orient: 'vertical' as const,
      right: 10,
      top: 'center',
      itemWidth: 8,
      itemHeight: 8,
      itemGap: 10,
      textStyle: { fontSize: 11, color: '#6b7280' },
      selectedMode: hasData,
    },
    color: colors,
    series: [
      {
        type: 'pie',
        radius: ['55%', '78%'],
        center: ['35%', '50%'],
        avoidLabelOverlap: false,
        silent: !hasData,
        label: {
          show: true,
          position: 'center' as const,
          fontSize: hasData ? 22 : 12,
          fontWeight: hasData ? '400' : '600',
          color: hasData ? '#374151' : '#9ca3af',
          formatter: () => hasData ? String(total) : 'No Data',
        },
        emphasis: {
          disabled: !hasData,
          scale: false,
          label: { show: true, fontSize: 22, fontWeight: '400' },
        },
        labelLine: { show: false },
        data: chartData,
        itemStyle: {
          borderColor: '#fff',
          borderWidth: 2,
        },
      },
    ],
  }), [chartData, colors, hasData, total])

  return (
    <div className="bg-white rounded-lg border border-border p-5 h-[340px]">
      <div className="mb-2">
        <h3 className="text-sm text-gray-600">
          {title} / <span className="font-bold text-gray-900">{subtitle}</span>
        </h3>
        <TimeRangeDropdown defaultValue={timeRange} onChange={onTimeRangeChange} />
      </div>
      <ReactECharts option={option} style={{ height: 270 }} opts={{ renderer: 'canvas' }} />
    </div>
  )
}

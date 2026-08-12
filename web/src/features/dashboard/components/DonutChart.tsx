import { useMemo } from 'react'
import ReactECharts from 'echarts-for-react'
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

  const chartData = useMemo(() => {
    if (!hasData) return labels.map((name) => ({ name, value: 1 }))
    return labels.map((name, i) => ({ name, value: series[i] ?? 0 }))
  }, [labels, series, hasData])

  const option = useMemo(() => ({
    animation: true,
    animationDuration: 300,
    tooltip: {
      trigger: 'item' as const,
      backgroundColor: '#fff',
      borderColor: '#e5e7eb',
      borderWidth: 1,
      textStyle: { fontSize: 12, color: '#374151' },
      formatter: hasData ? '{b}: {c}' : undefined,
    },
    legend: {
      orient: 'vertical' as const,
      right: 10,
      top: 'center',
      itemWidth: 8,
      itemHeight: 8,
      itemGap: 10,
      textStyle: { fontSize: 11, color: '#6b7280' },
    },
    color: hasData ? colors : colors.map(() => '#e5e7eb'),
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
        <TimeRangeDropdown defaultValue="24h" />
      </div>
      <ReactECharts option={option} style={{ height: 270 }} opts={{ renderer: 'canvas' }} />
    </div>
  )
}

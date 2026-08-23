import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { StatCard } from '@/features/dashboard/components/StatCard'
import { WorkloadEventsChart } from '@/features/dashboard/components/WorkloadEventsChart'
import { AccessDecisionsChart } from '@/features/dashboard/components/AccessDecisionsChart'
import { ConnectionsChart } from '@/features/dashboard/components/ConnectionsChart'
import { DonutChart } from '@/features/dashboard/components/DonutChart'
import {
  STAT_CARDS,
} from '@/features/dashboard/config'
import { fetchDashboardStats } from '@/features/dashboard/api/dashboardApi'
import { dashboardData as fallbackData } from '@/features/dashboard/data'

export function DashboardPage() {
  const [workloadTimeRange, setWorkloadTimeRange] = useState('6h')
  const [decisionsTimeRange, setDecisionsTimeRange] = useState('24h')
  const [summaryTimeRange, setSummaryTimeRange] = useState('24h')

  const { data, refetch, isError: isStatsError } = useQuery({
    queryKey: ['dashboard', 'stats', workloadTimeRange],
    queryFn: () => fetchDashboardStats(workloadTimeRange),
    staleTime: 30_000,
    placeholderData: (prev) => prev,
  })

  const { data: decisionsData, refetch: refetchDecisions } = useQuery({
    queryKey: ['dashboard', 'decisions', decisionsTimeRange],
    queryFn: () => fetchDashboardStats(decisionsTimeRange),
    staleTime: 30_000,
    placeholderData: (prev) => prev,
  })

  const { data: summaryData } = useQuery({
    queryKey: ['dashboard', 'summary', summaryTimeRange],
    queryFn: () => fetchDashboardStats(summaryTimeRange),
    staleTime: 30_000,
    placeholderData: (prev) => prev,
  })

  const dashboardData = data ?? fallbackData
  const decisionsSource = decisionsData ?? fallbackData
  const summarySource = summaryData ?? fallbackData
  const { stats } = dashboardData

  return (
    <div className="flex flex-col h-full">
      <div className="px-4 py-3 space-y-4 flex-1 overflow-y-auto">

      {isStatsError && (
        <div className="px-4 py-2 bg-red-50 border border-red-200 rounded-md text-sm text-red-700">
          Failed to load dashboard data. Please try refreshing.
        </div>
      )}

      <div className="grid grid-cols-5 gap-4">
        {STAT_CARDS.map((card) => {
          const cardData = stats[card.key as keyof typeof stats]
          return (
            <StatCard
              key={card.key}
              title={card.title}
              count={cardData.count}
              icon={card.icon}
              color={card.color}
              lines={[`Active: ${cardData.active}`, `Inactive: ${cardData.inactive}`]}
            />
          )
        })}
      </div>

      <WorkloadEventsChart data={dashboardData.workloadEvents} onTimeRangeChange={setWorkloadTimeRange} timeRange={workloadTimeRange} onRefresh={() => refetch()} />
      <AccessDecisionsChart data={decisionsSource.accessDecisions} onTimeRangeChange={setDecisionsTimeRange} timeRange={decisionsTimeRange} onRefresh={() => refetchDecisions()} />

      <div className="grid grid-cols-2 gap-4">
        <ConnectionsChart title="Client Workloads" subtitle="Most Connections" data={summarySource.clientConnections} timeRange={summaryTimeRange} onTimeRangeChange={setSummaryTimeRange} />
        <ConnectionsChart title="Server Workloads" subtitle="Most Connections" data={summarySource.serverConnections} barColor="bg-green-100" timeRange={summaryTimeRange} onTimeRangeChange={setSummaryTimeRange} />
      </div>

      <div className="grid grid-cols-2 gap-4">
        <DonutChart
          title="Credentials"
          subtitle="Usage by Type"
          data={summarySource.credentialUsage}
          timeRange={summaryTimeRange}
          onTimeRangeChange={setSummaryTimeRange}
        />
        <DonutChart
          title="Workload Connections"
          subtitle="Application Protocol"
          data={summarySource.protocolUsage}
          timeRange={summaryTimeRange}
          onTimeRangeChange={setSummaryTimeRange}
        />
      </div>

      <div className="grid grid-cols-2 gap-4">
        <DonutChart
          title="Denials"
          subtitle="By Reason"
          data={summarySource.denialReasons}
          timeRange={summaryTimeRange}
          onTimeRangeChange={setSummaryTimeRange}
        />
        <DonutChart
          title="Trust Providers"
          subtitle="Attestation Success Rate"
          data={summarySource.attestationResults}
          timeRange={summaryTimeRange}
          onTimeRangeChange={setSummaryTimeRange}
        />
      </div>
    </div>
    </div>
  )
}

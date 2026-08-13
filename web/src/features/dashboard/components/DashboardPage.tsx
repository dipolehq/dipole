import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { StatCard } from '@/features/dashboard/components/StatCard'
import { WorkloadEventsChart } from '@/features/dashboard/components/WorkloadEventsChart'
import { AccessDecisionsChart } from '@/features/dashboard/components/AccessDecisionsChart'
import { ConnectionsChart } from '@/features/dashboard/components/ConnectionsChart'
import { DonutChart } from '@/features/dashboard/components/DonutChart'
import { PageHeader } from '@/shared/components/PageHeader'
import {
  STAT_CARDS,
  CREDENTIAL_USAGE_CONFIG,
  PROTOCOL_USAGE_CONFIG,
  DENIAL_REASONS_CONFIG,
  ATTESTATION_RESULTS_CONFIG,
} from '@/features/dashboard/config'
import { fetchDashboardStats } from '@/features/dashboard/api/dashboardApi'
import { dashboardData as fallbackData } from '@/features/dashboard/data'

export function DashboardPage() {
  const [workloadTimeRange, setWorkloadTimeRange] = useState('6h')
  const [decisionsTimeRange, setDecisionsTimeRange] = useState('24h')

  const { data, refetch } = useQuery({
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

  const dashboardData = data ?? fallbackData
  const decisionsSource = decisionsData ?? fallbackData
  const { stats } = dashboardData

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-4 shrink-0 h-[44px] border-b border-border bg-white">
        <PageHeader />
      </div>
      <div className="px-4 py-3 space-y-4 flex-1 overflow-y-auto">

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
        <ConnectionsChart title="Client Workloads" subtitle="Most Connections" data={dashboardData.clientConnections} />
        <ConnectionsChart title="Server Workloads" subtitle="Most Connections" data={dashboardData.serverConnections} barColor="bg-green-100" />
      </div>

      <div className="grid grid-cols-2 gap-4">
        <DonutChart
          title="Credentials"
          subtitle="Usage by Type"
          total={dashboardData.credentialUsage.reduce((a, b) => a + b, 0)}
          labels={CREDENTIAL_USAGE_CONFIG.labels}
          series={dashboardData.credentialUsage}
          colors={CREDENTIAL_USAGE_CONFIG.colors}
        />
        <DonutChart
          title="Workload Connections"
          subtitle="Application Protocol"
          total={dashboardData.protocolUsage.reduce((a, b) => a + b, 0)}
          labels={PROTOCOL_USAGE_CONFIG.labels}
          series={dashboardData.protocolUsage}
          colors={PROTOCOL_USAGE_CONFIG.colors}
        />
      </div>

      <div className="grid grid-cols-2 gap-4">
        <DonutChart
          title="Denials"
          subtitle="By Reason"
          total={dashboardData.denialReasons.reduce((a, b) => a + b, 0)}
          labels={DENIAL_REASONS_CONFIG.labels}
          series={dashboardData.denialReasons}
          colors={DENIAL_REASONS_CONFIG.colors}
        />
        <DonutChart
          title="Trust Providers"
          subtitle="Attestation Success Rate"
          total={dashboardData.attestationResults.reduce((a, b) => a + b, 0)}
          labels={ATTESTATION_RESULTS_CONFIG.labels}
          series={dashboardData.attestationResults}
          colors={ATTESTATION_RESULTS_CONFIG.colors}
        />
      </div>
    </div>
    </div>
  )
}

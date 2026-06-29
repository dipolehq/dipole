import { Link } from 'react-router-dom'
import { StatCard } from '@/features/dashboard/components/StatCard'
import { WorkloadEventsChart } from '@/features/dashboard/components/WorkloadEventsChart'
import { AccessDecisionsChart } from '@/features/dashboard/components/AccessDecisionsChart'
import { ConnectionsChart } from '@/features/dashboard/components/ConnectionsChart'
import { DonutChart } from '@/features/dashboard/components/DonutChart'
import { ResourceSetDropdown } from '@/shared/components/ResourceSetDropdown'
import {
  STAT_CARDS,
  CREDENTIAL_USAGE_CONFIG,
  PROTOCOL_USAGE_CONFIG,
  DENIAL_REASONS_CONFIG,
  ATTESTATION_RESULTS_CONFIG,
} from '@/features/dashboard/config'
import { dashboardData } from '@/data/dashboard'

export function DashboardPage() {
  const { stats } = dashboardData

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-gray-900">Dashboard</h1>
        <div className="flex items-center gap-3">
          <Link
            to="/help"
            className="px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white text-sm font-medium rounded-md transition-colors"
          >
            Getting Started
          </Link>
          <ResourceSetDropdown />
        </div>
      </div>

      <div className="grid grid-cols-5 gap-4">
        {STAT_CARDS.map((card) => {
          const data = stats[card.key as keyof typeof stats]
          return (
            <StatCard
              key={card.key}
              title={card.title}
              count={data.count}
              icon={card.icon}
              color={card.color}
              lines={[`Active: ${data.active}`, `Inactive: ${data.inactive}`]}
            />
          )
        })}
      </div>

      <WorkloadEventsChart />
      <AccessDecisionsChart />

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
  )
}

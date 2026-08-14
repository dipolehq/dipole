import { useMemo } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { type ColumnDef } from '@tanstack/react-table'
import { Cpu, RefreshCw } from 'lucide-react'
import { PageHeader } from '@/shared/components/PageHeader'
import { DataTable } from '@/shared/components/ui/DataTable'
import { apiClient } from '@/shared/lib/api-client'

type McpPolicyStatus = 'Active' | 'Inactive' | 'Unknown'

interface McpTracingRow {
  id: string
  timestamp: string
  clientIP: string
  accessPolicyName: string
  accessPolicyStatus: McpPolicyStatus
  redirectUri: string
  clientWorkloadName: string
  clientWorkloadStatus: 'Identified' | 'Unknown'
  resourceServer: string
  serverWorkloadName: string
  serverWorkloadStatus: 'Identified' | 'Unknown'
  severity: string
}

interface McpAuthorizationTracingEntry {
  id: string
  timestamp: string
  clientIP: string
  accessPolicy: { name: string; status: McpPolicyStatus }
  redirectUri: string
  clientWorkload: { name: string; status: 'Identified' | 'Unknown' }
  resourceServer: string
  serverWorkload: { name: string; status: 'Identified' | 'Unknown' }
  severity: 'Info' | 'Warning' | 'Error'
}

async function fetchTracingEntries(): Promise<McpAuthorizationTracingEntry[]> {
  const res = await apiClient.get<{ entries: McpAuthorizationTracingEntry[] }>('/events/mcp-tracing')
  return res.entries ?? []
}

// Sample data for UI development when no live MCP traffic exists
const sampleEntries: McpAuthorizationTracingEntry[] = [
  { id: 's1', timestamp: new Date(Date.now() - 120000).toISOString(), clientIP: '49.37.169.218,172.71.124.8', accessPolicy: { name: 'Claude to MCP Gateway', status: 'Active' }, redirectUri: 'https://claude.ai/api/mcp/auth_callback', clientWorkload: { name: 'Claude Desktop', status: 'Identified' }, resourceServer: 'testing-dipole-mcpgateway.dipolehq.com', serverWorkload: { name: 'Dipole MCP Gateway', status: 'Identified' }, severity: 'Info' },
  { id: 's2', timestamp: new Date(Date.now() - 180000).toISOString(), clientIP: '49.37.169.218,162.158.170.8', accessPolicy: { name: 'Claude to MCP Gateway', status: 'Active' }, redirectUri: 'https://claude.ai/api/mcp/auth_callback', clientWorkload: { name: 'Claude Desktop', status: 'Identified' }, resourceServer: 'testing-dipole-mcpgateway.dipolehq.com', serverWorkload: { name: 'Dipole MCP Gateway', status: 'Identified' }, severity: 'Info' },
  { id: 's3', timestamp: new Date(Date.now() - 300000).toISOString(), clientIP: '103.21.58.44,104.23.21.5', accessPolicy: { name: 'Gemini to MCP Gateway', status: 'Active' }, redirectUri: 'http://localhost:58299/oauth/callback', clientWorkload: { name: 'Gemini CLI', status: 'Identified' }, resourceServer: 'testing-dipole-mcpgateway.dipolehq.com', serverWorkload: { name: 'Dipole MCP Gateway', status: 'Identified' }, severity: 'Info' },
  { id: 's4', timestamp: new Date(Date.now() - 420000).toISOString(), clientIP: '14.97.201.133,172.70.208.5', accessPolicy: { name: 'Authenticated', status: 'Unknown' }, redirectUri: 'amanbhardwaj0081@gmail.com', clientWorkload: { name: 'Unknown', status: 'Unknown' }, resourceServer: 'Unknown', serverWorkload: { name: 'Unknown', status: 'Unknown' }, severity: 'Error' },
  { id: 's5', timestamp: new Date(Date.now() - 540000).toISOString(), clientIP: '49.37.169.218,104.22.66.26', accessPolicy: { name: 'Claude to MCP Gateway', status: 'Active' }, redirectUri: 'https://claude.ai/api/mcp/auth_callback', clientWorkload: { name: 'Claude Desktop', status: 'Identified' }, resourceServer: 'testing-dipole-mcpgateway.dipolehq.com', serverWorkload: { name: 'Dipole MCP Gateway', status: 'Identified' }, severity: 'Info' },
  { id: 's6', timestamp: new Date(Date.now() - 660000).toISOString(), clientIP: '106.51.72.89,162.158.170.8', accessPolicy: { name: 'Gemini to MCP Gateway', status: 'Active' }, redirectUri: 'http://localhost:56686/oauth/callback', clientWorkload: { name: 'Gemini CLI', status: 'Identified' }, resourceServer: 'testing-dipole-mcpgateway.dipolehq.com', serverWorkload: { name: 'Dipole MCP Gateway', status: 'Identified' }, severity: 'Info' },
  { id: 's7', timestamp: new Date(Date.now() - 900000).toISOString(), clientIP: '49.37.169.218,104.22.66.26', accessPolicy: { name: 'Missing token', status: 'Unknown' }, redirectUri: '', clientWorkload: { name: 'Unknown', status: 'Unknown' }, resourceServer: 'Unknown', serverWorkload: { name: 'Unknown', status: 'Unknown' }, severity: 'Error' },
  { id: 's8', timestamp: new Date(Date.now() - 1020000).toISOString(), clientIP: '223.226.151.4,172.71.124.8', accessPolicy: { name: 'Claude to MCP Gateway', status: 'Active' }, redirectUri: 'https://claude.ai/api/mcp/auth_callback', clientWorkload: { name: 'Claude Desktop', status: 'Identified' }, resourceServer: 'testing-dipole-mcpgateway.dipolehq.com', serverWorkload: { name: 'Dipole MCP Gateway', status: 'Identified' }, severity: 'Info' },
  { id: 's9', timestamp: new Date(Date.now() - 1200000).toISOString(), clientIP: '49.37.169.218,162.158.170.8', accessPolicy: { name: 'Gemini to MCP Gateway', status: 'Active' }, redirectUri: 'http://localhost:58432/oauth/callback', clientWorkload: { name: 'Gemini CLI', status: 'Identified' }, resourceServer: 'testing-dipole-mcpgateway.dipolehq.com', serverWorkload: { name: 'Dipole MCP Gateway', status: 'Identified' }, severity: 'Info' },
  { id: 's10', timestamp: new Date(Date.now() - 1500000).toISOString(), clientIP: '103.21.58.44,104.23.21.5', accessPolicy: { name: 'Authenticated', status: 'Active' }, redirectUri: 'https://claude.ai/api/mcp/auth_callback', clientWorkload: { name: 'Claude Desktop', status: 'Identified' }, resourceServer: 'testing-dipole-mcpgateway.dipolehq.com', serverWorkload: { name: 'Dipole MCP Gateway', status: 'Identified' }, severity: 'Warning' },
]

function formatTimestamp(iso: string): string {
  const d = new Date(iso)
  return d.toLocaleString('en-IN', { month: '2-digit', day: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: true })
}

function SeverityBadge({ severity }: { severity: string }) {
  const colors: Record<string, string> = { Info: 'text-green-600', Warning: 'text-gray-600', Error: 'text-red-500' }
  return <span className={`text-xs font-medium ${colors[severity] ?? 'text-gray-600'}`}>{severity}</span>
}

function PolicyCell({ name, status }: { name: string; status: McpPolicyStatus }) {
  if (status === 'Unknown') return <span className="text-xs text-gray-600 font-medium">Unknown</span>
  return <div className="truncate text-xs text-gray-700 font-medium">{name}</div>
}

function WorkloadCell({ name, status, uri }: { name: string; status: 'Identified' | 'Unknown'; uri: string }) {
  return (
    <div className="min-w-0">
      {status === 'Identified' ? (
        <div className="text-xs text-gray-700 font-medium truncate">{name}</div>
      ) : (
        <div className="text-xs text-gray-600 font-medium">Unknown</div>
      )}
      <div className="text-[11px] text-gray-400 truncate" title={uri}>{uri}</div>
    </div>
  )
}

export function ReportingMcpTracingPage() {
  const queryClient = useQueryClient()

  const { data: rawEntries = [] } = useQuery({
    queryKey: ['mcp-tracing'],
    queryFn: fetchTracingEntries,
    refetchInterval: 3000,
  })

  const entries = rawEntries.length > 0 ? rawEntries : (import.meta.env.DEV ? sampleEntries : [])

  const rows: McpTracingRow[] = useMemo(() => entries.map((e) => ({
    id: e.id,
    timestamp: e.timestamp,
    clientIP: e.clientIP,
    accessPolicyName: e.accessPolicy.name,
    accessPolicyStatus: e.accessPolicy.status,
    redirectUri: e.redirectUri,
    clientWorkloadName: e.clientWorkload.name,
    clientWorkloadStatus: e.clientWorkload.status,
    resourceServer: e.resourceServer,
    serverWorkloadName: e.serverWorkload.name,
    serverWorkloadStatus: e.serverWorkload.status,
    severity: e.severity,
  })), [entries])

  const columns: ColumnDef<McpTracingRow, unknown>[] = useMemo(() => [
    {
      id: '_action',
      header: () => (
        <div className="flex items-center gap-1.5">
          <button
            type="button"
            onClick={() => queryClient.invalidateQueries({ queryKey: ['mcp-tracing'] })}
            className="text-gray-400 hover:text-gray-600 transition-colors"
            title="Refresh"
          >
            <RefreshCw size={15} strokeWidth={2.5} />
          </button>
          <span className="w-2 h-2 rounded-full bg-green-500" title="Live (polling every 3s)" />
        </div>
      ),
      cell: () => <Cpu size={14} className="text-gray-400" />,
      size: 50,
      enableSorting: false,
      enableHiding: false,
    },
    {
      accessorKey: 'timestamp',
      header: 'Timestamp',
      cell: ({ row }) => <span className="text-gray-700 whitespace-nowrap">{formatTimestamp(row.getValue('timestamp'))}</span>,
      size: 180,
    },
    {
      accessorKey: 'clientIP',
      header: 'Client IP',
      cell: ({ row }) => <span className="text-gray-700 truncate block">{(row.getValue('clientIP') as string).split(',')[0]}</span>,
      size: 130,
    },
    {
      accessorKey: 'accessPolicyName',
      header: 'Access Policy',
      cell: ({ row }) => <PolicyCell name={row.original.accessPolicyName} status={row.original.accessPolicyStatus} />,
    },
    {
      id: 'clientWorkload',
      header: 'Redirect URI / Client WL',
      cell: ({ row }) => <WorkloadCell name={row.original.clientWorkloadName} status={row.original.clientWorkloadStatus} uri={row.original.redirectUri} />,
      enableSorting: false,
    },
    {
      id: 'serverWorkload',
      header: 'Resource Server / Server WL',
      cell: ({ row }) => <WorkloadCell name={row.original.serverWorkloadName} status={row.original.serverWorkloadStatus} uri={row.original.resourceServer} />,
      enableSorting: false,
    },
    {
      accessorKey: 'severity',
      header: 'Severity',
      cell: ({ row }) => <SeverityBadge severity={row.getValue('severity')} />,
      size: 90,
    },
  ], [queryClient])

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-4 shrink-0 h-[44px] border-b border-border bg-white">
        <PageHeader />
      </div>
      <div className="flex-1 min-h-0">
        <DataTable
          columns={columns}
          data={rows}
          searchPlaceholder="Filter by IP, policy, workload, or URI..."
          searchKeys={['clientIP', 'accessPolicyName', 'clientWorkloadName', 'serverWorkloadName', 'redirectUri', 'resourceServer']}
          facetedFilters={[{ columnId: 'severity', label: 'Severity' }]}
          emptyMessage="Listening for MCP authorization requests..."
        />
      </div>
    </div>
  )
}

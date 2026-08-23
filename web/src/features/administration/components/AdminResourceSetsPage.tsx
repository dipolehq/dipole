import { useMemo } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { type ColumnDef } from '@tanstack/react-table'
import { ArrowLeftRight, MonitorSmartphone, CheckCircle, Clock, Key, Server } from 'lucide-react'
import { DataTable } from '@/shared/components/ui/DataTable'
import { useResourceSets } from '@/features/administration/hooks/useResourceSets'

interface ResourceSetRow {
  id: string
  name: string
  description: string
  resourceSetId: string
  accessPolicies: number
  clientWorkloads: number
  trustProviders: number
  accessConditions: number
  credentialProviders: number
  serverWorkloads: number
  modifiedAt: string
}

function formatDate(iso: string): string {
  if (!iso) return '—'
  return new Date(iso).toLocaleDateString('en-GB', { day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

function ResourceIcons({ row }: { row: ResourceSetRow }) {
  const items = [
    { icon: ArrowLeftRight, count: row.accessPolicies },
    { icon: MonitorSmartphone, count: row.clientWorkloads },
    { icon: CheckCircle, count: row.trustProviders },
    { icon: Clock, count: row.accessConditions },
    { icon: Key, count: row.credentialProviders },
    { icon: Server, count: row.serverWorkloads },
  ]
  return (
    <div className="flex items-center gap-5">
      {items.map((item, i) => (
        <div key={i} className="flex items-center gap-1 text-gray-500">
          <item.icon size={14} strokeWidth={2} />
          <span className="text-[11px]">{item.count}</span>
        </div>
      ))}
    </div>
  )
}

export function AdminResourceSetsPage() {
  const navigate = useNavigate()

  const { data, isLoading } = useResourceSets(1, 500, '')
  const items = data?.items ?? []

  const tableData: ResourceSetRow[] = useMemo(
    () =>
      items.map((item) => ({
        id: item.id,
        name: item.name ?? '',
        description: item.description ?? '',
        resourceSetId: item.id ?? '',
        accessPolicies: item.accessPolicies ?? 0,
        clientWorkloads: item.clientWorkloads ?? 0,
        trustProviders: item.trustProviders ?? 0,
        accessConditions: item.accessConditions ?? 0,
        credentialProviders: item.credentialProviders ?? 0,
        serverWorkloads: item.serverWorkloads ?? 0,
        modifiedAt: item.updatedAt ?? item.createdAt ?? '',
      })),
    [items],
  )

  const columns: ColumnDef<ResourceSetRow, unknown>[] = useMemo(
    () => [
      {
        accessorKey: 'name',
        header: 'Name',
        cell: ({ row }) => (
          <Link
            to={`/administration/resource_sets/${row.original.id}`}
            className="font-medium text-gray-900 hover:text-blue-700 transition-colors truncate block"
            onClick={(e) => e.stopPropagation()}
          >
            {row.getValue('name')}
          </Link>
        ),
        size: 140,
        enableHiding: false,
      },
      {
        accessorKey: 'description',
        header: 'Description',
        cell: ({ row }) => <span className="text-gray-700 truncate block">{row.getValue('description')}</span>,
        size: 200,
      },
      {
        accessorKey: 'resourceSetId',
        header: 'ID',
        cell: ({ row }) => <span className="text-gray-500 font-mono text-[11px] truncate block">{row.getValue('resourceSetId')}</span>,
        size: 240,
      },
      {
        id: 'resources',
        header: 'Resources',
        cell: ({ row }) => <ResourceIcons row={row.original} />,
        size: 280,
        enableSorting: false,
      },
      {
        accessorKey: 'modifiedAt',
        header: 'Modified At',
        cell: ({ row }) => <span className="text-gray-700 whitespace-nowrap">{formatDate(row.getValue('modifiedAt'))}</span>,
        size: 160,
      },
    ],
    [],
  )

  return (
    <div className="flex flex-col h-full">
      <DataTable
        columns={columns}
        data={tableData}
        isLoading={isLoading}
        searchPlaceholder="Filter by name..."
        searchKeys={['name', 'description']}
        actionLabel="New Resource Set"
        onAction={() => navigate('/administration/resource_sets/new')}
        onRowClick={(row) => navigate(`/administration/resource_sets/${row.id}`)}
        emptyMessage="No resource sets found."
      />
    </div>
  )
}

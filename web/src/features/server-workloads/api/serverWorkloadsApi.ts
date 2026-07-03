import { serverWorkloadListData } from '@/data/server-workloads'
import { serverWorkloadsData } from '@/data/policies/serverWorkloads'
import { buildServerWorkloadValues } from '@/data/policies/policyFormMappers'
import type { ServerWorkloadListItem } from '@/data/server-workloads'
import type { ServerWorkloadFormValues } from '@/shared/components/forms/types'

const NETWORK_DELAY_MS = 80

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

export interface FetchServerWorkloadsResult {
  items: ServerWorkloadListItem[]
  total: number
}

export async function fetchServerWorkloads(): Promise<FetchServerWorkloadsResult> {
  await sleep(NETWORK_DELAY_MS)
  return {
    items: serverWorkloadListData,
    total: serverWorkloadListData.length,
  }
}

export interface ServerWorkloadMetadata {
  entityId: string
  createdAt: string
  createdBy: string
  lastModifiedAt: string
  lastModifiedBy: string
  isActive: boolean
}

export interface FetchServerWorkloadDetailResult {
  values: ServerWorkloadFormValues
  metadata: ServerWorkloadMetadata
}

export async function fetchServerWorkload(id: string): Promise<FetchServerWorkloadDetailResult | null> {
  await sleep(NETWORK_DELAY_MS)
  const sw = serverWorkloadsData.find((w) => w.id === id)
  if (!sw) return null

  return {
    values: buildServerWorkloadValues(id),
    metadata: {
      entityId: sw.id,
      createdAt: sw.createdAt,
      createdBy: sw.createdBy,
      lastModifiedAt: sw.createdAt,
      lastModifiedBy: sw.createdBy,
      isActive: sw.isActive,
    },
  }
}

export async function saveServerWorkload(_id: string, values: ServerWorkloadFormValues): Promise<ServerWorkloadFormValues> {
  await sleep(NETWORK_DELAY_MS)
  return values
}

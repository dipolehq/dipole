import { clientWorkloadListData } from '@/data/client-workloads'
import { clientWorkloadsData } from '@/data/policies/clientWorkloads'
import { buildClientWorkloadValues } from '@/data/policies/policyFormMappers'
import type { ClientWorkloadListItem } from '@/data/client-workloads'
import type { ClientWorkloadFormValues } from '@/shared/components/forms/types'

const NETWORK_DELAY_MS = 80

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

export interface FetchClientWorkloadsResult {
  items: ClientWorkloadListItem[]
  total: number
}

export async function fetchClientWorkloads(): Promise<FetchClientWorkloadsResult> {
  await sleep(NETWORK_DELAY_MS)
  return {
    items: clientWorkloadListData,
    total: clientWorkloadListData.length,
  }
}

export interface ClientWorkloadMetadata {
  entityId: string
  createdAt: string
  createdBy: string
  lastModifiedAt: string
  lastModifiedBy: string
  isActive: boolean
}

export interface FetchClientWorkloadDetailResult {
  values: ClientWorkloadFormValues
  metadata: ClientWorkloadMetadata
}

export async function fetchClientWorkload(id: string): Promise<FetchClientWorkloadDetailResult | null> {
  await sleep(NETWORK_DELAY_MS)
  const cw = clientWorkloadsData.find((w) => w.id === id)
  if (!cw) return null

  return {
    values: buildClientWorkloadValues(id),
    metadata: {
      entityId: cw.id,
      createdAt: cw.createdAt,
      createdBy: cw.createdBy,
      lastModifiedAt: cw.createdAt,
      lastModifiedBy: cw.createdBy,
      isActive: cw.isActive,
    },
  }
}

export async function saveClientWorkload(_id: string, values: ClientWorkloadFormValues): Promise<ClientWorkloadFormValues> {
  await sleep(NETWORK_DELAY_MS)
  return values
}

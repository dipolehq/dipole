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
  createdAt: string
  createdBy: string
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
      createdAt: cw.createdAt,
      createdBy: cw.createdBy,
    },
  }
}

export async function saveClientWorkload(_id: string, values: ClientWorkloadFormValues): Promise<ClientWorkloadFormValues> {
  await sleep(NETWORK_DELAY_MS)
  return values
}

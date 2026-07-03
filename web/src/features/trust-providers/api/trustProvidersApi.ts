import { trustProviderListData } from '@/data/trust-providers'
import { trustProvidersData } from '@/data/policies/trustProviders'
import { buildTrustProviderValues } from '@/data/policies/policyFormMappers'
import type { TrustProviderListItem } from '@/data/trust-providers'
import type { TrustProviderFormValues } from '@/shared/components/forms/types'

const NETWORK_DELAY_MS = 80

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

export interface FetchTrustProvidersResult {
  items: TrustProviderListItem[]
  total: number
}

export async function fetchTrustProviders(): Promise<FetchTrustProvidersResult> {
  await sleep(NETWORK_DELAY_MS)
  return {
    items: trustProviderListData,
    total: trustProviderListData.length,
  }
}

export interface TrustProviderMetadata {
  entityId: string
  createdAt: string
  createdBy: string
  lastModifiedAt: string
  lastModifiedBy: string
  isActive: boolean
}

export interface FetchTrustProviderDetailResult {
  values: TrustProviderFormValues
  metadata: TrustProviderMetadata
}

export async function fetchTrustProvider(id: string): Promise<FetchTrustProviderDetailResult | null> {
  await sleep(NETWORK_DELAY_MS)
  const tp = trustProvidersData.find((t) => t.id === id)
  if (!tp) return null

  return {
    values: buildTrustProviderValues(id),
    metadata: {
      entityId: tp.id,
      createdAt: tp.createdAt,
      createdBy: tp.createdBy,
      lastModifiedAt: tp.createdAt,
      lastModifiedBy: tp.createdBy,
      isActive: tp.isActive,
    },
  }
}

export async function saveTrustProvider(_id: string, values: TrustProviderFormValues): Promise<TrustProviderFormValues> {
  await sleep(NETWORK_DELAY_MS)
  return values
}

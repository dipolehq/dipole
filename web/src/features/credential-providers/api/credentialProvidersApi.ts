import { credentialProviderListData } from '@/data/credential-providers'
import { credentialProvidersData } from '@/data/policies/credentialProviders'
import { buildCredentialProviderValues } from '@/data/policies/policyFormMappers'
import type { CredentialProviderListItem } from '@/data/credential-providers'
import type { CredentialProviderFormValues } from '@/shared/components/forms/types'

const NETWORK_DELAY_MS = 80

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

export interface FetchCredentialProvidersResult {
  items: CredentialProviderListItem[]
  total: number
}

export async function fetchCredentialProviders(): Promise<FetchCredentialProvidersResult> {
  await sleep(NETWORK_DELAY_MS)
  return {
    items: credentialProviderListData,
    total: credentialProviderListData.length,
  }
}

export interface CredentialProviderMetadata {
  entityId: string
  createdAt: string
  createdBy: string
  lastModifiedAt: string
  lastModifiedBy: string
  isActive: boolean
}

export interface FetchCredentialProviderDetailResult {
  values: CredentialProviderFormValues
  metadata: CredentialProviderMetadata
}

export async function fetchCredentialProvider(id: string): Promise<FetchCredentialProviderDetailResult | null> {
  await sleep(NETWORK_DELAY_MS)
  const cp = credentialProvidersData.find((c) => c.id === id)
  if (!cp) return null

  return {
    values: buildCredentialProviderValues(id),
    metadata: {
      entityId: cp.id,
      createdAt: cp.createdAt,
      createdBy: cp.createdBy,
      lastModifiedAt: cp.createdAt,
      lastModifiedBy: cp.createdBy,
      isActive: cp.isActive,
    },
  }
}

export async function saveCredentialProvider(_id: string, values: CredentialProviderFormValues): Promise<CredentialProviderFormValues> {
  await sleep(NETWORK_DELAY_MS)
  return values
}

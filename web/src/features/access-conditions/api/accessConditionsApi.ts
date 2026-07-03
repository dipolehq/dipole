import { accessConditionListData } from '@/data/access-conditions'
import { accessConditionsData } from '@/data/policies/accessConditions'
import { buildAccessConditionValues } from '@/data/policies/policyFormMappers'
import type { AccessConditionListItem } from '@/data/access-conditions'
import type { AccessConditionFormValues } from '@/shared/components/forms/types'

const NETWORK_DELAY_MS = 80

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

export interface FetchAccessConditionsResult {
  items: AccessConditionListItem[]
  total: number
}

export async function fetchAccessConditions(): Promise<FetchAccessConditionsResult> {
  await sleep(NETWORK_DELAY_MS)
  return {
    items: accessConditionListData,
    total: accessConditionListData.length,
  }
}

export interface AccessConditionMetadata {
  entityId: string
  createdAt: string
  createdBy: string
  lastModifiedAt: string
  lastModifiedBy: string
  isActive: boolean
}

export interface FetchAccessConditionDetailResult {
  values: AccessConditionFormValues
  metadata: AccessConditionMetadata
}

export async function fetchAccessCondition(id: string): Promise<FetchAccessConditionDetailResult | null> {
  await sleep(NETWORK_DELAY_MS)
  const ac = accessConditionsData.find((c) => c.id === id)
  if (!ac) return null

  return {
    values: buildAccessConditionValues(id),
    metadata: {
      entityId: ac.id,
      createdAt: ac.createdAt,
      createdBy: ac.createdBy,
      lastModifiedAt: ac.createdAt,
      lastModifiedBy: ac.createdBy,
      isActive: ac.isActive,
    },
  }
}

export async function saveAccessCondition(_id: string, values: AccessConditionFormValues): Promise<AccessConditionFormValues> {
  await sleep(NETWORK_DELAY_MS)
  return values
}

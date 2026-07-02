/**
 * Mock API layer for policies. Today this reads from the static data files in
 * `/src/data/policies/*`; swap the implementation to `fetch()` calls when a
 * real backend exists and every consumer of these hooks keeps working.
 */
import { policyDetailData } from '@/data/policies/policyDetail'
import {
  buildClientWorkloadValues,
  buildServerWorkloadValues,
  buildTrustProviderValuesList,
  buildAccessConditionValuesList,
  buildCredentialProviderValues,
} from '@/data/policies/policyFormMappers'
import type { PolicyFormValues, PolicyMetadata } from '@/features/policies/types/formValues'

const NETWORK_DELAY_MS = 120

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

export interface FetchPolicyResult {
  values: PolicyFormValues
  metadata: PolicyMetadata
}

/**
 * Fetches full form data for a policy. Returns `null` when the id is unknown.
 */
export async function fetchPolicy(id: string): Promise<FetchPolicyResult | null> {
  await sleep(NETWORK_DELAY_MS)

  const policy = policyDetailData[id]
  if (!policy) return null

  const values: PolicyFormValues = {
    name: policy.name,
    description: policy.description,
    status: policy.status,
    tags: policy.tags,

    clientWorkloadId: policy.clientWorkloadId,
    client: buildClientWorkloadValues(policy.clientWorkloadId),

    serverWorkloadId: policy.serverWorkloadId,
    server: buildServerWorkloadValues(policy.serverWorkloadId),

    trustProviderIds: policy.trustProviderIds,
    trustProviders: buildTrustProviderValuesList(policy.trustProviderIds),

    accessConditionIds: policy.accessConditionIds,
    accessConditions: buildAccessConditionValuesList(policy.accessConditionIds),

    credentialProviderId: policy.credentialProviderId,
    credential: buildCredentialProviderValues(policy.credentialProviderId),
  }

  const metadata: PolicyMetadata = {
    policyId: policy.policyId,
    createdAt: policy.createdAt,
    createdBy: policy.createdBy,
    lastModifiedAt: policy.lastModifiedAt,
    lastModifiedBy: policy.lastModifiedBy,
  }

  return { values, metadata }
}

/**
 * Persists a policy edit. This mock implementation just echoes the input back;
 * real code would POST/PUT to the backend.
 */
export async function savePolicy(_id: string, values: PolicyFormValues): Promise<PolicyFormValues> {
  await sleep(NETWORK_DELAY_MS)
  return values
}

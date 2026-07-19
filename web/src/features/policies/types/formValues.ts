import type {
  ClientWorkloadFormValues,
  ServerWorkloadFormValues,
  TrustProviderFormValues,
  AccessConditionFormValues,
  CredentialProviderFormValues,
  FormTag,
} from '@/shared/components/forms/types'

/**
 * The unified schema managed by react-hook-form at the top of the Policy form.
 * Every tab reads/writes into a well-known slice of this object.
 */
export interface PolicyFormValues {
  // Details tab
  name: string
  description: string
  status: string // 'active' | 'inactive' | 'deprecated'
  tags: FormTag[]

  // Client Workload tab
  clientWorkloadId: string
  client: ClientWorkloadFormValues

  // Server Workload tab
  serverWorkloadId: string
  server: ServerWorkloadFormValues

  // Trust Providers tab (multi)
  trustProviderIds: string[]
  trustProviders: TrustProviderFormValues[]

  // Access Conditions tab (multi)
  accessConditionIds: string[]
  accessConditions: AccessConditionFormValues[]

  // Credential Provider tab
  credentialProviderId: string
  credential: CredentialProviderFormValues
}

/** Metadata attached to a persisted policy (not editable in the form). */
export interface PolicyMetadata {
  policyId: string
  createdAt: string
  createdBy: string
  lastModifiedAt: string
  lastModifiedBy: string
}

export type PolicyFormMode = 'create' | 'view' | 'edit'

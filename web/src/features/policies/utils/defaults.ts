import {
  EMPTY_CLIENT_WORKLOAD,
  EMPTY_SERVER_WORKLOAD,
  EMPTY_CREDENTIAL_PROVIDER,
} from '@/shared/components/forms/types'
import type { PolicyFormValues } from '@/features/policies/types/formValues'

export const EMPTY_POLICY_FORM_VALUES: PolicyFormValues = {
  name: '',
  description: '',
  status: '',
  tags: [],

  clientWorkloadId: '',
  client: { ...EMPTY_CLIENT_WORKLOAD },

  serverWorkloadId: '',
  server: { ...EMPTY_SERVER_WORKLOAD },

  trustProviderIds: [],
  trustProviders: [],

  accessConditionIds: [],
  accessConditions: [],

  credentialProviderId: '',
  credential: { ...EMPTY_CREDENTIAL_PROVIDER },
}

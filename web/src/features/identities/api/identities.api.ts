import { apiClient } from '@/shared/lib/api-client'
import type { IdentityListResponse } from '@/features/identities/types/Identity.types'

export function fetchIdentities(page = 1, perPage = 20): Promise<IdentityListResponse> {
  return apiClient.get<IdentityListResponse>(
    `/client-workloads?page=${page}&perPage=${perPage}`,
  )
}

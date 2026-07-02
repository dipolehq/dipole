import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { fetchPolicy, savePolicy } from '@/features/policies/api/policiesApi'
import type { PolicyFormValues } from '@/features/policies/types/formValues'

const POLICY_KEY = ['policies'] as const

export function policyQueryKey(id: string): readonly unknown[] {
  return [...POLICY_KEY, 'detail', id]
}

/**
 * Loads the full form data for a policy by id. Backed by React Query so
 * consumers get caching, refetching, loading and error states for free.
 */
export function usePolicy(id: string | undefined) {
  return useQuery({
    queryKey: policyQueryKey(id ?? ''),
    queryFn: () => {
      if (!id) return Promise.resolve(null)
      return fetchPolicy(id)
    },
    enabled: !!id,
  })
}

/**
 * Mutation for saving edits to an existing policy.
 */
export function useSavePolicy(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (values: PolicyFormValues) => savePolicy(id, values),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: policyQueryKey(id) })
    },
  })
}

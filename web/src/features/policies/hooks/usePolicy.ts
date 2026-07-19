import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  fetchPolicies,
  fetchPolicy,
  createPolicy,
  savePolicy,
  deletePolicy,
} from '@/features/policies/api/policiesApi'
import type { PolicyFormValues } from '@/features/policies/types/formValues'
import type { FetchPoliciesResult } from '@/features/policies/api/policiesApi'

export function usePolicies(page = 1, perPage = 20, search = '') {
  return useQuery<FetchPoliciesResult>({
    queryKey: ['policies', 'list', page, perPage, search],
    queryFn: () => fetchPolicies(page, perPage, search),
  })
}

export function usePolicy(id: string | undefined) {
  return useQuery({
    queryKey: ['policies', 'detail', id],
    queryFn: () => {
      if (!id) return Promise.resolve(null)
      return fetchPolicy(id)
    },
    enabled: !!id,
  })
}

export function useCreatePolicy() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (values: PolicyFormValues) => createPolicy(values),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['policies', 'list'] })
    },
  })
}

export function useSavePolicy(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (values: PolicyFormValues) => savePolicy(id, values),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['policies', 'detail', id] })
      qc.invalidateQueries({ queryKey: ['policies', 'list'] })
    },
  })
}

export function useDeletePolicy() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deletePolicy(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['policies', 'list'] })
    },
  })
}

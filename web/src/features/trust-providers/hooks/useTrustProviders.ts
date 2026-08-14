import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  fetchTrustProviders,
  fetchTrustProvider,
  createTrustProvider,
  saveTrustProvider,
  deleteTrustProvider,
} from '@/features/trust-providers/api/trustProvidersApi'
import type { TrustProviderInput } from '@/features/trust-providers/api/trustProvidersApi'

export function useTrustProviders(page = 1, perPage = 20, search = '') {
  return useQuery({
    queryKey: ['trust-providers', 'list', page, perPage, search],
    queryFn: () => fetchTrustProviders(page, perPage, search),
  })
}

export function useTrustProvider(id: string | undefined) {
  return useQuery({
    queryKey: ['trust-providers', 'detail', id],
    queryFn: () => {
      if (!id) return Promise.resolve(null)
      return fetchTrustProvider(id)
    },
    enabled: !!id,
  })
}

export function useCreateTrustProvider() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: TrustProviderInput) => createTrustProvider(input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['trust-providers', 'list'] })
    },
  })
}

export function useSaveTrustProvider(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: TrustProviderInput) => saveTrustProvider(id, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['trust-providers', 'detail', id] })
      qc.invalidateQueries({ queryKey: ['trust-providers', 'list'] })
    },
  })
}

export function useDeleteTrustProvider() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteTrustProvider(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['trust-providers', 'list'] })
    },
  })
}

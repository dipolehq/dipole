import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { fetchTrustProviders, fetchTrustProvider, saveTrustProvider } from '@/features/trust-providers/api/trustProvidersApi'
import type { TrustProviderFormValues } from '@/shared/components/forms/types'

export function useTrustProviders() {
  return useQuery({
    queryKey: ['trust-providers', 'list'],
    queryFn: fetchTrustProviders,
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

export function useSaveTrustProvider(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (values: TrustProviderFormValues) => saveTrustProvider(id, values),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['trust-providers', 'detail', id] })
      qc.invalidateQueries({ queryKey: ['trust-providers', 'list'] })
    },
  })
}

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { fetchCredentialProviders, fetchCredentialProvider, saveCredentialProvider } from '@/features/credential-providers/api/credentialProvidersApi'
import type { CredentialProviderFormValues } from '@/shared/components/forms/types'

export function useCredentialProviders() {
  return useQuery({
    queryKey: ['credential-providers', 'list'],
    queryFn: fetchCredentialProviders,
  })
}

export function useCredentialProvider(id: string | undefined) {
  return useQuery({
    queryKey: ['credential-providers', 'detail', id],
    queryFn: () => {
      if (!id) return Promise.resolve(null)
      return fetchCredentialProvider(id)
    },
    enabled: !!id,
  })
}

export function useSaveCredentialProvider(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (values: CredentialProviderFormValues) => saveCredentialProvider(id, values),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['credential-providers', 'detail', id] })
      qc.invalidateQueries({ queryKey: ['credential-providers', 'list'] })
    },
  })
}

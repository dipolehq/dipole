import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  fetchCredentialProviders,
  fetchCredentialProvider,
  createCredentialProvider,
  saveCredentialProvider,
  deleteCredentialProvider,
} from '@/features/credential-providers/api/credentialProvidersApi'
import type { CredentialProviderInput } from '@/features/credential-providers/api/credentialProvidersApi'

export function useCredentialProviders(page = 1, perPage = 20, search = '') {
  return useQuery({
    queryKey: ['credential-providers', 'list', page, perPage, search],
    queryFn: () => fetchCredentialProviders(page, perPage, search),
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

export function useCreateCredentialProvider() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CredentialProviderInput) => createCredentialProvider(input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['credential-providers', 'list'] })
    },
  })
}

export function useSaveCredentialProvider(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CredentialProviderInput) => saveCredentialProvider(id, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['credential-providers', 'detail', id] })
      qc.invalidateQueries({ queryKey: ['credential-providers', 'list'] })
    },
  })
}

export function useDeleteCredentialProvider() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteCredentialProvider(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['credential-providers', 'list'] })
    },
  })
}

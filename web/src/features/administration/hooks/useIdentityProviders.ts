import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  fetchIdentityProviders,
  fetchIdentityProvider,
  createIdentityProvider,
  saveIdentityProvider,
  deleteIdentityProvider,
} from '@/features/administration/api/identityProvidersApi'
import type { IdpInput } from '@/features/administration/api/identityProvidersApi'

export function useIdentityProviders(page = 1, perPage = 20, search = '') {
  return useQuery({
    queryKey: ['identity-providers', 'list', page, perPage, search],
    queryFn: () => fetchIdentityProviders(page, perPage, search),
  })
}

export function useIdentityProvider(id: string | undefined) {
  return useQuery({
    queryKey: ['identity-providers', 'detail', id],
    queryFn: () => {
      if (!id) return Promise.resolve(null)
      return fetchIdentityProvider(id)
    },
    enabled: !!id,
  })
}

export function useCreateIdentityProvider() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: IdpInput) => createIdentityProvider(input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['identity-providers'] })
    },
  })
}

export function useSaveIdentityProvider(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: IdpInput) => saveIdentityProvider(id, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['identity-providers'] })
    },
  })
}

export function useDeleteIdentityProvider() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteIdentityProvider(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['identity-providers'] })
    },
  })
}

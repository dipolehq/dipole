import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  fetchResourceSets,
  fetchResourceSet,
  createResourceSet,
  saveResourceSet,
  deleteResourceSet,
} from '@/features/administration/api/resourceSetsApi'
import type { ResourceSetInput } from '@/features/administration/api/resourceSetsApi'

export function useResourceSets(page = 1, perPage = 20, search = '') {
  return useQuery({
    queryKey: ['resource-sets', 'list', page, perPage, search],
    queryFn: () => fetchResourceSets(page, perPage, search),
  })
}

export function useResourceSet(id: string | undefined) {
  return useQuery({
    queryKey: ['resource-sets', 'detail', id],
    queryFn: () => {
      if (!id) return Promise.resolve(null)
      return fetchResourceSet(id)
    },
    enabled: !!id,
  })
}

export function useCreateResourceSet() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: ResourceSetInput) => createResourceSet(input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['resource-sets'] })
    },
  })
}

export function useSaveResourceSet(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: ResourceSetInput) => saveResourceSet(id, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['resource-sets'] })
    },
  })
}

export function useDeleteResourceSet() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteResourceSet(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['resource-sets'] })
    },
  })
}

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  fetchServerWorkloads,
  fetchServerWorkload,
  createServerWorkload,
  saveServerWorkload,
} from '@/features/server-workloads/api/serverWorkloadsApi'
import type { ServerWorkloadInput } from '@/features/server-workloads/api/serverWorkloadsApi'

export function useServerWorkloads(page = 1, perPage = 20, search = '') {
  return useQuery({
    queryKey: ['server-workloads', 'list', page, perPage, search],
    queryFn: () => fetchServerWorkloads(page, perPage, search),
  })
}

export function useServerWorkload(id: string | undefined) {
  return useQuery({
    queryKey: ['server-workloads', 'detail', id],
    queryFn: () => {
      if (!id) return Promise.resolve(null)
      return fetchServerWorkload(id)
    },
    enabled: !!id,
  })
}

export function useCreateServerWorkload() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: ServerWorkloadInput) => createServerWorkload(input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['server-workloads', 'list'] })
    },
  })
}

export function useSaveServerWorkload(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: ServerWorkloadInput) => saveServerWorkload(id, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['server-workloads', 'detail', id] })
      qc.invalidateQueries({ queryKey: ['server-workloads', 'list'] })
    },
  })
}

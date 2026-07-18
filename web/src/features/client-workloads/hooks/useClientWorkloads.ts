import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  fetchClientWorkloads,
  fetchClientWorkload,
  createClientWorkload,
  saveClientWorkload,
} from '@/features/client-workloads/api/clientWorkloadsApi'
import type { ClientWorkloadInput } from '@/features/client-workloads/api/clientWorkloadsApi'

export function useClientWorkloads(page = 1, perPage = 20, search = '') {
  return useQuery({
    queryKey: ['client-workloads', 'list', page, perPage, search],
    queryFn: () => fetchClientWorkloads(page, perPage, search),
  })
}

export function useClientWorkload(id: string | undefined) {
  return useQuery({
    queryKey: ['client-workloads', 'detail', id],
    queryFn: () => {
      if (!id) return Promise.resolve(null)
      return fetchClientWorkload(id)
    },
    enabled: !!id,
  })
}

export function useCreateClientWorkload() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: ClientWorkloadInput) => createClientWorkload(input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['client-workloads', 'list'] })
    },
  })
}

export function useSaveClientWorkload(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: ClientWorkloadInput) => saveClientWorkload(id, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['client-workloads', 'detail', id] })
      qc.invalidateQueries({ queryKey: ['client-workloads', 'list'] })
    },
  })
}

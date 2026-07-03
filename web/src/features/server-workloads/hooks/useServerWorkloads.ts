import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { fetchServerWorkloads, fetchServerWorkload, saveServerWorkload } from '@/features/server-workloads/api/serverWorkloadsApi'
import type { ServerWorkloadFormValues } from '@/shared/components/forms/types'

export function useServerWorkloads() {
  return useQuery({
    queryKey: ['server-workloads', 'list'],
    queryFn: fetchServerWorkloads,
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

export function useSaveServerWorkload(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (values: ServerWorkloadFormValues) => saveServerWorkload(id, values),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['server-workloads', 'detail', id] })
      qc.invalidateQueries({ queryKey: ['server-workloads', 'list'] })
    },
  })
}

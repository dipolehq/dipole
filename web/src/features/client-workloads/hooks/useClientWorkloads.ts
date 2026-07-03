import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { fetchClientWorkloads, fetchClientWorkload, saveClientWorkload } from '@/features/client-workloads/api/clientWorkloadsApi'
import type { ClientWorkloadFormValues } from '@/shared/components/forms/types'

export function useClientWorkloads() {
  return useQuery({
    queryKey: ['client-workloads', 'list'],
    queryFn: fetchClientWorkloads,
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

export function useSaveClientWorkload(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (values: ClientWorkloadFormValues) => saveClientWorkload(id, values),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['client-workloads', 'detail', id] })
      qc.invalidateQueries({ queryKey: ['client-workloads', 'list'] })
    },
  })
}

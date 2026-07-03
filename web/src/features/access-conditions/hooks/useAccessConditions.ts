import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { fetchAccessConditions, fetchAccessCondition, saveAccessCondition } from '@/features/access-conditions/api/accessConditionsApi'
import type { AccessConditionFormValues } from '@/shared/components/forms/types'

export function useAccessConditions() {
  return useQuery({
    queryKey: ['access-conditions', 'list'],
    queryFn: fetchAccessConditions,
  })
}

export function useAccessCondition(id: string | undefined) {
  return useQuery({
    queryKey: ['access-conditions', 'detail', id],
    queryFn: () => {
      if (!id) return Promise.resolve(null)
      return fetchAccessCondition(id)
    },
    enabled: !!id,
  })
}

export function useSaveAccessCondition(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (values: AccessConditionFormValues) => saveAccessCondition(id, values),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['access-conditions', 'detail', id] })
      qc.invalidateQueries({ queryKey: ['access-conditions', 'list'] })
    },
  })
}

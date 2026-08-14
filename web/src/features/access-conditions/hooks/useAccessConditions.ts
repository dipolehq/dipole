import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  fetchAccessConditions,
  fetchAccessCondition,
  createAccessCondition,
  saveAccessCondition,
  deleteAccessCondition,
} from '@/features/access-conditions/api/accessConditionsApi'
import type { AccessConditionInput } from '@/features/access-conditions/api/accessConditionsApi'

export function useAccessConditions(page = 1, perPage = 20, search = '') {
  return useQuery({
    queryKey: ['access-conditions', 'list', page, perPage, search],
    queryFn: () => fetchAccessConditions(page, perPage, search),
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

export function useCreateAccessCondition() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: AccessConditionInput) => createAccessCondition(input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['access-conditions', 'list'] })
    },
  })
}

export function useSaveAccessCondition(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: AccessConditionInput) => saveAccessCondition(id, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['access-conditions', 'detail', id] })
      qc.invalidateQueries({ queryKey: ['access-conditions', 'list'] })
    },
  })
}

export function useDeleteAccessCondition() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteAccessCondition(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['access-conditions', 'list'] })
    },
  })
}

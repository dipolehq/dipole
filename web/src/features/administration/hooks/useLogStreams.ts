import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  fetchLogStreams,
  fetchLogStream,
  createLogStream,
  saveLogStream,
  deleteLogStream,
} from '@/features/administration/api/logStreamsApi'
import type { LogStreamInput } from '@/features/administration/api/logStreamsApi'

export function useLogStreams(page = 1, perPage = 20, search = '') {
  return useQuery({
    queryKey: ['log-streams', 'list', page, perPage, search],
    queryFn: () => fetchLogStreams(page, perPage, search),
  })
}

export function useLogStream(id: string | undefined) {
  return useQuery({
    queryKey: ['log-streams', 'detail', id],
    queryFn: () => {
      if (!id) return Promise.resolve(null)
      return fetchLogStream(id)
    },
    enabled: !!id,
  })
}

export function useCreateLogStream() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: LogStreamInput) => createLogStream(input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['log-streams'] })
    },
  })
}

export function useSaveLogStream(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: LogStreamInput) => saveLogStream(id, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['log-streams'] })
    },
  })
}

export function useDeleteLogStream() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteLogStream(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['log-streams'] })
    },
  })
}

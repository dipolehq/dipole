import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  fetchAgentControllers,
  fetchAgentController,
  createAgentController,
  saveAgentController,
  deleteAgentController,
  regenerateDeviceCode,
} from '@/features/edge-components/api/agentControllersApi'
import type { AgentControllerInput } from '@/features/edge-components/api/agentControllersApi'

export function useAgentControllers(page = 1, perPage = 20, search = '') {
  return useQuery({
    queryKey: ['agent-controllers', 'list', page, perPage, search],
    queryFn: () => fetchAgentControllers(page, perPage, search),
  })
}

export function useAgentController(id: string | undefined) {
  return useQuery({
    queryKey: ['agent-controllers', 'detail', id],
    queryFn: () => {
      if (!id) return Promise.resolve(null)
      return fetchAgentController(id)
    },
    enabled: !!id,
  })
}

export function useCreateAgentController() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: AgentControllerInput) => createAgentController(input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['agent-controllers', 'list'] })
    },
  })
}

export function useSaveAgentController(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: AgentControllerInput) => saveAgentController(id, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['agent-controllers', 'detail', id] })
      qc.invalidateQueries({ queryKey: ['agent-controllers', 'list'] })
    },
  })
}

export function useDeleteAgentController() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteAgentController(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['agent-controllers', 'list'] })
    },
  })
}

export function useRegenerateDeviceCode() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (controllerId: string) => regenerateDeviceCode(controllerId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['agent-controllers'] })
    },
  })
}

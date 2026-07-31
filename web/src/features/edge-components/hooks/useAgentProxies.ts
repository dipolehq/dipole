import { useQuery } from '@tanstack/react-query'
import { fetchAgentProxies } from '@/features/edge-components/api/agentProxiesApi'

export function useAgentProxies(page = 1, perPage = 20, search = '') {
  return useQuery({
    queryKey: ['agent-proxies', 'list', page, perPage, search],
    queryFn: () => fetchAgentProxies(page, perPage, search),
  })
}

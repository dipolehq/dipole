import { useQuery } from '@tanstack/react-query'
import { fetchIdentities } from '@/features/identities/api/identities.api'

export function useIdentities(page = 1, perPage = 20) {
  return useQuery({
    queryKey: ['identities', page, perPage],
    queryFn: () => fetchIdentities(page, perPage),
  })
}

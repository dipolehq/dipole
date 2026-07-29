import { createContext, useContext, useState, useEffect, type ReactNode } from 'react'
import { useQuery } from '@tanstack/react-query'
import { apiClient, setActiveResourceSet } from '@/shared/lib/api-client'

export interface ResourceSet {
  id: string
  name: string
  description: string
}

interface ResourceSetContextValue {
  current: ResourceSet
  all: ResourceSet[]
  setCurrent: (rs: ResourceSet) => void
  isLoading: boolean
}

const DEFAULT_RESOURCE_SET: ResourceSet = {
  id: 'ffffffff-ffff-ffff-ffff-ffffffffffff',
  name: 'Default',
  description: '',
}

const ResourceSetContext = createContext<ResourceSetContextValue>({
  current: DEFAULT_RESOURCE_SET,
  all: [DEFAULT_RESOURCE_SET],
  setCurrent: () => {},
  isLoading: false,
})

export function ResourceSetProvider({ children }: { children: ReactNode }) {
  const [current, setCurrent] = useState<ResourceSet>(DEFAULT_RESOURCE_SET)

  // Sync the active Resource Set to the API client module whenever selection changes.
  useEffect(() => {
    setActiveResourceSet(current.id)
  }, [current.id])

  const { data, isLoading } = useQuery({
    queryKey: ['resource-sets', 'global'],
    queryFn: async () => {
      const res = await apiClient.get<{ items: ResourceSet[] }>('/resource-sets?perPage=100')
      return res.items
    },
    staleTime: 60_000,
  })

  const all = data ?? [DEFAULT_RESOURCE_SET]

  return (
    <ResourceSetContext.Provider value={{ current, all, setCurrent, isLoading }}>
      {children}
    </ResourceSetContext.Provider>
  )
}

export function useResourceSetContext() {
  return useContext(ResourceSetContext)
}

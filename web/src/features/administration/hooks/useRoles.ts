import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { fetchRoles, fetchRole, createRole, saveRole, deleteRole } from '@/features/administration/api/rolesApi'
import type { RoleInput } from '@/features/administration/api/rolesApi'

export function useRoles(page = 1, perPage = 20, search = '') {
  return useQuery({
    queryKey: ['roles', 'list', page, perPage, search],
    queryFn: () => fetchRoles(page, perPage, search),
  })
}

export function useRole(id: string | undefined) {
  return useQuery({
    queryKey: ['roles', 'detail', id],
    queryFn: () => {
      if (!id) return Promise.resolve(null)
      return fetchRole(id)
    },
    enabled: !!id,
  })
}

export function useCreateRole() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: RoleInput) => createRole(input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['roles'] })
    },
  })
}

export function useSaveRole(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: RoleInput) => saveRole(id, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['roles'] })
    },
  })
}

export function useDeleteRole() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteRole(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['roles'] })
    },
  })
}

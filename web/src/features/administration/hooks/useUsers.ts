import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { fetchUsers, fetchUser, createUser, saveUser, deleteUser } from '@/features/administration/api/usersApi'
import type { UserInput } from '@/features/administration/api/usersApi'

export function useUsers(page = 1, perPage = 20, search = '') {
  return useQuery({
    queryKey: ['users', 'list', page, perPage, search],
    queryFn: () => fetchUsers(page, perPage, search),
  })
}

export function useUser(id: string | undefined) {
  return useQuery({
    queryKey: ['users', 'detail', id],
    queryFn: () => {
      if (!id) return Promise.resolve(null)
      return fetchUser(id)
    },
    enabled: !!id,
  })
}

export function useCreateUser() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: UserInput) => createUser(input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['users', 'list'] })
    },
  })
}

export function useSaveUser(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: UserInput) => saveUser(id, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['users', 'detail', id] })
      qc.invalidateQueries({ queryKey: ['users', 'list'] })
    },
  })
}

export function useDeleteUser() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteUser(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['users', 'list'] })
    },
  })
}

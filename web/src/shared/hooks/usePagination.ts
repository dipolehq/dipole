import { useState } from 'react'

interface PaginationState {
  page: number
  perPage: number
  setPage: (page: number) => void
  setPerPage: (perPage: number) => void
  nextPage: () => void
  prevPage: () => void
}

export function usePagination(initialPerPage = 20): PaginationState {
  const [page, setPage] = useState(1)
  const [perPage, setPerPage] = useState(initialPerPage)

  return {
    page,
    perPage,
    setPage,
    setPerPage,
    nextPage: () => setPage((p) => p + 1),
    prevPage: () => setPage((p) => Math.max(1, p - 1)),
  }
}

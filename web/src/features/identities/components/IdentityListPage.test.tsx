import { render, screen } from '@testing-library/react'
import { QueryClientProvider, QueryClient } from '@tanstack/react-query'
import { describe, it, expect, vi } from 'vitest'
import { IdentityListPage } from './IdentityListPage'

vi.mock('@/features/identities/hooks/useIdentities', () => ({
  useIdentities: () => ({
    data: {
      items: [
        { id: '1', name: 'Test Workload', identifiers: [{ type: 'k8s_namespace', value: 'prod' }], description: '', createdAt: '', updatedAt: '' },
      ],
      total: 1,
      page: 1,
      perPage: 20,
    },
    isLoading: false,
    error: null,
  }),
}))

function renderWithProviders(ui: React.ReactElement) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(<QueryClientProvider client={client}>{ui}</QueryClientProvider>)
}

describe('IdentityListPage', () => {
  it('renders identity items', () => {
    renderWithProviders(<IdentityListPage />)
    expect(screen.getByText('Test Workload')).toBeInTheDocument()
    expect(screen.getByText('1 identifier')).toBeInTheDocument()
  })
})

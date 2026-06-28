import type { IdentityListResponse } from '@/features/identities/types/Identity.types'

const MOCK_IDENTITIES: IdentityListResponse = {
  items: [
    {
      id: '7c466803-9dd4-4388-9e45-420c57a0432c',
      name: 'Payments Service',
      description: 'Handles payment processing',
      identifiers: [
        { type: 'k8s_namespace', value: 'production' },
        { type: 'k8s_pod_name_prefix', value: 'payments-' },
      ],
      createdAt: '2026-06-15T10:30:00Z',
      updatedAt: '2026-06-15T10:30:00Z',
    },
    {
      id: 'a1b2c3d4-5678-9abc-def0-111222333444',
      name: 'CI/CD Pipeline',
      description: 'GitHub Actions deploy workflow',
      identifiers: [
        { type: 'github_id_token_repository', value: 'dipolehq/payments' },
      ],
      createdAt: '2026-06-10T08:00:00Z',
      updatedAt: '2026-06-10T08:00:00Z',
    },
    {
      id: 'b2c3d4e5-6789-abcd-ef01-222333444555',
      name: 'Support Resolution Agent',
      description: 'AI agent that handles customer support tickets',
      identifiers: [
        { type: 'k8s_namespace', value: 'ai-agents' },
        { type: 'k8s_service_account_name', value: 'support-agent-sa' },
      ],
      createdAt: '2026-06-20T14:15:00Z',
      updatedAt: '2026-06-25T09:00:00Z',
    },
  ],
  total: 3,
  page: 1,
  perPage: 20,
}

export function fetchIdentities(_page = 1, _perPage = 20): Promise<IdentityListResponse> {
  // Mock: returns static data with a small delay to simulate network
  return new Promise((resolve) => {
    setTimeout(() => resolve(MOCK_IDENTITIES), 300)
  })
}

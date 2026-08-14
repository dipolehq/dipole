import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Copy, Trash2 } from 'lucide-react'
import { apiClient } from '@/shared/lib/api-client'
import { FloatingInput, FloatingDropdown } from '@/shared/components/FloatingInput'
import { useResourceSetContext } from '@/shared/context/ResourceSetContext'

interface TokenListItem {
  id: string
  name: string
  resourceSetId?: string
  createdAt: string
}

export function APITokensSection() {
  const queryClient = useQueryClient()
  const { all: resourceSets } = useResourceSetContext()

  const [tokenName, setTokenName] = useState('API Token')
  const [scopeResourceSetId, setScopeResourceSetId] = useState('global')
  const [generatedToken, setGeneratedToken] = useState('')

  const { data: tokenListData } = useQuery({
    queryKey: ['profile', 'tokens'],
    queryFn: () => apiClient.get<{ tokens: TokenListItem[] }>('/profile/tokens'),
  })
  const tokens = tokenListData?.tokens ?? []

  const generateMutation = useMutation({
    mutationFn: () =>
      apiClient.post<{ token: string; id: string; name: string; resourceSetId?: string }>('/profile/token', {
        name: tokenName,
        resourceSetId: scopeResourceSetId === 'global' ? undefined : scopeResourceSetId,
      }),
    onSuccess: (res) => {
      setGeneratedToken(res.token)
      queryClient.invalidateQueries({ queryKey: ['profile', 'tokens'] })
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => apiClient.delete(`/profile/tokens/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['profile', 'tokens'] })
    },
  })

  function getResourceSetName(id?: string): string {
    if (!id) return 'All (Global)'
    const rs = resourceSets.find((r) => r.id === id)
    return rs?.name ?? id.slice(0, 8) + '…'
  }

  return (
    <div className="border border-border rounded-lg p-5">
      <h4 className="text-sm font-semibold text-gray-800 mb-4">API Tokens</h4>

      {/* Generate new token */}
      <div className="space-y-3 mb-5">
        <div className="flex gap-2 items-end">
          <div className="flex-1">
            <FloatingInput
              label="Token Name"
              value={tokenName}
              onChange={setTokenName}
            />
          </div>
          <div className="w-56">
            <FloatingDropdown
              label="Resource Set"
              options={[
                { value: 'global', label: 'All Resource Sets (Global)' },
                ...resourceSets.map((rs) => ({ value: rs.id, label: rs.name })),
              ]}
              value={scopeResourceSetId}
              onChange={setScopeResourceSetId}
            />
          </div>
          <button
            type="button"
            onClick={() => generateMutation.mutate()}
            disabled={generateMutation.isPending || !tokenName.trim()}
            className="inline-flex items-center gap-2 h-[44px] px-4 rounded-md border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors disabled:opacity-50 whitespace-nowrap"
          >
            {generateMutation.isPending ? 'Generating…' : 'Generate Token'}
          </button>
        </div>

        {generatedToken && (
          <div className="bg-green-50 border border-green-200 rounded-md p-3">
            <div className="flex items-center justify-between mb-1">
              <span className="text-[10px] text-green-700 font-medium">New token (shown once — copy it now):</span>
              <button
                type="button"
                onClick={() => navigator.clipboard.writeText(generatedToken)}
                className="text-green-600 hover:text-green-800"
              >
                <Copy size={12} />
              </button>
            </div>
            <code className="text-xs font-mono text-green-900 break-all select-all">{generatedToken}</code>
          </div>
        )}
      </div>

      {/* Token list */}
      {tokens.length > 0 && (
        <div>
          <h5 className="text-xs font-semibold text-gray-600 mb-2">Issued Tokens</h5>
          <div className="border border-border rounded-md overflow-hidden">
            <table className="w-full text-xs">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-3 py-2 text-left text-gray-600 font-semibold">Name</th>
                  <th className="px-3 py-2 text-left text-gray-600 font-semibold">Scope</th>
                  <th className="px-3 py-2 text-left text-gray-600 font-semibold">Created</th>
                  <th className="px-3 py-2 w-8" />
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {tokens.map((t) => (
                  <tr key={t.id} className="hover:bg-gray-50">
                    <td className="px-3 py-2 text-gray-800">{t.name}</td>
                    <td className="px-3 py-2 text-gray-600">{getResourceSetName(t.resourceSetId)}</td>
                    <td className="px-3 py-2 text-gray-500">{new Date(t.createdAt).toLocaleDateString()}</td>
                    <td className="px-3 py-2">
                      <button
                        type="button"
                        onClick={() => { if (confirm('Revoke this token?')) deleteMutation.mutate(t.id) }}
                        className="text-gray-400 hover:text-red-500 transition-colors"
                      >
                        <Trash2 size={13} />
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  )
}

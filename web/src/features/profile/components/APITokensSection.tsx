import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Copy, Trash2 } from 'lucide-react'
import { apiClient } from '@/shared/lib/api-client'
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
  const [scopeResourceSetId, setScopeResourceSetId] = useState('')
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
        resourceSetId: scopeResourceSetId || undefined,
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
        <div className="flex gap-2">
          <input
            type="text"
            value={tokenName}
            onChange={(e) => setTokenName(e.target.value)}
            placeholder="Token name"
            className="flex-1 px-3 py-2 border border-gray-300 rounded-md text-xs focus:outline-none focus:ring-1 focus:ring-orange-500"
          />
          <select
            value={scopeResourceSetId}
            onChange={(e) => setScopeResourceSetId(e.target.value)}
            className="px-3 py-2 border border-gray-300 rounded-md text-xs focus:outline-none focus:ring-1 focus:ring-orange-500 bg-white"
          >
            <option value="">All Resource Sets (Global)</option>
            {resourceSets.map((rs) => (
              <option key={rs.id} value={rs.id}>{rs.name}</option>
            ))}
          </select>
          <button
            type="button"
            onClick={() => generateMutation.mutate()}
            disabled={generateMutation.isPending || !tokenName.trim()}
            className="px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors disabled:opacity-50 whitespace-nowrap"
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

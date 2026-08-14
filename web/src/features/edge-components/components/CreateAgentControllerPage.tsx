import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { PageHeader } from '@/shared/components/PageHeader'
import { FloatingInput, FloatingTextarea, FloatingDropdown } from '@/shared/components/FloatingInput'
import { TagsField } from '@/shared/components/TagsField'
import { useCreateAgentController } from '@/features/edge-components/hooks/useAgentControllers'
import { apiErrorMessage } from '@/features/edge-components/api/agentControllersApi'
import { fetchTrustProviders } from '@/features/policies/api/entityApi'
import type { FormTag } from '@/shared/components/forms/types'

export function CreateAgentControllerPage() {
  const navigate = useNavigate()
  const createMutation = useCreateAgentController()

  const [error, setError] = useState<string | null>(null)
  const [status, setStatus] = useState('active')
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [trustProviderId, setTrustProviderId] = useState('')
  const [allowedTlsHostname, setAllowedTlsHostname] = useState('')
  const [tags, setTags] = useState<FormTag[]>([])

  const { data: tpData } = useQuery({
    queryKey: ['trust-providers', 'dropdown'],
    queryFn: () => fetchTrustProviders({ perPage: 100 }),
    staleTime: 60_000,
  })
  const trustProviderOptions = (tpData?.items ?? []).map((tp) => ({ value: tp.id, label: tp.name }))

  async function handleSave() {
    setError(null)
    try {
      await createMutation.mutateAsync({
        name,
        description,
        isActive: status === 'active',
        trustProviderId: trustProviderId || undefined,
        tlsHostname: allowedTlsHostname || undefined,
        tags,
      })
      navigate('/edge-components')
    } catch (err) {
      setError(apiErrorMessage(err, 'Failed to create agent controller'))
    }
  }

  function handleCancel() {
    navigate('/edge-components')
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-4 shrink-0 h-[44px] border-b border-border bg-white">
        <PageHeader />
      </div>

      <div className="flex items-center justify-end gap-2 px-4 py-3 shrink-0">
        <button type="button" onClick={handleSave} disabled={createMutation.isPending} className="inline-flex items-center gap-2 h-9 px-4 rounded-md border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors disabled:opacity-50">
          {createMutation.isPending ? 'Saving…' : 'Save'}
        </button>
        <button type="button" onClick={handleCancel} className="inline-flex items-center gap-2 h-9 px-4 rounded-md border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors">
          Cancel
        </button>
      </div>

      {error && (
        <div className="mx-4 mb-2 px-4 py-2 bg-red-50 border border-red-200 rounded-md text-sm text-red-700">
          {error}
        </div>
      )}

      <div className="flex-1 min-h-0 mx-4 border border-border bg-white overflow-y-auto">
        <div className="grid grid-cols-[1fr_380px] h-full">
          <div className="px-6 py-5 overflow-y-auto border-r border-border">
            <div className="space-y-6">
              <div>
                <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider mb-4">Details</h4>
                <div className="space-y-4">
                  <FloatingDropdown label="Status" options={[{ value: 'active', label: 'Active' }, { value: 'inactive', label: 'Inactive' }]} value={status} onChange={setStatus} />
                  <FloatingInput label="Name" value={name} onChange={setName} required />
                  <FloatingTextarea label="Description (Optional)" value={description} onChange={setDescription} rows={2} />
                </div>
              </div>

              <div>
                <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider mb-4">Configuration</h4>
                <div className="space-y-4">
                  <FloatingDropdown label="Trust Provider" options={trustProviderOptions} value={trustProviderId} onChange={setTrustProviderId} />
                  <FloatingInput label="Allowed TLS Hostname (Optional)" value={allowedTlsHostname} onChange={setAllowedTlsHostname} />
                </div>
              </div>

              <div>
                <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider mb-4">Tags</h4>
                <TagsField value={tags} onChange={setTags} />
              </div>
            </div>
          </div>

          <div className="px-5 py-5 overflow-y-auto">
            <div className="space-y-3">
              <p className="text-xs text-gray-600 leading-relaxed">
                <span className="font-semibold text-gray-800">Agent Controllers</span> manage Agent Proxy registration and token provisioning. They act as a registration broker between Agent Proxies and the Dipole Cloud.
              </p>
              <p className="text-xs text-gray-600 leading-relaxed">
                Associating a <span className="font-semibold text-gray-800">Trust Provider</span> enables automated identity attestation during registration, removing the need for manual device codes.
              </p>
              <p className="text-xs text-gray-600 leading-relaxed">
                The <span className="font-semibold text-gray-800">Allowed TLS Hostname</span> specifies the FQDN or wildcard domain included in the managed TLS certificate for secure communication.
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

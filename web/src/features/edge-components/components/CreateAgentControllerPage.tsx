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
  const trustProviderOptions = (tpData?.items ?? []).map((tp) => ({
    value: tp.id,
    label: tp.name,
  }))

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
      {error && (
        <div className="px-6 py-2 bg-red-50 border-b border-red-200 text-sm text-red-700 shrink-0">
          {error}
        </div>
      )}

      <div className="flex items-center justify-between px-4 shrink-0 h-[60px] border-b border-border bg-white">
        <PageHeader />
        <div className="flex items-center gap-3">
          <button type="button" onClick={handleSave} disabled={createMutation.isPending} className="px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white text-sm font-medium rounded-md transition-colors disabled:opacity-60">
            {createMutation.isPending ? 'Saving…' : 'Save'}
          </button>
          <button type="button" onClick={handleCancel} className="px-4 py-2 border border-gray-300 text-sm font-medium text-gray-600 hover:text-gray-900 hover:border-gray-400 rounded-md transition-colors">Cancel</button>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto bg-white">
        <div className="flex h-full">
          <div className="w-[60%] px-6 py-6 overflow-y-auto">
            <div className="space-y-5">
              <div className="border border-border rounded-lg p-5">
                <h4 className="text-sm font-semibold text-gray-800 mb-4">Details</h4>
                <div className="space-y-4">
                  <FloatingDropdown
                    label="Status"
                    options={[
                      { value: 'active', label: 'Active' },
                      { value: 'inactive', label: 'Inactive' },
                    ]}
                    value={status}
                    onChange={setStatus}
                  />
                  <FloatingInput label="Name" value={name} onChange={setName} required />
                  <FloatingTextarea label="Description (Optional)" value={description} onChange={setDescription} rows={2} />
                </div>
              </div>

              <div className="border border-border rounded-lg p-5">
                <h4 className="text-sm font-semibold text-gray-800 mb-4">Configuration</h4>
                <div className="space-y-4">
                  <FloatingDropdown
                    label="Trust Provider"
                    options={trustProviderOptions}
                    value={trustProviderId}
                    onChange={setTrustProviderId}
                  />
                  <FloatingInput label="Allowed TLS Hostname (Optional)" value={allowedTlsHostname} onChange={setAllowedTlsHostname} />
                </div>
              </div>

              <div className="border border-border rounded-lg p-5">
                <h4 className="text-sm font-semibold text-gray-800 mb-4">Tags</h4>
                <TagsField value={tags} onChange={setTags} />
              </div>
            </div>
          </div>

          <div className="w-[40%] px-6 py-6 overflow-y-auto">
            <div className="border border-border rounded-lg p-5 sticky top-0">
              <p className="text-xs text-gray-700 leading-relaxed mb-3">
                <span className="font-semibold text-orange-600">Agent Controllers</span> manage Agent Proxy registration and token provisioning. They act as a registration broker between Agent Proxies and the Dipole Cloud.
              </p>
              <p className="text-xs text-gray-700 leading-relaxed mb-3">
                Associating a <span className="font-semibold">Trust Provider</span> enables automated identity attestation during registration, removing the need for manual device codes.
              </p>
              <p className="text-xs text-gray-700 leading-relaxed">
                The <span className="font-semibold">Allowed TLS Hostname</span> specifies the FQDN or wildcard domain included in the managed TLS certificate for secure communication.
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { PageHeader } from '@/shared/components/PageHeader'
import { Dropdown } from '@/shared/components/Dropdown'
import { TagsField } from '@/shared/components/TagsField'

const TRUST_PROVIDER_OPTIONS = [
  { value: '', label: 'None' },
  { value: '1', label: 'AWS Production' },
  { value: '2', label: 'AWS Staging' },
  { value: '3', label: 'Kubernetes Production' },
]

const STATUS_OPTIONS = [
  { value: 'active', label: 'Active' },
  { value: 'inactive', label: 'Inactive' },
]

export function CreateAgentControllerPage() {
  const navigate = useNavigate()

  const [status, setStatus] = useState('active')
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [trustProviderId, setTrustProviderId] = useState('')
  const [allowedTlsHostname, setAllowedTlsHostname] = useState('')

  function handleSave() {
    console.info('[create-agent-controller]', { name, description, trustProviderId, allowedTlsHostname })
    navigate('/edge-components')
  }

  function handleCancel() {
    navigate('/edge-components')
  }

  return (
    <div className="px-4 py-3 flex flex-col h-full">
      <div className="flex items-center justify-between mb-4 shrink-0 min-h-[36px]">
        <PageHeader />
      </div>

      <div className="bg-white rounded-lg border border-border flex-1 flex min-h-0">
        <nav className="w-48 border-r border-border py-3 shrink-0 flex flex-col">
          <div className="flex-1">
            <button
              type="button"
              className="w-full text-left px-5 py-2.5 text-xs font-medium text-orange-600 bg-orange-50 relative"
            >
              <span className="absolute left-0 top-0 bottom-0 w-0.5 bg-orange-500 rounded-r" />
              <span>Controller Details</span>
            </button>
          </div>
          <div className="px-4 py-3 border-t border-border space-y-2">
            <button
              type="button"
              onClick={handleSave}
              className="w-full px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors"
            >
              Save Controller
            </button>
            <button
              type="button"
              onClick={handleCancel}
              className="w-full px-4 py-2 border border-gray-300 text-xs font-medium text-gray-600 hover:text-gray-900 hover:border-gray-400 rounded-md transition-colors"
            >
              Cancel
            </button>
          </div>
        </nav>

        <div className="flex-1 overflow-y-auto p-6">
          <div>
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-sm font-semibold text-gray-900">Agent Controller</h3>
              <button
                type="button"
                onClick={handleSave}
                className="px-4 py-1.5 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors"
              >
                Save
              </button>
            </div>
            <div className="border-b border-border mb-6" />

            <div className="space-y-4">
              <div>
                <label className="block text-xs font-medium text-gray-700 mb-1.5">Status</label>
                <Dropdown
                  options={STATUS_OPTIONS}
                  value={status}
                  onChange={setStatus}
                  placeholder="Select status..."
                />
              </div>
              <div>
                <label className="block text-xs text-gray-500 mb-1">Name <span className="text-red-500">*</span></label>
                <input
                  type="text"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="Agent Controller name"
                  className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400"
                />
              </div>
              <div>
                <label className="block text-xs text-gray-500 mb-1">Description (Optional)</label>
                <textarea
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="Description (Optional)"
                  rows={2}
                  className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400 resize-y"
                />
              </div>
              <div>
                <label className="block text-xs text-gray-500 mb-1">Trust Provider</label>
                <Dropdown
                  options={TRUST_PROVIDER_OPTIONS}
                  value={trustProviderId}
                  onChange={setTrustProviderId}
                  placeholder="Select a Trust Provider..."
                />
                <p className="text-[11px] text-gray-400 mt-1">
                  Associating a Trust Provider enables automated identity attestation during registration.
                </p>
              </div>
              <div>
                <label className="block text-xs text-gray-500 mb-1">Allowed TLS Hostname (Optional)</label>
                <input
                  type="text"
                  value={allowedTlsHostname}
                  onChange={(e) => setAllowedTlsHostname(e.target.value)}
                  placeholder="e.g., controller.prod.dipolehq.com"
                  className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400"
                />
                <p className="text-[11px] text-gray-400 mt-1">
                  FQDN or wildcard domain (e.g., *.example.com) included in the managed TLS certificate.
                </p>
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-700 mb-1.5">Tags</label>
                <TagsField />
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

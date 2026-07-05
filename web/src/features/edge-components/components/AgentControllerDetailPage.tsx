import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { PageHeader } from '@/shared/components/PageHeader'
import { Dropdown } from '@/shared/components/Dropdown'
import { TagsField } from '@/shared/components/TagsField'
import { agentControllersData } from '@/data/edge-components/agentControllers'

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

export function AgentControllerDetailPage() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const controller = agentControllersData.find((ac) => ac.id === id)

  const [isEditing, setIsEditing] = useState(false)
  const [status, setStatus] = useState('active')
  const [name, setName] = useState(controller?.name ?? '')
  const [description, setDescription] = useState(controller?.description ?? '')
  const [trustProviderId, setTrustProviderId] = useState(controller?.trustProviderId ?? '')
  const [allowedTlsHostname, setAllowedTlsHostname] = useState(controller?.allowedTlsHostname ?? '')

  if (!controller) {
    return (
      <div className="p-6">
        <PageHeader />
        <p className="mt-6 text-sm text-gray-500">Agent Controller not found.</p>
        <button onClick={() => navigate('/edge-components')} className="mt-4 text-xs text-orange-500 hover:text-orange-600 font-medium">← Back to Edge Components</button>
      </div>
    )
  }

  const disabled = !isEditing

  function handleEditToggle() {
    if (isEditing) {
      console.info('[save-agent-controller]', { status, name, description, trustProviderId, allowedTlsHostname })
      setIsEditing(false)
    } else {
      setIsEditing(true)
    }
  }

  function handleCancel() {
    if (isEditing) {
      setName(controller?.name ?? '')
      setDescription(controller?.description ?? '')
      setTrustProviderId(controller?.trustProviderId ?? '')
      setAllowedTlsHostname(controller?.allowedTlsHostname ?? '')
      setStatus('active')
      setIsEditing(false)
    } else {
      navigate('/edge-components')
    }
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
              onClick={handleEditToggle}
              className="w-full px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors"
            >
              {isEditing ? 'Save Controller' : 'Edit Controller'}
            </button>
            <button
              type="button"
              onClick={handleCancel}
              className="w-full px-4 py-2 border border-gray-300 text-xs font-medium text-gray-600 hover:text-gray-900 hover:border-gray-400 rounded-md transition-colors"
            >
              {isEditing ? 'Cancel' : 'Back'}
            </button>
          </div>
        </nav>

        <div className="flex-1 overflow-y-auto p-6">
          <div>
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-sm font-semibold text-gray-900">Agent Controller</h3>
              {isEditing && (
                <button type="button" onClick={handleEditToggle} className="px-4 py-1.5 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors">Save</button>
              )}
            </div>
            <div className="border-b border-border mb-6" />

            <div className="space-y-4">
              {/* Status */}
              <div>
                <label className="block text-xs font-medium text-gray-700 mb-1.5">Status</label>
                {disabled ? (
                  <div className="px-3 py-2 border border-border rounded-md text-sm text-gray-600 bg-gray-50 capitalize">{status}</div>
                ) : (
                  <Dropdown
                    options={STATUS_OPTIONS}
                    value={status}
                    onChange={setStatus}
                    placeholder="Select status..."
                  />
                )}
              </div>

              {/* Agent Controller ID */}
              <div>
                <label className="block text-xs text-gray-500 mb-1">Agent Controller ID</label>
                <div className="relative">
                  <div className="w-full px-3 py-2 pr-10 border border-border rounded-md text-sm text-gray-500 bg-gray-50 font-mono">{controller.id}</div>
                  <button
                    type="button"
                    onClick={() => navigator.clipboard.writeText(controller.id)}
                    className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 transition-colors"
                    title="Copy ID"
                  >
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24"><rect x="9" y="9" width="13" height="13" rx="2" /><path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1" /></svg>
                  </button>
                </div>
              </div>

              {/* Name */}
              <div>
                <label className="block text-xs text-gray-500 mb-1">Name <span className="text-red-500">*</span></label>
                <input
                  type="text"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  disabled={disabled}
                  placeholder="Agent Controller name"
                  className={`w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400 ${disabled ? 'bg-gray-50 text-gray-600' : ''}`}
                />
              </div>

              {/* Description */}
              <div>
                <label className="block text-xs text-gray-500 mb-1">Description (Optional)</label>
                <textarea
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  disabled={disabled}
                  placeholder="Description (Optional)"
                  rows={2}
                  className={`w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400 resize-y ${disabled ? 'bg-gray-50 text-gray-600' : ''}`}
                />
              </div>

              {/* Trust Provider */}
              <div>
                <label className="block text-xs text-gray-500 mb-1">Trust Provider</label>
                {disabled ? (
                  <div className="px-3 py-2 border border-border rounded-md text-sm text-gray-600 bg-gray-50">
                    {controller.trustProviderName ?? 'None'}
                  </div>
                ) : (
                  <Dropdown
                    options={TRUST_PROVIDER_OPTIONS}
                    value={trustProviderId}
                    onChange={setTrustProviderId}
                    placeholder="Select a Trust Provider..."
                  />
                )}
              </div>

              {/* Allowed TLS Hostname */}
              <div>
                <label className="block text-xs text-gray-500 mb-1">Allowed TLS Hostname (Optional)</label>
                <input
                  type="text"
                  value={allowedTlsHostname}
                  onChange={(e) => setAllowedTlsHostname(e.target.value)}
                  disabled={disabled}
                  placeholder="e.g., controller.prod.dipolehq.com"
                  className={`w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400 ${disabled ? 'bg-gray-50 text-gray-600' : ''}`}
                />
              </div>

              {/* Tags */}
              <div>
                <label className="block text-xs font-medium text-gray-700 mb-1.5">Tags</label>
                {disabled ? (
                  <div className="px-3 py-2 border border-border rounded-md text-sm text-gray-500 bg-gray-50">No tags</div>
                ) : (
                  <TagsField />
                )}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

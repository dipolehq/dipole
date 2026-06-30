import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { PageHeader } from '@/shared/components/PageHeader'
import { DetailsTab } from '@/features/policies/components/tabs/details/DetailsTab'
import { ClientWorkloadTab } from '@/features/policies/components/tabs/client-workload/ClientWorkloadTab'
import { ServerWorkloadTab } from '@/features/policies/components/tabs/server-workload/ServerWorkloadTab'
import { TrustProviderTab } from '@/features/policies/components/tabs/trust-provider/TrustProviderTab'
import { AccessConditionTab } from '@/features/policies/components/tabs/access-conditions/AccessConditionTab'
import { CredentialProviderTab } from '@/features/policies/components/tabs/credential-provider/CredentialProviderTab'

const TABS = [
  { key: 'details', label: 'Policy Details' },
  { key: 'client', label: 'Client Workload' },
  { key: 'trust', label: 'Trust Provider' },
  { key: 'conditions', label: 'Access Conditions' },
  { key: 'credential', label: 'Credential Providers' },
  { key: 'server', label: 'Server Workload' },
] as const

type TabKey = typeof TABS[number]['key']

export function CreatePolicyPage() {
  const navigate = useNavigate()
  const [activeTab, setActiveTab] = useState<TabKey>('details')
  const [savedTabs, setSavedTabs] = useState<Set<TabKey>>(new Set())

  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [isActive, setIsActive] = useState(true)
  const [clientWorkloadId, setClientWorkloadId] = useState('')
  const [serverWorkloadId, setServerWorkloadId] = useState('')
  const [trustProviderIds, setTrustProviderIds] = useState<string[]>([])
  const [accessConditionIds, setAccessConditionIds] = useState<string[]>([])
  const [credentialProviderId, setCredentialProviderId] = useState('')

  function toggleArrayItem(arr: string[], id: string): string[] {
    return arr.includes(id) ? arr.filter((i) => i !== id) : [...arr, id]
  }

  function handleSaveTab() {
    setSavedTabs((prev) => new Set([...prev, activeTab]))
  }

  return (
    <div className="p-6 flex flex-col h-screen overflow-hidden">
      <div className="flex items-center justify-between mb-6 shrink-0">
        <PageHeader />
      </div>

      <div className="bg-white rounded-lg border border-border flex-1 flex min-h-0">
        <nav className="w-48 border-r border-border py-3 shrink-0 flex flex-col">
          <div className="flex-1">
            {TABS.map((tab) => (
              <button
                key={tab.key}
                onClick={() => setActiveTab(tab.key)}
                className={`w-full text-left px-5 py-2.5 text-xs font-medium transition-colors relative flex items-center justify-between ${
                  activeTab === tab.key ? 'text-orange-600 bg-orange-50' : 'text-gray-500 hover:text-gray-800 hover:bg-gray-50'
                }`}
              >
                {activeTab === tab.key && <span className="absolute left-0 top-0 bottom-0 w-0.5 bg-orange-500 rounded-r" />}
                <span>{tab.label}</span>
                {savedTabs.has(tab.key) && (
                  <span className="w-4 h-4 bg-green-500 rounded-full flex items-center justify-center shrink-0">
                    <svg className="w-2.5 h-2.5 text-white" fill="none" stroke="currentColor" strokeWidth="3" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" /></svg>
                  </span>
                )}
              </button>
            ))}
          </div>
          <div className="px-4 py-3 border-t border-border space-y-2">
            <button
              onClick={() => navigate('/access-policies')}
              className="w-full px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors"
            >
              Save Policy
            </button>
            <button
              onClick={() => navigate('/access-policies')}
              className="w-full px-4 py-2 border border-gray-300 text-xs font-medium text-gray-600 hover:text-gray-900 hover:border-gray-400 rounded-md transition-colors"
            >
              Cancel
            </button>
          </div>
        </nav>

        <div className="flex-1 overflow-y-auto p-6">
          <div className="pb-12">
            <div className="flex justify-end mb-2">
              <button
                onClick={handleSaveTab}
                className="px-4 py-1.5 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors"
              >
                Save
              </button>
            </div>
            {activeTab === 'details' && (
              <DetailsTab name={name} description={description} isActive={isActive} onNameChange={setName} onDescriptionChange={setDescription} onActiveChange={setIsActive} />
            )}
            {activeTab === 'client' && (
              <ClientWorkloadTab selectedId={clientWorkloadId} onSelect={setClientWorkloadId} />
            )}
            {activeTab === 'server' && (
              <ServerWorkloadTab selectedId={serverWorkloadId} onSelect={setServerWorkloadId} />
            )}
            {activeTab === 'trust' && (
              <TrustProviderTab selectedIds={trustProviderIds} onToggle={(id) => setTrustProviderIds(toggleArrayItem(trustProviderIds, id))} />
            )}
            {activeTab === 'conditions' && (
              <AccessConditionTab selectedIds={accessConditionIds} onToggle={(id) => setAccessConditionIds(toggleArrayItem(accessConditionIds, id))} />
            )}
            {activeTab === 'credential' && (
              <CredentialProviderTab selectedId={credentialProviderId} onSelect={setCredentialProviderId} />
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

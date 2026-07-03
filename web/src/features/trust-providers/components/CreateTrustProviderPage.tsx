import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { PageHeader } from '@/shared/components/PageHeader'
import { Dropdown } from '@/shared/components/Dropdown'
import { TrustProviderForm } from '@/shared/components/forms/TrustProviderForm'
import { EMPTY_TRUST_PROVIDER } from '@/shared/components/forms/types'
import type { TrustProviderFormValues } from '@/shared/components/forms/types'

const TABS = [
  { key: 'details', label: 'Trust Provider' },
] as const

type TabKey = (typeof TABS)[number]['key']

export function CreateTrustProviderPage() {
  const navigate = useNavigate()
  const [activeTab, setActiveTab] = useState<TabKey>('details')
  const [values, setValues] = useState<TrustProviderFormValues>({ ...EMPTY_TRUST_PROVIDER })
  const [status, setStatus] = useState('active')

  function handleSave() {
    console.info('[create-trust-provider] submitted', values)
    navigate('/trust-providers')
  }

  function handleCancel() {
    navigate('/trust-providers')
  }

  return (
    <div className="px-4 py-3 flex flex-col h-full">
      <div className="flex items-center justify-between mb-4 shrink-0 min-h-[36px]">
        <PageHeader />
      </div>

      <div className="bg-white rounded-lg border border-border flex-1 flex min-h-0">
        <nav className="w-48 border-r border-border py-3 shrink-0 flex flex-col">
          <div className="flex-1">
            {TABS.map((tab) => (
              <button
                key={tab.key}
                type="button"
                onClick={() => setActiveTab(tab.key)}
                className={`w-full text-left px-5 py-2.5 text-xs font-medium transition-colors relative ${
                  activeTab === tab.key
                    ? 'text-orange-600 bg-orange-50'
                    : 'text-gray-500 hover:text-gray-800 hover:bg-gray-50'
                }`}
              >
                {activeTab === tab.key && (
                  <span className="absolute left-0 top-0 bottom-0 w-0.5 bg-orange-500 rounded-r" />
                )}
                <span>{tab.label}</span>
              </button>
            ))}
          </div>
          <div className="px-4 py-3 border-t border-border space-y-2">
            <button
              type="button"
              onClick={handleSave}
              className="w-full px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors"
            >
              Save Trust Provider
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
          <div className="pb-12">
            {activeTab === 'details' && (
              <>
                <div className="flex items-center justify-between mb-4">
                  <h3 className="text-sm font-semibold text-gray-900">Trust Provider</h3>
                  <button
                    type="button"
                    onClick={handleSave}
                    className="px-4 py-1.5 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors"
                  >
                    Save
                  </button>
                </div>
                <div className="border-b border-border mb-6" />
                <div className="space-y-6">
                  <div>
                    <label className="block text-xs font-medium text-gray-700 mb-1.5">Status</label>
                    <Dropdown
                      options={[
                        { value: 'active', label: 'Active' },
                        { value: 'inactive', label: 'Inactive' },
                      ]}
                      value={status}
                      onChange={setStatus}
                      placeholder="Select status..."
                    />
                  </div>
                  <TrustProviderForm values={values} onChange={setValues} />
                </div>
              </>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

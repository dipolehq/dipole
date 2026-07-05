import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { PageHeader } from '@/shared/components/PageHeader'
import { FloatingDropdown } from '@/shared/components/FloatingInput'
import { TrustProviderForm } from '@/shared/components/forms/TrustProviderForm'
import { EMPTY_TRUST_PROVIDER } from '@/shared/components/forms/types'
import type { TrustProviderFormValues } from '@/shared/components/forms/types'

export function CreateTrustProviderPage() {
  const navigate = useNavigate()
  const [values, setValues] = useState<TrustProviderFormValues>({ ...EMPTY_TRUST_PROVIDER })
  const [status, setStatus] = useState('')

  function handleSave() {
    console.info('[create-trust-provider] submitted', values)
    navigate('/trust-providers')
  }

  function handleCancel() {
    navigate('/trust-providers')
  }

  return (
    <div className="flex flex-col h-full">
      {/* Header bar */}
      <div className="flex items-center justify-between px-4 py-3 shrink-0 min-h-[36px] border-b border-border bg-white">
        <PageHeader />
        <div className="flex items-center gap-3">
          <button
            type="button"
            onClick={handleSave}
            className="px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white text-sm font-medium rounded-md transition-colors"
          >
            Save
          </button>
          <button
            type="button"
            onClick={handleCancel}
            className="px-4 py-2 border border-gray-300 text-sm font-medium text-gray-600 hover:text-gray-900 hover:border-gray-400 rounded-md transition-colors"
          >
            Cancel
          </button>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto bg-white">
        <div className="flex h-full">
          {/* Form — left 60% */}
          <div className="w-[60%] px-6 py-6 overflow-y-auto">
            <div className="space-y-6">
              <TrustProviderForm
                values={values}
                onChange={setValues}
                statusSlot={
                  <FloatingDropdown
                    label="Status"
                    options={[
                      { value: 'active', label: 'Active' },
                      { value: 'inactive', label: 'Inactive' },
                    ]}
                    value={status}
                    onChange={setStatus}
                  />
                }
              />
            </div>
          </div>

          {/* Help panel — right 40% */}
          <div className="w-[40%] px-6 py-6 overflow-y-auto">
            <div className="border border-border rounded-lg p-5 sticky top-0">
              <p className="text-xs text-gray-700 leading-relaxed mb-3">
                <span className="font-semibold text-orange-600">Trust Providers</span> verify workload identity using cryptographic attestation from the runtime environment. They answer: "Can the environment VOUCH for this workload's identity?"
              </p>
              <p className="text-xs text-gray-700 leading-relaxed mb-3">
                Instead of a shared secret, the workload's environment provides cryptographic proof. Supported types include Kubernetes Service Account tokens, AWS Metadata/Role, GitHub/GitLab OIDC tokens, and generic OIDC providers.
              </p>
              <p className="text-xs text-gray-700 leading-relaxed">
                <span className="font-semibold">Match Rules</span> define which claims or attributes must match for attestation to succeed. All rules use AND logic — every configured rule must pass.
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

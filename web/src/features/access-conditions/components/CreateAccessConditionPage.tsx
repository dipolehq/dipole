import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { PageHeader } from '@/shared/components/PageHeader'
import { FloatingDropdown } from '@/shared/components/FloatingInput'
import { AccessConditionForm } from '@/shared/components/forms/AccessConditionForm'
import { EMPTY_ACCESS_CONDITION } from '@/shared/components/forms/types'
import type { AccessConditionFormValues } from '@/shared/components/forms/types'
import { useCreateAccessCondition } from '@/features/access-conditions/hooks/useAccessConditions'
import { apiErrorMessage } from '@/features/access-conditions/api/accessConditionsApi'

export function CreateAccessConditionPage() {
  const navigate = useNavigate()
  const createMutation = useCreateAccessCondition()
  const [values, setValues] = useState<AccessConditionFormValues>({ ...EMPTY_ACCESS_CONDITION })
  const [status, setStatus] = useState('active')
  const [error, setError] = useState<string | null>(null)

  async function handleSave() {
    setError(null)
    try {
      await createMutation.mutateAsync({ values, isActive: status !== 'inactive' })
      navigate('/access-conditions')
    } catch (err) {
      setError(apiErrorMessage(err, 'Failed to create access condition'))
    }
  }

  function handleCancel() {
    navigate('/access-conditions')
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
            disabled={createMutation.isPending}
            className="px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white text-sm font-medium rounded-md transition-colors disabled:opacity-60"
          >
            {createMutation.isPending ? 'Saving…' : 'Save'}
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

      {error && (
        <div className="px-6 py-2 bg-red-50 border-b border-red-200 text-sm text-red-700 shrink-0">
          {error}
        </div>
      )}

      {/* Content */}
      <div className="flex-1 overflow-y-auto bg-white">
        <div className="flex h-full">
          {/* Form — left 60% */}
          <div className="w-[60%] px-6 py-6 overflow-y-auto">
            <div className="space-y-6">
              <AccessConditionForm
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
                <span className="font-semibold text-orange-600">Access Conditions</span> add
                contextual security checks beyond identity verification. Even if a workload's
                identity is verified, additional conditions must be satisfied before access is
                granted.
              </p>
              <p className="text-xs text-gray-700 leading-relaxed mb-3">
                Supported conditions include <span className="font-semibold">Time-based</span>{' '}
                restrictions (specific days and hours) and{' '}
                <span className="font-semibold">GeoIP</span> restrictions (country and subdivision).
              </p>
              <p className="text-xs text-gray-700 leading-relaxed">
                All conditions use <span className="font-semibold">AND logic</span> — every
                configured condition must pass for authorization to proceed.
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

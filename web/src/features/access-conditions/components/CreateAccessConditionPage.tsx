import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { FloatingDropdown } from '@/shared/components/FloatingInput'
import { AccessConditionForm } from '@/shared/components/forms/AccessConditionForm'
import { EMPTY_ACCESS_CONDITION } from '@/shared/components/forms/types'
import type { AccessConditionFormValues } from '@/shared/components/forms/types'
import { useCreateAccessCondition } from '@/features/access-conditions/hooks/useAccessConditions'
import { apiErrorMessage } from '@/features/access-conditions/api/accessConditionsApi'
import { accessConditionSchema } from '@/shared/lib/validation'

export function CreateAccessConditionPage() {
  const navigate = useNavigate()
  const createMutation = useCreateAccessCondition()
  const [values, setValues] = useState<AccessConditionFormValues>({ ...EMPTY_ACCESS_CONDITION })
  const [status, setStatus] = useState('active')
  const [error, setError] = useState<string | null>(null)

  async function handleSave() {
    setError(null)
    const result = accessConditionSchema.safeParse(values)
    if (!result.success) {
      setError(result.error.errors[0]?.message ?? 'Please fill in all required fields')
      return
    }
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

      {/* Toolbar */}
      <div className="flex items-center justify-end gap-2 px-4 py-3 shrink-0">
        <button
          type="button"
          onClick={handleSave}
          disabled={createMutation.isPending}
          className="inline-flex items-center gap-2 h-9 px-4 rounded-md border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors disabled:opacity-50"
        >
          {createMutation.isPending ? 'Saving…' : 'Save'}
        </button>
        <button
          type="button"
          onClick={handleCancel}
          className="inline-flex items-center gap-2 h-9 px-4 rounded-md border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors"
        >
          Cancel
        </button>
      </div>

      {error && (
        <div className="mx-4 mb-2 px-4 py-2 bg-red-50 border border-red-200 rounded-md text-sm text-red-700">
          {error}
        </div>
      )}

      {/* Content box */}
      <div className="flex-1 min-h-0 mx-4 border border-border bg-white overflow-y-auto">
        <div className="grid grid-cols-[1fr_380px] h-full">
          {/* Form area */}
          <div className="px-6 py-5 overflow-y-auto border-r border-border">
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

          {/* Help text */}
          <div className="px-5 py-5 overflow-y-auto">
            <div className="space-y-3">
              <p className="text-xs text-gray-600 leading-relaxed">
                <span className="font-semibold text-gray-800">Access Conditions</span> add
                contextual security checks beyond identity verification. Even if a workload's
                identity is verified, additional conditions must be satisfied before access is
                granted.
              </p>
              <p className="text-xs text-gray-600 leading-relaxed">
                Supported conditions include <span className="font-semibold text-gray-800">Time-based</span>{' '}
                restrictions (specific days and hours) and{' '}
                <span className="font-semibold text-gray-800">GeoIP</span> restrictions (country and
                subdivision).
              </p>
              <p className="text-xs text-gray-600 leading-relaxed">
                All conditions use <span className="font-semibold text-gray-800">AND logic</span> — every
                configured condition must pass for authorization to proceed.
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

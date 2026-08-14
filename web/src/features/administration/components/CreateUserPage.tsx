import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { PageHeader } from '@/shared/components/PageHeader'
import { FloatingInput, FloatingDropdown } from '@/shared/components/FloatingInput'
import { useCreateUser } from '@/features/administration/hooks/useUsers'
import { apiErrorMessage } from '@/features/administration/api/usersApi'
import { apiClient } from '@/shared/lib/api-client'

const STATUS_OPTIONS = [
  { value: 'active', label: 'Active' },
  { value: 'inactive', label: 'Inactive' },
]

const COUNTRY_CODE_OPTIONS = [
  { value: '+91', label: '+91' },
  { value: '+1', label: '+1' },
  { value: '+44', label: '+44' },
  { value: '+61', label: '+61' },
  { value: '+49', label: '+49' },
  { value: '+81', label: '+81' },
  { value: '+65', label: '+65' },
]

export function CreateUserPage() {
  const navigate = useNavigate()
  const createMutation = useCreateUser()

  const [error, setError] = useState<string | null>(null)
  const [status, setStatus] = useState('active')
  const [firstName, setFirstName] = useState('')
  const [lastName, setLastName] = useState('')
  const [email, setEmail] = useState('')
  const [countryCode, setCountryCode] = useState('+91')
  const [phone, setPhone] = useState('')
  const [selectedRole, setSelectedRole] = useState('')

  const { data: rolesData } = useQuery({
    queryKey: ['roles', 'dropdown'],
    queryFn: () => apiClient.get<{ items: { id: string; name: string }[] }>('/roles?perPage=100'),
    staleTime: 60_000,
  })
  const roleOptions = (rolesData?.items ?? []).map((r) => ({ value: r.id, label: r.name }))

  async function handleSave() {
    setError(null)
    try {
      await createMutation.mutateAsync({
        firstName,
        lastName,
        email,
        phone,
        isActive: status === 'active',
        roleId: selectedRole || undefined,
      })
      navigate('/administration/users')
    } catch (err) {
      setError(apiErrorMessage(err, 'Failed to create user'))
    }
  }

  function handleCancel() {
    navigate('/administration/users')
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-4 shrink-0 h-[44px] border-b border-border bg-white">
        <PageHeader />
      </div>

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

      <div className="flex-1 min-h-0 mx-4 border border-border bg-white overflow-y-auto">
        <div className="grid grid-cols-[1fr_380px] h-full">
          <div className="px-6 py-5 overflow-y-auto border-r border-border">
            <div className="space-y-6">
              <div>
                <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider mb-4">Details</h4>
                <div className="space-y-4">
                  <FloatingDropdown label="Status" options={STATUS_OPTIONS} value={status} onChange={setStatus} />
                  <FloatingInput label="First Name" value={firstName} onChange={setFirstName} required />
                  <FloatingInput label="Last Name" value={lastName} onChange={setLastName} required />
                  <FloatingInput label="Email" value={email} onChange={setEmail} type="email" required />
                  <div className="flex gap-3">
                    <FloatingDropdown label="Code" options={COUNTRY_CODE_OPTIONS} value={countryCode} onChange={setCountryCode} className="w-32" />
                    <div className="flex-1">
                      <FloatingInput label="Phone Number" value={phone} onChange={setPhone} type="tel" />
                    </div>
                  </div>
                </div>
              </div>

              <div>
                <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider mb-4">Role Assignment</h4>
                <FloatingDropdown label="Role" options={roleOptions} value={selectedRole} onChange={setSelectedRole} />
              </div>
            </div>
          </div>

          <div className="px-5 py-5 overflow-y-auto">
            <div className="space-y-3">
              <p className="text-xs text-gray-600 leading-relaxed">
                <span className="font-semibold text-gray-800">Users</span> are administrators who manage the Dipole platform. Each user is assigned a role that determines their level of access.
              </p>
              <p className="text-xs text-gray-600 leading-relaxed">
                Assign a <span className="font-semibold text-gray-800">Role</span> to control what actions the user can perform across entities like Access Policies, Workloads, and Credential Providers.
              </p>
              <p className="text-xs text-gray-600 leading-relaxed">
                Users can also be provisioned automatically via an <span className="font-semibold text-gray-800">Identity Provider</span> (SSO) with role mappings.
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

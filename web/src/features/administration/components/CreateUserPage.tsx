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

  // Fetch roles for dropdown
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
        phone: phone,
        isActive: status === 'active',
        roleId: selectedRole || undefined,
      })
      navigate('/administration')
    } catch (err) {
      setError(apiErrorMessage(err, 'Failed to create user'))
    }
  }

  function handleCancel() {
    navigate('/administration')
  }

  return (
    <div className="flex flex-col h-full">
      {error && (
        <div className="px-6 py-2 bg-red-50 border-b border-red-200 text-sm text-red-700 shrink-0">
          {error}
        </div>
      )}

      <div className="flex items-center justify-between px-4 py-3 shrink-0 min-h-[36px] border-b border-border bg-white">
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

              <div className="border border-border rounded-lg p-5">
                <h4 className="text-sm font-semibold text-gray-800 mb-4">Role Assignment</h4>
                <div className="space-y-4">
                  <FloatingDropdown label="Role" options={roleOptions} value={selectedRole} onChange={setSelectedRole} />
                </div>
              </div>
            </div>
          </div>

          <div className="w-[40%] px-6 py-6 overflow-y-auto">
            <div className="border border-border rounded-lg p-5 sticky top-0">
              <p className="text-xs text-gray-700 leading-relaxed mb-3">
                <span className="font-semibold text-orange-600">Users</span> are administrators who manage the Dipole platform. Each user is assigned a role that determines their level of access.
              </p>
              <p className="text-xs text-gray-700 leading-relaxed mb-3">
                Assign a <span className="font-semibold">Role</span> to control what actions the user can perform across entities like Access Policies, Workloads, and Credential Providers.
              </p>
              <p className="text-xs text-gray-700 leading-relaxed">
                Users can also be provisioned automatically via an <span className="font-semibold">Identity Provider</span> (SSO) with role mappings.
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Pencil, Trash2 } from 'lucide-react'
import { PageHeader } from '@/shared/components/PageHeader'
import { FloatingInput, FloatingDropdown } from '@/shared/components/FloatingInput'
import { DeleteConfirmDialog } from '@/shared/components/ui/DeleteConfirmDialog'
import { useUser, useSaveUser, useDeleteUser } from '@/features/administration/hooks/useUsers'
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

export function UserDetailPage() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()

  const { data: user, isLoading } = useUser(id)
  const saveMutation = useSaveUser(id ?? '')
  const deleteMutation = useDeleteUser()

  const [isEditing, setIsEditing] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)

  const [status, setStatus] = useState('')
  const [firstName, setFirstName] = useState('')
  const [lastName, setLastName] = useState('')
  const [email, setEmail] = useState('')
  const [countryCode, setCountryCode] = useState('+91')
  const [phone, setPhone] = useState('')
  const [selectedRole, setSelectedRole] = useState('')
  const [initialized, setInitialized] = useState(false)

  if (user && !initialized) {
    setStatus(user.isActive ? 'active' : 'inactive')
    setFirstName(user.firstName)
    setLastName(user.lastName)
    setEmail(user.email)
    setPhone(user.phone ?? '')
    setSelectedRole(user.roleId ?? '')
    setInitialized(true)
  }

  if (initialized && user && user.id !== id) {
    setInitialized(false)
  }

  const { data: rolesData } = useQuery({
    queryKey: ['roles', 'dropdown'],
    queryFn: () => apiClient.get<{ items: { id: string; name: string }[] }>('/roles?perPage=100'),
    staleTime: 60_000,
  })
  const roleOptions = (rolesData?.items ?? []).map((r) => ({ value: r.id, label: r.name }))

  if (!id || (!isLoading && user === null)) {
    return (
      <div className="flex flex-col h-full">
        <div className="flex items-center justify-between px-4 shrink-0 h-[44px] border-b border-border bg-white">
          <PageHeader />
        </div>
        <div className="flex-1 flex items-center justify-center">
          <p className="text-sm text-gray-400">User not found.</p>
        </div>
      </div>
    )
  }

  const disabled = !isEditing
  const displayName = `${firstName || ''} ${lastName || ''}`.trim() || 'Untitled'

  async function handleSave() {
    setError(null)
    try {
      await saveMutation.mutateAsync({
        firstName,
        lastName,
        email,
        phone,
        isActive: status === 'active',
        roleId: selectedRole || undefined,
      })
      setIsEditing(false)
    } catch (err) {
      setError(apiErrorMessage(err, 'Failed to save user'))
    }
  }

  function handleEdit() {
    setError(null)
    setIsEditing(true)
  }

  function handleCancel() {
    setError(null)
    if (user) {
      setStatus(user.isActive ? 'active' : 'inactive')
      setFirstName(user.firstName)
      setLastName(user.lastName)
      setEmail(user.email)
      setPhone(user.phone ?? '')
      setSelectedRole(user.roleId ?? '')
    }
    setIsEditing(false)
  }

  async function handleDelete() {
    try {
      await deleteMutation.mutateAsync(id!)
      navigate('/administration/users')
    } catch (err) {
      setError(apiErrorMessage(err, 'Failed to delete user'))
    }
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-4 shrink-0 h-[44px] border-b border-border bg-white">
        <PageHeader />
      </div>

      <div className="flex items-center justify-between gap-3 px-4 py-3 shrink-0">
        <div className="flex items-center gap-3">
          <h2 className="text-sm font-medium text-gray-900">{displayName}</h2>
          <span className={`text-xs font-medium px-2 py-0.5 rounded ${status === 'active' ? 'bg-green-50 text-green-700' : 'bg-gray-100 text-gray-500'}`}>
            {status === 'active' ? 'Active' : 'Inactive'}
          </span>
        </div>
        <div className="flex items-center gap-2">
          {isEditing ? (
            <>
              <button type="button" onClick={handleSave} disabled={saveMutation.isPending} className="inline-flex items-center gap-2 h-9 px-4 rounded-md border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors disabled:opacity-50">
                {saveMutation.isPending ? 'Saving…' : 'Save'}
              </button>
              <button type="button" onClick={handleCancel} className="inline-flex items-center gap-2 h-9 px-4 rounded-md border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors">
                Cancel
              </button>
            </>
          ) : (
            <>
              <button type="button" onClick={handleEdit} className="inline-flex items-center gap-2 h-9 px-4 rounded-md border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors">
                <Pencil size={14} strokeWidth={2} className="opacity-60" /> Edit
              </button>
              <button type="button" onClick={() => setDeleteDialogOpen(true)} className="inline-flex items-center gap-2 h-9 px-4 rounded-md border border-red-200 bg-white text-sm font-medium text-red-600 hover:bg-red-50 transition-colors">
                <Trash2 size={14} strokeWidth={2} className="opacity-60" /> Delete
              </button>
            </>
          )}
        </div>
      </div>

      {error && (
        <div className="mx-4 mb-2 px-4 py-2 bg-red-50 border border-red-200 rounded-md text-sm text-red-700">
          {error}
        </div>
      )}

      <div className="flex-1 min-h-0 mx-4 border border-border bg-white overflow-y-auto">
        <div className="grid grid-cols-[1fr_380px] h-full">
          <div className="px-6 py-5 overflow-y-auto border-r border-border">
            <fieldset disabled={disabled} className={disabled ? 'opacity-60 pointer-events-none' : ''}>
              <div className="space-y-6">
                <div>
                  <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider mb-4">Details</h4>
                  <div className="space-y-4">
                    <FloatingDropdown label="Status" options={STATUS_OPTIONS} value={status} onChange={setStatus} disabled={disabled} />
                    <FloatingInput label="First Name" value={firstName} onChange={setFirstName} disabled={disabled} required />
                    <FloatingInput label="Last Name" value={lastName} onChange={setLastName} disabled={disabled} required />
                    <FloatingInput label="Email" value={email} onChange={setEmail} disabled={disabled} type="email" required />
                    <div className="flex gap-3">
                      <FloatingDropdown label="Code" options={COUNTRY_CODE_OPTIONS} value={countryCode} onChange={setCountryCode} disabled={disabled} className="w-32" />
                      <div className="flex-1">
                        <FloatingInput label="Phone Number" value={phone} onChange={setPhone} disabled={disabled} type="tel" />
                      </div>
                    </div>
                  </div>
                </div>

                <div>
                  <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider mb-4">Role Assignment</h4>
                  <FloatingDropdown label="Role" options={roleOptions} value={selectedRole} onChange={setSelectedRole} disabled={disabled} />
                </div>
              </div>
            </fieldset>
          </div>

          <div className="px-5 py-5 overflow-y-auto">
            <div className="space-y-4">
              <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider">Metadata</h4>
              {user && (
                <>
                  <FloatingInput label="ID" value={user.id} disabled />
                  <FloatingInput label="Created" value={new Date(user.createdAt).toLocaleString('en-IN', { dateStyle: 'medium', timeStyle: 'short' })} disabled />
                  <FloatingInput label="Last Modified" value={new Date(user.updatedAt).toLocaleString('en-IN', { dateStyle: 'medium', timeStyle: 'short' })} disabled />
                </>
              )}
            </div>
          </div>
        </div>
      </div>

      <DeleteConfirmDialog
        open={deleteDialogOpen}
        onClose={() => setDeleteDialogOpen(false)}
        onConfirm={() => { setDeleteDialogOpen(false); handleDelete() }}
        resourceName={displayName}
        resourceId={id ?? ''}
        resourceType="user"
        loading={deleteMutation.isPending}
      />
    </div>
  )
}

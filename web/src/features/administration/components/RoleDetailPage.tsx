import { useState, useEffect, useRef } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { Pencil, Trash2, X } from 'lucide-react'
import { useQuery } from '@tanstack/react-query'
import { PageHeader } from '@/shared/components/PageHeader'
import { FloatingInput, FloatingTextarea, FloatingDropdown } from '@/shared/components/FloatingInput'
import { DeleteConfirmDialog } from '@/shared/components/ui/DeleteConfirmDialog'
import { useClickOutside } from '@/shared/hooks/useClickOutside'
import { useRole, useSaveRole, useDeleteRole } from '@/features/administration/hooks/useRoles'
import { apiErrorMessage } from '@/features/administration/api/rolesApi'
import { apiClient } from '@/shared/lib/api-client'
import { PERMISSION_ENTITIES, type PermissionLevel } from '@/features/administration/permissionEntities'

const STATUS_OPTIONS = [
  { value: 'active', label: 'Active' },
  { value: 'inactive', label: 'Inactive' },
]

export function RoleDetailPage() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()

  const { data: role, isLoading } = useRole(id)
  const saveMutation = useSaveRole(id ?? '')
  const deleteMutation = useDeleteRole()

  const { data: resourceSetsData } = useQuery({
    queryKey: ['resource-sets', 'dropdown'],
    queryFn: () => apiClient.get<{ items: { id: string; name: string }[] }>('/resource-sets?perPage=100'),
    staleTime: 60_000,
  })
  const resourceSetOptions = (resourceSetsData?.items ?? []).map((rs) => ({ value: rs.id, label: rs.name }))

  const [isEditing, setIsEditing] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [status, setStatus] = useState('')
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [resourceSets, setResourceSets] = useState<string[]>(['default'])
  const [permissions, setPermissions] = useState<Record<string, PermissionLevel>>(
    Object.fromEntries(PERMISSION_ENTITIES.map((e) => [e.id, 'no_access' as PermissionLevel]))
  )

  useEffect(() => {
    if (role) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setStatus(role.isActive ? 'active' : 'inactive')
      setName(role.name)
      setDescription(role.description)
      const perms = (role.permissions ?? {}) as Record<string, string>
      setPermissions(Object.fromEntries(PERMISSION_ENTITIES.map((e) => [e.id, (perms[e.id] as PermissionLevel) ?? 'no_access'])))
    }
  }, [role])

  if (!id || (!isLoading && role === null)) {
    return (
      <div className="flex flex-col h-full">
        <div className="flex items-center justify-between px-4 shrink-0 h-[44px] border-b border-border bg-white"><PageHeader /></div>
        <div className="flex-1 flex items-center justify-center">
          <p className="text-sm text-gray-400">Role not found.</p>
        </div>
      </div>
    )
  }

  const disabled = !isEditing

  function handlePermissionChange(entityId: string, level: PermissionLevel) {
    if (disabled) return
    setPermissions({ ...permissions, [entityId]: level })
  }

  function handleEdit() {
    setError(null)
    setIsEditing(true)
  }

  async function handleSave() {
    setError(null)
    try {
      await saveMutation.mutateAsync({
        name,
        description,
        isActive: status === 'active',
        permissions: permissions as Record<string, string>,
        resourceSetIds: resourceSets,
      })
      setIsEditing(false)
    } catch (err) {
      setError(apiErrorMessage(err, 'Failed to save role'))
    }
  }

  function handleCancel() {
    setError(null)
    if (role) {
      setStatus(role.isActive ? 'active' : 'inactive')
      setName(role.name)
      setDescription(role.description)
      const perms = (role.permissions ?? {}) as Record<string, string>
      setPermissions(Object.fromEntries(PERMISSION_ENTITIES.map((e) => [e.id, (perms[e.id] as PermissionLevel) ?? 'no_access'])))
    }
    setIsEditing(false)
  }

  async function handleDelete() {
    try {
      if (!id) return
      await deleteMutation.mutateAsync(id)
      navigate('/administration/roles')
    } catch (err) {
      setError(apiErrorMessage(err, 'Failed to delete role'))
    }
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-4 shrink-0 h-[44px] border-b border-border bg-white">
        <PageHeader />
      </div>

      <div className="flex items-center justify-between gap-3 px-4 py-3 shrink-0">
        <div className="flex items-center gap-3">
          <h2 className="text-sm font-medium text-gray-900">{name || 'Untitled'}</h2>
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
                  <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider mb-4">Role Details</h4>
                  <div className="space-y-4">
                    <FloatingDropdown label="Status" options={STATUS_OPTIONS} value={status} onChange={setStatus} disabled={disabled} />
                    <FloatingInput label="Name" value={name} onChange={setName} disabled={disabled} required />
                    <FloatingTextarea label="Description (Optional)" value={description} onChange={setDescription} disabled={disabled} rows={2} />
                  </div>
                </div>

                <div>
                  <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider mb-4">Resource Set Assignments</h4>
                  <ResourceSetMultiSelect options={resourceSetOptions} selected={resourceSets} onChange={setResourceSets} disabled={disabled} />
                </div>

                <div>
                  <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider mb-4">Permissions</h4>
                  <PermissionsTable permissions={permissions} onChange={handlePermissionChange} disabled={disabled} />
                </div>
              </div>
            </fieldset>
          </div>

          <div className="px-5 py-5 overflow-y-auto">
            <div className="space-y-4">
              <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider">Metadata</h4>
              {role && (
                <>
                  <FloatingInput label="ID" value={role.id} disabled />
                  <FloatingInput label="Created" value={new Date(role.createdAt).toLocaleString('en-IN', { dateStyle: 'medium', timeStyle: 'short' })} disabled />
                  <FloatingInput label="Last Modified" value={new Date(role.updatedAt).toLocaleString('en-IN', { dateStyle: 'medium', timeStyle: 'short' })} disabled />
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
        resourceName={name || 'Untitled'}
        resourceId={id ?? ''}
        resourceType="role"
        loading={deleteMutation.isPending}
      />
    </div>
  )
}

function PermissionsTable({ permissions, onChange, disabled = false }: { permissions: Record<string, PermissionLevel>; onChange: (entityId: string, level: PermissionLevel) => void; disabled?: boolean }) {
  return (
    <div className="border border-border rounded-lg overflow-hidden">
      <table className="w-full text-xs">
        <thead>
          <tr className="bg-gray-50 border-b border-border">
            <th className="px-4 py-3 text-[11px] font-bold text-gray-700 uppercase tracking-wider text-left">Entity</th>
            <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-center w-28 whitespace-nowrap">No Access</th>
            <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-center w-28 whitespace-nowrap">Read Only</th>
            <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-center w-28 whitespace-nowrap">Read/Write</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-border">
          {PERMISSION_ENTITIES.map((entity) => (
            <tr key={entity.id} className="hover:bg-gray-50 transition-colors">
              <td className="px-4 py-3 text-gray-700">{entity.label}</td>
              <td className="px-4 py-3 text-center"><input type="checkbox" checked={permissions[entity.id] === 'no_access'} onChange={() => onChange(entity.id, 'no_access')} disabled={disabled} className="w-3.5 h-3.5 accent-gray-600 cursor-pointer disabled:cursor-default disabled:opacity-60" /></td>
              <td className="px-4 py-3 text-center"><input type="checkbox" checked={permissions[entity.id] === 'read_only'} onChange={() => onChange(entity.id, 'read_only')} disabled={disabled} className="w-3.5 h-3.5 accent-gray-600 cursor-pointer disabled:cursor-default disabled:opacity-60" /></td>
              <td className="px-4 py-3 text-center"><input type="checkbox" checked={permissions[entity.id] === 'read_write'} onChange={() => onChange(entity.id, 'read_write')} disabled={disabled} className="w-3.5 h-3.5 accent-gray-600 cursor-pointer disabled:cursor-default disabled:opacity-60" /></td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function ResourceSetMultiSelect({ options, selected, onChange, disabled = false }: { options: { value: string; label: string }[]; selected: string[]; onChange: (selected: string[]) => void; disabled?: boolean }) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)
  useClickOutside(ref, () => setOpen(false))

  const displayText = selected.length > 0 ? options.filter((o) => selected.includes(o.value)).map((o) => o.label).join(', ') : ''

  return (
    <div ref={ref} className="relative">
      <button type="button" onClick={() => !disabled && setOpen(!open)} disabled={disabled} className={`w-full flex items-center justify-between px-3 py-2 border rounded-md text-sm transition-colors ${open ? 'border-gray-400 ring-1 ring-gray-300' : 'border-border'} ${displayText ? 'text-gray-900' : 'text-gray-400'} ${disabled ? 'bg-gray-50 cursor-default' : ''}`}>
        <span className="truncate">{displayText || 'Select Resource Set(s)...'}</span>
        <svg className={`w-3.5 h-3.5 shrink-0 text-gray-600 transition-transform ${open ? 'rotate-180' : ''}`} fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" /></svg>
      </button>
      {open && !disabled && (
        <div className="absolute top-full left-0 right-0 mt-1 bg-white border border-border rounded-md shadow-lg z-50 max-h-48 overflow-y-auto">
          {options.map((opt) => (
            <button key={opt.value} type="button" onClick={() => { if (selected.includes(opt.value)) onChange(selected.filter((v) => v !== opt.value)); else onChange([...selected, opt.value]) }} className={`w-full text-left px-4 py-2.5 text-sm transition-colors ${selected.includes(opt.value) ? 'bg-gray-100 text-gray-700 font-medium' : 'text-gray-600 hover:bg-gray-50'}`}>
              {opt.label}
            </button>
          ))}
        </div>
      )}
      {selected.length > 0 && !disabled && (
        <div className="mt-2 space-y-1">
          {selected.map((value) => {
            const label = options.find((o) => o.value === value)?.label ?? value
            return (
              <div key={value} className="flex items-center justify-between px-3 py-2 bg-gray-100 rounded-md">
                <span className="text-sm text-gray-700">{label}</span>
                <button type="button" onClick={() => onChange(selected.filter((v) => v !== value))} className="text-gray-500 hover:text-gray-700 transition-colors"><X size={14} /></button>
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}

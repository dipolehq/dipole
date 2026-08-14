import { useState, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { X } from 'lucide-react'
import { PageHeader } from '@/shared/components/PageHeader'
import { FloatingInput, FloatingTextarea, FloatingDropdown } from '@/shared/components/FloatingInput'
import { useClickOutside } from '@/shared/hooks/useClickOutside'
import { useCreateRole } from '@/features/administration/hooks/useRoles'
import { apiErrorMessage } from '@/features/administration/api/rolesApi'
import { apiClient } from '@/shared/lib/api-client'

const STATUS_OPTIONS = [
  { value: 'active', label: 'Active' },
  { value: 'inactive', label: 'Inactive' },
]

type PermissionLevel = 'no_access' | 'read_only' | 'read_write'

const PERMISSION_ENTITIES = [
  'Access Policies', 'Client Workloads', 'Trust Providers', 'Access Conditions',
  'Integrations', 'Credential Providers', 'Server Workloads', 'Agent Controllers',
  'Access Authorization Events', 'Audit Logs', 'Workload Events', 'Users', 'Roles',
  'Log Streams', 'Identity Providers', 'Resource Sets', 'Global Policy Compliance',
] as const

export function CreateRolePage() {
  const navigate = useNavigate()
  const createMutation = useCreateRole()

  // Fetch resource sets and existing roles from the backend
  const { data: resourceSetsData } = useQuery({
    queryKey: ['resource-sets', 'dropdown'],
    queryFn: () => apiClient.get<{ items: { id: string; name: string }[] }>('/resource-sets?perPage=100'),
    staleTime: 60_000,
  })
  const resourceSetOptions = (resourceSetsData?.items ?? []).map((rs) => ({ value: rs.id, label: rs.name }))

  const { data: rolesData } = useQuery({
    queryKey: ['roles', 'dropdown'],
    queryFn: () => apiClient.get<{ items: { id: string; name: string; permissions?: Record<string, string> }[] }>('/roles?perPage=100'),
    staleTime: 60_000,
  })
  const copyFromRoleOptions = [
    { value: 'none', label: 'None' },
    ...(rolesData?.items ?? []).map((r) => ({ value: r.id, label: r.name })),
  ]
  const [error, setError] = useState<string | null>(null)
  const [status, setStatus] = useState('active')
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [resourceSets, setResourceSets] = useState<string[]>([])
  const [copyFromRole, setCopyFromRole] = useState('')
  const [permissions, setPermissions] = useState<Record<string, PermissionLevel>>(
    Object.fromEntries(PERMISSION_ENTITIES.map((e) => [e, 'no_access' as PermissionLevel]))
  )

  function handlePermissionChange(entity: string, level: PermissionLevel) {
    setPermissions({ ...permissions, [entity]: level })
  }

  function handleCopyFromRole(roleId: string) {
    setCopyFromRole(roleId)
    if (roleId === 'none') return
    const role = rolesData?.items?.find((r) => r.id === roleId)
    if (role?.permissions) {
      const perms = role.permissions as Record<string, string>
      setPermissions(Object.fromEntries(PERMISSION_ENTITIES.map((e) => [e, (perms[e] as PermissionLevel) ?? 'no_access'])))
    }
  }

  async function handleSave() {
    setError(null)
    try {
      await createMutation.mutateAsync({
        name,
        description,
        isActive: status === 'active',
        permissions: permissions as Record<string, string>,
        resourceSetIds: resourceSets,
      })
      navigate('/administration/roles')
    } catch (err) {
      setError(apiErrorMessage(err, 'Failed to create role'))
    }
  }

  function handleCancel() {
    navigate('/administration/roles')
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-4 shrink-0 h-[44px] border-b border-border bg-white">
        <PageHeader />
      </div>

      <div className="flex items-center justify-end gap-2 px-4 py-3 shrink-0">
        <button type="button" onClick={handleSave} disabled={createMutation.isPending} className="inline-flex items-center gap-2 h-9 px-4 rounded-md border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors disabled:opacity-50">
          {createMutation.isPending ? 'Saving…' : 'Save'}
        </button>
        <button type="button" onClick={handleCancel} className="inline-flex items-center gap-2 h-9 px-4 rounded-md border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors">
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
                <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider mb-4">Role Details</h4>
                <div className="space-y-4">
                  <FloatingDropdown label="Status" options={STATUS_OPTIONS} value={status} onChange={setStatus} />
                  <FloatingInput label="Name" value={name} onChange={setName} required />
                  <FloatingTextarea label="Description (Optional)" value={description} onChange={setDescription} rows={2} />
                </div>
              </div>

              <div>
                <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider mb-4">Resource Set Assignments</h4>
                <ResourceSetMultiSelect options={resourceSetOptions} selected={resourceSets} onChange={setResourceSets} />
              </div>

              <div>
                <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider mb-4">Permissions</h4>
                <div className="mb-4">
                  <FloatingDropdown label="Copy from existing Role" options={copyFromRoleOptions} value={copyFromRole} onChange={handleCopyFromRole} />
                </div>
                <PermissionsTable permissions={permissions} onChange={handlePermissionChange} />
              </div>
            </div>
          </div>

          <div className="px-5 py-5 overflow-y-auto">
            <div className="space-y-3">
              <p className="text-xs text-gray-600 leading-relaxed">
                <span className="font-semibold text-gray-800">Roles</span> define what actions a user can perform across the Dipole platform. Each role is a collection of permissions scoped to specific entities.
              </p>
              <p className="text-xs text-gray-600 leading-relaxed">
                Use <span className="font-semibold text-gray-800">Resource Set Assignments</span> to limit the role's scope to specific resource groups, enabling multi-tenancy within your organization.
              </p>
              <p className="text-xs text-gray-600 leading-relaxed">
                You can <span className="font-semibold text-gray-800">copy permissions</span> from an existing role as a starting point and customize from there.
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

function PermissionsTable({ permissions, onChange, disabled = false }: { permissions: Record<string, PermissionLevel>; onChange: (entity: string, level: PermissionLevel) => void; disabled?: boolean }) {
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
            <tr key={entity} className="hover:bg-gray-50 transition-colors">
              <td className="px-4 py-3 text-gray-700">{entity}</td>
              <td className="px-4 py-3 text-center">
                <input type="checkbox" checked={permissions[entity] === 'no_access'} onChange={() => onChange(entity, 'no_access')} disabled={disabled} className="w-3.5 h-3.5 accent-gray-600 cursor-pointer disabled:cursor-default disabled:opacity-60" />
              </td>
              <td className="px-4 py-3 text-center">
                <input type="checkbox" checked={permissions[entity] === 'read_only'} onChange={() => onChange(entity, 'read_only')} disabled={disabled} className="w-3.5 h-3.5 accent-gray-600 cursor-pointer disabled:cursor-default disabled:opacity-60" />
              </td>
              <td className="px-4 py-3 text-center">
                <input type="checkbox" checked={permissions[entity] === 'read_write'} onChange={() => onChange(entity, 'read_write')} disabled={disabled} className="w-3.5 h-3.5 accent-gray-600 cursor-pointer disabled:cursor-default disabled:opacity-60" />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function ResourceSetMultiSelect({ options, selected, onChange }: { options: { value: string; label: string }[]; selected: string[]; onChange: (selected: string[]) => void }) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)
  useClickOutside(ref, () => setOpen(false))

  const displayText = selected.length > 0 ? options.filter((o) => selected.includes(o.value)).map((o) => o.label).join(', ') : ''

  return (
    <div ref={ref} className="relative">
      <button type="button" onClick={() => setOpen(!open)} className={`w-full flex items-center justify-between px-3 py-2 border rounded-md text-sm transition-colors ${open ? 'border-gray-400 ring-1 ring-gray-300' : 'border-border'} ${displayText ? 'text-gray-900' : 'text-gray-400'}`}>
        <span className="truncate">{displayText || 'Select Resource Set(s)...'}</span>
        <svg className={`w-3.5 h-3.5 shrink-0 text-gray-600 transition-transform ${open ? 'rotate-180' : ''}`} fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" /></svg>
      </button>
      {open && (
        <div className="absolute top-full left-0 right-0 mt-1 bg-white border border-border rounded-md shadow-lg z-50 max-h-48 overflow-y-auto">
          {options.map((opt) => (
            <button key={opt.value} type="button" onClick={() => { if (selected.includes(opt.value)) onChange(selected.filter((v) => v !== opt.value)); else onChange([...selected, opt.value]) }} className={`w-full text-left px-4 py-2.5 text-sm transition-colors ${selected.includes(opt.value) ? 'bg-gray-100 text-gray-700 font-medium' : 'text-gray-600 hover:bg-gray-50'}`}>
              {opt.label}
            </button>
          ))}
        </div>
      )}
      {selected.length > 0 && (
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

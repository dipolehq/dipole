import { useState, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { X } from 'lucide-react'
import { PageHeader } from '@/shared/components/PageHeader'
import { FloatingInput, FloatingTextarea, FloatingDropdown } from '@/shared/components/FloatingInput'
import { useClickOutside } from '@/shared/hooks/useClickOutside'
import { useCreateRole } from '@/features/administration/hooks/useRoles'
import { apiErrorMessage } from '@/features/administration/api/rolesApi'

const STATUS_OPTIONS = [
  { value: 'active', label: 'Active' },
  { value: 'inactive', label: 'Inactive' },
]

const RESOURCE_SET_OPTIONS = [
  { value: 'default', label: 'Default' },
  { value: 'production', label: 'Production' },
  { value: 'staging', label: 'Staging' },
]

const COPY_FROM_ROLE_OPTIONS = [
  { value: 'none', label: 'None' },
  { value: 'SuperAdmin', label: 'SuperAdmin' },
  { value: 'Auditor', label: 'Auditor' },
]

type PermissionLevel = 'no_access' | 'read_only' | 'read_write'

const PERMISSION_ENTITIES = [
  'Access Policies',
  'Client Workloads',
  'Trust Providers',
  'Access Conditions',
  'Integrations',
  'Credential Providers',
  'Server Workloads',
  'Agent Controllers',
  'Access Authorization Events',
  'Audit Logs',
  'Workload Events',
  'Users',
  'Roles',
  'Log Streams',
  'Identity Providers',
  'Resource Sets',
  'Global Policy Compliance',
] as const

export function CreateRolePage() {
  const navigate = useNavigate()
  const createMutation = useCreateRole()
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

  function handleCopyFromRole(roleName: string) {
    setCopyFromRole(roleName)
    if (roleName === 'SuperAdmin') {
      setPermissions(Object.fromEntries(PERMISSION_ENTITIES.map((e) => [e, 'read_write' as PermissionLevel])))
    } else if (roleName === 'Auditor') {
      setPermissions(Object.fromEntries(PERMISSION_ENTITIES.map((e) => [e, 'read_only' as PermissionLevel])))
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
      {error && (
        <div className="px-6 py-2 bg-red-50 border-b border-red-200 text-sm text-red-700 shrink-0">
          {error}
        </div>
      )}
      {/* Header bar */}
      <div className="flex items-center justify-between px-4 shrink-0 h-[60px] border-b border-border bg-white">
        <PageHeader />
        <div className="flex items-center gap-3">
          <button type="button" onClick={handleSave} disabled={createMutation.isPending} className="px-4 py-2 bg-gray-800 hover:bg-gray-900 text-white text-sm font-medium rounded-md transition-colors disabled:opacity-60">
            {createMutation.isPending ? 'Saving…' : 'Save'}
          </button>
          <button type="button" onClick={handleCancel} className="px-4 py-2 border border-gray-300 text-sm font-medium text-gray-600 hover:text-gray-900 hover:border-gray-400 rounded-md transition-colors">Cancel</button>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto bg-white">
        <div className="flex h-full">
          {/* Form — left 60% */}
          <div className="w-[60%] px-6 py-6 overflow-y-auto">
            <div className="space-y-5">
              <div className="border border-border rounded-lg p-5">
                <h4 className="text-sm font-semibold text-gray-800 mb-4">Role Details</h4>
                <div className="space-y-4">
                  <FloatingDropdown label="Status" options={STATUS_OPTIONS} value={status} onChange={setStatus} />
                  <FloatingInput label="Name" value={name} onChange={setName} required />
                  <FloatingTextarea label="Description (Optional)" value={description} onChange={setDescription} rows={2} />
                </div>
              </div>

              <div className="border border-border rounded-lg p-5">
                <h4 className="text-sm font-semibold text-gray-800 mb-4">Resource Set Assignments</h4>
                <ResourceSetMultiSelect options={RESOURCE_SET_OPTIONS} selected={resourceSets} onChange={setResourceSets} />
              </div>

              <div className="border border-border rounded-lg p-5">
                <h4 className="text-sm font-semibold text-gray-800 mb-4">Permissions</h4>
                <div className="mb-4">
                  <FloatingDropdown label="Copy from existing Role" options={COPY_FROM_ROLE_OPTIONS} value={copyFromRole} onChange={handleCopyFromRole} />
                </div>
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
                            <input type="checkbox" checked={permissions[entity] === 'no_access'} onChange={() => handlePermissionChange(entity, 'no_access')} className="w-3.5 h-3.5 accent-gray-600 cursor-pointer" />
                          </td>
                          <td className="px-4 py-3 text-center">
                            <input type="checkbox" checked={permissions[entity] === 'read_only'} onChange={() => handlePermissionChange(entity, 'read_only')} className="w-3.5 h-3.5 accent-gray-600 cursor-pointer" />
                          </td>
                          <td className="px-4 py-3 text-center">
                            <input type="checkbox" checked={permissions[entity] === 'read_write'} onChange={() => handlePermissionChange(entity, 'read_write')} className="w-3.5 h-3.5 accent-gray-600 cursor-pointer" />
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            </div>
          </div>

          {/* Help panel — right 40% */}
          <div className="w-[40%] px-6 py-6 overflow-y-auto">
            <div className="border border-border rounded-lg p-5 sticky top-0">
              <p className="text-xs text-gray-700 leading-relaxed mb-3">
                <span className="font-semibold text-gray-700">Roles</span> define what actions a user can perform across the Dipole platform. Each role is a collection of permissions scoped to specific entities.
              </p>
              <p className="text-xs text-gray-700 leading-relaxed mb-3">
                Use <span className="font-semibold">Resource Set Assignments</span> to limit the role&apos;s scope to specific resource groups, enabling multi-tenancy within your organization.
              </p>
              <p className="text-xs text-gray-700 leading-relaxed">
                You can <span className="font-semibold">copy permissions</span> from an existing role as a starting point and customize from there.
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

// --- Multi-select for Resource Sets ---

function ResourceSetMultiSelect({
  options,
  selected,
  onChange,
}: {
  options: { value: string; label: string }[]
  selected: string[]
  onChange: (selected: string[]) => void
}) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)
  useClickOutside(ref, () => setOpen(false))

  const displayText = selected.length > 0
    ? options.filter((o) => selected.includes(o.value)).map((o) => o.label).join(', ')
    : ''

  function handleToggle(value: string) {
    if (selected.includes(value)) {
      onChange(selected.filter((v) => v !== value))
    } else {
      onChange([...selected, value])
    }
  }

  function handleRemove(value: string) {
    onChange(selected.filter((v) => v !== value))
  }

  return (
    <div ref={ref} className="relative">
      <button
        type="button"
        onClick={() => setOpen(!open)}
        className={`w-full flex items-center justify-between px-3 py-2 border rounded-md text-sm transition-colors ${
          open ? 'border-gray-400 ring-2 ring-gray-200' : 'border-border'
        } ${displayText ? 'text-gray-900' : 'text-gray-400'}`}
      >
        <span className="truncate">{displayText || 'Select Resource Set(s)...'}</span>
        <svg className={`w-3.5 h-3.5 shrink-0 text-gray-600 transition-transform ${open ? 'rotate-180' : ''}`} fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
        </svg>
      </button>

      {open && (
        <div className="absolute top-full left-0 right-0 mt-1 bg-white border border-border rounded-md shadow-lg z-50 max-h-48 overflow-y-auto">
          {options.map((opt) => (
            <button
              key={opt.value}
              type="button"
              onClick={() => handleToggle(opt.value)}
              className={`w-full text-left px-4 py-2.5 text-sm transition-colors ${
                selected.includes(opt.value) ? 'bg-gray-100 text-gray-700 font-medium' : 'text-gray-600 hover:bg-gray-50'
              }`}
            >
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
                <button type="button" onClick={() => handleRemove(value)} className="text-gray-500 hover:text-gray-700 transition-colors">
                  <X size={14} />
                </button>
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}

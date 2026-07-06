import { useState, useRef } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { FileText, X } from 'lucide-react'
import { PageHeader } from '@/shared/components/PageHeader'
import { FloatingInput, FloatingTextarea, FloatingDropdown } from '@/shared/components/FloatingInput'
import { useClickOutside } from '@/shared/hooks/useClickOutside'
import { rolesData } from '@/data/administration/roles'

const STATUS_OPTIONS = [
  { value: 'active', label: 'Active' },
  { value: 'inactive', label: 'Inactive' },
]

const RESOURCE_SET_OPTIONS = [
  { value: 'default', label: 'Default' },
  { value: 'production', label: 'Production' },
  { value: 'staging', label: 'Staging' },
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

const DETAIL_SECTIONS = [
  { value: 'details', label: 'Details' },
  { value: 'metrics', label: 'Metrics' },
  { value: 'logs', label: 'Logs' },
] as const

type DetailSection = (typeof DETAIL_SECTIONS)[number]['value']

export function RoleDetailPage() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const role = rolesData.find((r) => r.id === id)

  const [selectedSection, setSelectedSection] = useState<DetailSection>('details')
  const [sectionOpen, setSectionOpen] = useState(false)
  const [isEditing, setIsEditing] = useState(false)
  const [status, setStatus] = useState(role?.isActive ? 'active' : 'inactive')
  const [name, setName] = useState(role?.name ?? '')
  const [description, setDescription] = useState(role?.description ?? '')
  const [resourceSets, setResourceSets] = useState<string[]>(['default'])
  const [permissions, setPermissions] = useState<Record<string, PermissionLevel>>(
    Object.fromEntries(PERMISSION_ENTITIES.map((e) => [e, 'no_access' as PermissionLevel]))
  )

  if (!role) {
    return (
      <div className="flex flex-col h-full">
        <div className="flex items-center justify-between px-4 py-3 shrink-0 min-h-[36px] border-b border-border bg-white"><PageHeader /></div>
        <div className="flex-1 flex items-center justify-center flex-col">
          <p className="text-sm text-gray-500">Role not found.</p>
          <button onClick={() => navigate('/administration')} className="mt-4 text-xs text-orange-500 hover:text-orange-600 font-medium">← Back to Administration</button>
        </div>
      </div>
    )
  }

  const disabled = !isEditing
  const selectedLabel = DETAIL_SECTIONS.find((s) => s.value === selectedSection)?.label ?? ''

  function handlePermissionChange(entity: string, level: PermissionLevel) {
    if (disabled) return
    setPermissions({ ...permissions, [entity]: level })
  }

  function handleEditToggle() {
    if (isEditing) {
      console.info('[save-role]', { name, description, status, resourceSets, permissions })
      setIsEditing(false)
    } else {
      setIsEditing(true)
    }
  }

  function handleCancel() {
    if (isEditing) {
      setName(role?.name ?? '')
      setDescription(role?.description ?? '')
      setStatus(role?.isActive ? 'active' : 'inactive')
      setIsEditing(false)
    } else {
      navigate('/administration')
    }
  }

  return (
    <div className="flex flex-col h-full">
      {/* Header bar */}
      <div className="flex items-center justify-between px-4 py-3 shrink-0 min-h-[36px] border-b border-border bg-white">
        <PageHeader />
        <div className="flex items-center gap-3">
          <button type="button" onClick={handleEditToggle} className="px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white text-sm font-medium rounded-md transition-colors">{isEditing ? 'Save' : 'Edit'}</button>
          <button type="button" onClick={handleCancel} className="px-4 py-2 border border-gray-300 text-sm font-medium text-gray-600 hover:text-gray-900 hover:border-gray-400 rounded-md transition-colors">{isEditing ? 'Cancel' : 'Back'}</button>
          <div className="relative">
            <button type="button" onClick={() => setSectionOpen(!sectionOpen)} className="flex items-center gap-1.5 w-[160px] px-4 py-2 bg-[#28282B] hover:bg-[#333336] text-white text-sm font-medium rounded-md transition-colors">
              <FileText size={14} className="shrink-0" />
              <span className="truncate">{selectedLabel}</span>
              <svg className={`w-3.5 h-3.5 ml-auto shrink-0 transition-transform ${sectionOpen ? 'rotate-180' : ''}`} fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" /></svg>
            </button>
            {sectionOpen && (
              <div className="absolute top-full right-0 mt-1 bg-white border border-border rounded-md shadow-lg z-50 w-[180px] overflow-hidden">
                {DETAIL_SECTIONS.map((opt) => (
                  <button key={opt.value} type="button" onClick={() => { setSelectedSection(opt.value); setSectionOpen(false) }} className={`w-full text-left px-4 py-2.5 text-sm transition-colors ${selectedSection === opt.value ? 'bg-orange-50 text-orange-600 font-medium' : 'text-gray-600 hover:bg-gray-50'}`}>{opt.label}</button>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto bg-white">
        {selectedSection === 'details' && (
          <div className="flex h-full">
            {/* Form — left 60% */}
            <div className="w-[60%] px-6 py-6 overflow-y-auto">
              <div className={`space-y-5 ${disabled ? 'opacity-60 pointer-events-none' : ''}`}>
                <div className="border border-border rounded-lg p-5">
                  <h4 className="text-sm font-semibold text-gray-800 mb-4">Role Details</h4>
                  <div className="space-y-4">
                    <FloatingDropdown label="Status" options={STATUS_OPTIONS} value={status} onChange={setStatus} disabled={disabled} />
                    <FloatingInput label="Name" value={name} onChange={setName} disabled={disabled} required />
                    <FloatingTextarea label="Description (Optional)" value={description} onChange={setDescription} disabled={disabled} rows={2} />
                  </div>
                </div>

                <div className="border border-border rounded-lg p-5">
                  <h4 className="text-sm font-semibold text-gray-800 mb-4">Resource Set Assignments</h4>
                  <ResourceSetMultiSelect options={RESOURCE_SET_OPTIONS} selected={resourceSets} onChange={setResourceSets} disabled={disabled} />
                </div>

                <div className="border border-border rounded-lg p-5">
                  <h4 className="text-sm font-semibold text-gray-800 mb-4">Permissions</h4>
                  <div className="border border-border rounded-lg overflow-hidden">
                    <table className="w-full text-xs">
                      <thead>
                        <tr className="bg-gray-50 border-b border-border">
                          <th className="px-4 py-3 text-[11px] font-bold text-orange-600 uppercase tracking-wider text-left">Entity</th>
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
                              <input type="checkbox" checked={permissions[entity] === 'no_access'} onChange={() => handlePermissionChange(entity, 'no_access')} disabled={disabled} className="w-3.5 h-3.5 accent-orange-500 cursor-pointer disabled:cursor-default disabled:opacity-60" />
                            </td>
                            <td className="px-4 py-3 text-center">
                              <input type="checkbox" checked={permissions[entity] === 'read_only'} onChange={() => handlePermissionChange(entity, 'read_only')} disabled={disabled} className="w-3.5 h-3.5 accent-orange-500 cursor-pointer disabled:cursor-default disabled:opacity-60" />
                            </td>
                            <td className="px-4 py-3 text-center">
                              <input type="checkbox" checked={permissions[entity] === 'read_write'} onChange={() => handlePermissionChange(entity, 'read_write')} disabled={disabled} className="w-3.5 h-3.5 accent-orange-500 cursor-pointer disabled:cursor-default disabled:opacity-60" />
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </div>
              </div>
            </div>

            {/* Right panel — 40% */}
            <div className="w-[40%] px-6 py-6 overflow-y-auto">
              <div className="border border-border rounded-lg p-5 mb-5">
                <h4 className="text-sm font-semibold text-gray-800 mb-4">Metadata</h4>
                <div className="space-y-4">
                  <FloatingInput label="ID" value={role.id} disabled />
                  <FloatingInput label="Users" value={String(role.users)} disabled />
                </div>
              </div>

              <div className="border border-border rounded-lg p-5">
                <p className="text-xs text-gray-700 leading-relaxed mb-3">
                  <span className="font-semibold text-orange-600">Roles</span> define what actions a user can perform across the Dipole platform. Each role is a collection of permissions scoped to specific entities.
                </p>
                <p className="text-xs text-gray-700 leading-relaxed">
                  Use <span className="font-semibold">Resource Set Assignments</span> to limit the role&apos;s scope to specific resource groups.
                </p>
              </div>
            </div>
          </div>
        )}

        {selectedSection === 'metrics' && (
          <div className="flex items-center justify-center h-full"><p className="text-sm text-gray-400">Metrics — coming soon</p></div>
        )}
        {selectedSection === 'logs' && (
          <div className="flex items-center justify-center h-full"><p className="text-sm text-gray-400">Logs — coming soon</p></div>
        )}
      </div>
    </div>
  )
}

// --- Multi-select for Resource Sets ---

function ResourceSetMultiSelect({
  options,
  selected,
  onChange,
  disabled = false,
}: {
  options: { value: string; label: string }[]
  selected: string[]
  onChange: (selected: string[]) => void
  disabled?: boolean
}) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)
  useClickOutside(ref, () => setOpen(false))

  const displayText = selected.length > 0
    ? options.filter((o) => selected.includes(o.value)).map((o) => o.label).join(', ')
    : ''

  function handleToggle(value: string) {
    if (disabled) return
    if (selected.includes(value)) {
      onChange(selected.filter((v) => v !== value))
    } else {
      onChange([...selected, value])
    }
  }

  function handleRemove(value: string) {
    if (disabled) return
    onChange(selected.filter((v) => v !== value))
  }

  return (
    <div ref={ref} className="relative">
      <button
        type="button"
        onClick={() => !disabled && setOpen(!open)}
        disabled={disabled}
        className={`w-full flex items-center justify-between px-3 py-2 border rounded-md text-sm transition-colors ${
          open ? 'border-orange-400 ring-2 ring-orange-100' : 'border-border'
        } ${displayText ? 'text-gray-900' : 'text-gray-400'} ${disabled ? 'bg-gray-50 cursor-default' : ''}`}
      >
        <span className="truncate">{displayText || 'Select Resource Set(s)...'}</span>
        <svg className={`w-3.5 h-3.5 shrink-0 text-orange-500 transition-transform ${open ? 'rotate-180' : ''}`} fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
        </svg>
      </button>

      {open && !disabled && (
        <div className="absolute top-full left-0 right-0 mt-1 bg-white border border-border rounded-md shadow-lg z-50 max-h-48 overflow-y-auto">
          {options.map((opt) => (
            <button key={opt.value} type="button" onClick={() => handleToggle(opt.value)} className={`w-full text-left px-4 py-2.5 text-sm transition-colors ${selected.includes(opt.value) ? 'bg-gray-100 text-orange-600 font-medium' : 'text-gray-600 hover:bg-gray-50'}`}>
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
                <button type="button" onClick={() => handleRemove(value)} className="text-orange-400 hover:text-orange-600 transition-colors">
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

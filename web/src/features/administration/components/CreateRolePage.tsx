import { useState, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { X } from 'lucide-react'
import { PageHeader } from '@/shared/components/PageHeader'
import { Dropdown } from '@/shared/components/Dropdown'
import { useClickOutside } from '@/shared/hooks/useClickOutside'

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
  { value: '', label: 'Copy from existing Role...' },
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

type TabKey = 'details' | 'permissions'

export function CreateRolePage() {
  const navigate = useNavigate()
  const [activeTab, setActiveTab] = useState<TabKey>('details')
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

  function handleSave() {
    console.info('[create-role] submitted', { name, description, status, resourceSets, permissions })
    navigate('/administration')
  }

  function handleCancel() {
    navigate('/administration')
  }

  return (
    <div className="px-4 py-3 flex flex-col h-full">
      <div className="flex items-center justify-between mb-4 shrink-0 min-h-[36px]">
        <PageHeader />
      </div>

      <div className="bg-white rounded-lg border border-border flex-1 flex min-h-0">
        {/* Left sidebar */}
        <nav className="w-48 border-r border-border py-3 shrink-0 flex flex-col">
          <div className="flex-1">
            <button
              type="button"
              onClick={() => setActiveTab('details')}
              className={`w-full text-left px-5 py-2.5 text-xs font-medium transition-colors relative ${
                activeTab === 'details' ? 'text-orange-600 bg-orange-50' : 'text-gray-500 hover:text-gray-800 hover:bg-gray-50'
              }`}
            >
              {activeTab === 'details' && <span className="absolute left-0 top-0 bottom-0 w-0.5 bg-orange-500 rounded-r" />}
              <span>Role Details</span>
            </button>
            <button
              type="button"
              onClick={() => setActiveTab('permissions')}
              className={`w-full text-left px-5 py-2.5 text-xs font-medium transition-colors relative ${
                activeTab === 'permissions' ? 'text-orange-600 bg-orange-50' : 'text-gray-500 hover:text-gray-800 hover:bg-gray-50'
              }`}
            >
              {activeTab === 'permissions' && <span className="absolute left-0 top-0 bottom-0 w-0.5 bg-orange-500 rounded-r" />}
              <span>Role Permissions</span>
            </button>
          </div>
          <div className="px-4 py-3 border-t border-border space-y-2">
            <button
              type="button"
              onClick={handleSave}
              className="w-full px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors"
            >
              Save Role
            </button>
            <button
              type="button"
              onClick={handleCancel}
              className="w-full px-4 py-2 border border-gray-300 text-xs font-medium text-gray-600 hover:text-gray-900 hover:border-gray-400 rounded-md transition-colors"
            >
              Cancel
            </button>
          </div>
        </nav>

        {/* Main content */}
        <div className="flex-1 overflow-y-auto p-6">
          {activeTab === 'details' ? (
            <div>
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-sm font-semibold text-gray-900">User Roles</h3>
                <button
                  type="button"
                  onClick={handleSave}
                  className="px-4 py-1.5 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors"
                >
                  Save
                </button>
              </div>
              <div className="border-b border-border mb-6" />

              {/* Status */}
              <div className="mb-6">
                <label className="block text-xs font-medium text-gray-700 mb-1.5">Status</label>
                <Dropdown
                  options={STATUS_OPTIONS}
                  value={status}
                  onChange={setStatus}
                  placeholder="Select status..."
                />
              </div>

              {/* Name */}
              <div className="mb-4">
                <label className="block text-xs text-gray-500 mb-1">
                  Name <span className="text-red-500">*</span>
                </label>
                <input
                  type="text"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="Name"
                  className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400"
                />
              </div>

              {/* Description */}
              <div className="mb-6">
                <label className="block text-xs text-gray-500 mb-1">Description (Optional)</label>
                <textarea
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="Description (Optional)"
                  rows={3}
                  className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400 resize-y"
                />
              </div>

              {/* Resource Set Assignments */}
              <h4 className="text-[11px] font-bold text-gray-600 uppercase tracking-wider mb-4">Resource Set(s) Assignments</h4>
              <div className="mb-8">
                <ResourceSetMultiSelect
                  options={RESOURCE_SET_OPTIONS}
                  selected={resourceSets}
                  onChange={setResourceSets}
                />
              </div>
            </div>
          ) : (
            <div>
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-sm font-semibold text-gray-900">Permissions</h3>
                <button
                  type="button"
                  onClick={handleSave}
                  className="px-4 py-1.5 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors"
                >
                  Save
                </button>
              </div>
              <div className="border-b border-border mb-6" />

              {/* Copy from existing role */}
              <div className="mb-6">
                <Dropdown
                  options={COPY_FROM_ROLE_OPTIONS}
                  value={copyFromRole}
                  onChange={handleCopyFromRole}
                  placeholder="Copy from existing Role..."
                />
              </div>

              {/* Permissions table */}
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
                          <input
                            type="checkbox"
                            checked={permissions[entity] === 'no_access'}
                            onChange={() => handlePermissionChange(entity, 'no_access')}
                            className="w-3.5 h-3.5 accent-orange-500 cursor-pointer"
                          />
                        </td>
                        <td className="px-4 py-3 text-center">
                          <input
                            type="checkbox"
                            checked={permissions[entity] === 'read_only'}
                            onChange={() => handlePermissionChange(entity, 'read_only')}
                            className="w-3.5 h-3.5 accent-orange-500 cursor-pointer"
                          />
                        </td>
                        <td className="px-4 py-3 text-center">
                          <input
                            type="checkbox"
                            checked={permissions[entity] === 'read_write'}
                            onChange={() => handlePermissionChange(entity, 'read_write')}
                            className="w-3.5 h-3.5 accent-orange-500 cursor-pointer"
                          />
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}
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
          open ? 'border-orange-400 ring-2 ring-orange-100' : 'border-border'
        } ${displayText ? 'text-gray-900' : 'text-gray-400'}`}
      >
        <span className="truncate">{displayText || 'Select Resource Set(s)...'}</span>
        <svg className={`w-3.5 h-3.5 shrink-0 text-orange-500 transition-transform ${open ? 'rotate-180' : ''}`} fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
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
                selected.includes(opt.value) ? 'bg-gray-100 text-orange-600 font-medium' : 'text-gray-600 hover:bg-gray-50'
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
                <button
                  type="button"
                  onClick={() => handleRemove(value)}
                  className="text-orange-400 hover:text-orange-600 transition-colors"
                >
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

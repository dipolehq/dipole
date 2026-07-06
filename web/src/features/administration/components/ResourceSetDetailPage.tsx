import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { FileText, X } from 'lucide-react'
import { PageHeader } from '@/shared/components/PageHeader'
import { FloatingInput, FloatingTextarea, FloatingDropdown } from '@/shared/components/FloatingInput'
import { resourceSetsData } from '@/data/administration/resourceSets'

const AVAILABLE_ROLES = [
  { value: 'SuperAdmin', label: 'SuperAdmin' },
  { value: 'Auditor', label: 'Auditor' },
  { value: 'Team Lead', label: 'Team Lead' },
  { value: 'Editor', label: 'Editor' },
  { value: 'Operator', label: 'Operator' },
]

const DETAIL_SECTIONS = [
  { value: 'details', label: 'Details' },
  { value: 'metrics', label: 'Metrics' },
  { value: 'logs', label: 'Logs' },
] as const

type DetailSection = (typeof DETAIL_SECTIONS)[number]['value']

export function ResourceSetDetailPage() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const resourceSet = resourceSetsData.find((rs) => rs.id === id)

  const [selectedSection, setSelectedSection] = useState<DetailSection>('details')
  const [sectionOpen, setSectionOpen] = useState(false)
  const [isEditing, setIsEditing] = useState(false)
  const [displayName, setDisplayName] = useState(resourceSet?.name ?? '')
  const [description, setDescription] = useState('')
  const [selectedRole, setSelectedRole] = useState('')
  const [assignedRoles, setAssignedRoles] = useState<string[]>([])

  if (!resourceSet) {
    return (
      <div className="flex flex-col h-full">
        <div className="flex items-center justify-between px-4 py-3 shrink-0 border-b border-border bg-white">
          <PageHeader />
        </div>
        <div className="flex-1 flex items-center justify-center flex-col">
          <p className="text-sm text-gray-500">Resource Set not found.</p>
          <button onClick={() => navigate('/administration')} className="mt-4 text-xs text-orange-500 hover:text-orange-600 font-medium">← Back to Administration</button>
        </div>
      </div>
    )
  }

  const disabled = !isEditing
  const selectedLabel = DETAIL_SECTIONS.find((s) => s.value === selectedSection)?.label ?? ''

  function handleAddRole() {
    if (selectedRole && !assignedRoles.includes(selectedRole)) {
      setAssignedRoles([...assignedRoles, selectedRole])
      setSelectedRole('')
    }
  }

  function handleRemoveRole(role: string) {
    setAssignedRoles(assignedRoles.filter((r) => r !== role))
  }

  function handleEditToggle() {
    if (isEditing) {
      console.info('[save-resource-set]', { displayName, description, assignedRoles })
      setIsEditing(false)
    } else {
      setIsEditing(true)
    }
  }

  function handleCancel() {
    if (isEditing) {
      setDisplayName(resourceSet?.name ?? '')
      setDescription('')
      setAssignedRoles([])
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
                  <h4 className="text-sm font-semibold text-gray-800 mb-4">Resource Set Details</h4>
                  <div className="space-y-4">
                    <FloatingInput label="Display Name" value={displayName} onChange={setDisplayName} disabled={disabled} required />
                    <FloatingTextarea label="Description (Optional)" value={description} onChange={setDescription} disabled={disabled} rows={3} />
                  </div>
                </div>

                <div className="border border-border rounded-lg p-5">
                  <h4 className="text-sm font-semibold text-gray-800 mb-4">Role Assignments</h4>
                  <div className="flex items-end gap-2 mb-3">
                    <div className="flex-1">
                      <FloatingDropdown label="Select Role" options={AVAILABLE_ROLES} value={selectedRole} onChange={setSelectedRole} disabled={disabled} />
                    </div>
                    <button type="button" onClick={handleAddRole} disabled={!selectedRole || disabled} className="px-4 py-2.5 bg-orange-500 hover:bg-orange-600 text-white text-sm font-medium rounded-md transition-colors disabled:opacity-40 disabled:cursor-not-allowed">Add</button>
                  </div>
                  {assignedRoles.length > 0 && (
                    <div className="border border-border rounded-md divide-y divide-border">
                      {assignedRoles.map((role) => (
                        <div key={role} className="flex items-center justify-between px-3 py-2">
                          <span className="text-sm text-gray-700">{role}</span>
                          {!disabled && (
                            <button type="button" onClick={() => handleRemoveRole(role)} className="text-orange-400 hover:text-orange-600 transition-colors">
                              <X size={14} />
                            </button>
                          )}
                        </div>
                      ))}
                    </div>
                  )}
                  {assignedRoles.length === 0 && (
                    <p className="text-xs text-gray-400">No roles assigned</p>
                  )}
                </div>
              </div>
            </div>

            {/* Right panel — 40% */}
            <div className="w-[40%] px-6 py-6 overflow-y-auto">
              <div className="border border-border rounded-lg p-5 mb-5">
                <h4 className="text-sm font-semibold text-gray-800 mb-4">Metadata</h4>
                <div className="space-y-4">
                  <FloatingInput label="ID" value={resourceSet.id} disabled />
                  <FloatingInput label="Roles" value={String(resourceSet.roles)} disabled />
                  <FloatingInput label="Users" value={String(resourceSet.users)} disabled />
                </div>
              </div>

              <div className="border border-border rounded-lg p-5">
                <p className="text-xs text-gray-700 leading-relaxed mb-3">
                  <span className="font-semibold text-orange-600">Resource Sets</span> provide multi-tenancy within your organization. Each Resource Set is an isolated group of resources.
                </p>
                <p className="text-xs text-gray-700 leading-relaxed">
                  Assign <span className="font-semibold">Roles</span> to control which users can manage resources within this Resource Set.
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

import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { X } from 'lucide-react'
import { PageHeader } from '@/shared/components/PageHeader'
import { Dropdown } from '@/shared/components/Dropdown'
import { resourceSetsData } from '@/data/administration/resourceSets'

const AVAILABLE_ROLES = [
  { value: 'SuperAdmin', label: 'SuperAdmin' },
  { value: 'Auditor', label: 'Auditor' },
  { value: 'Team Lead', label: 'Team Lead' },
  { value: 'Editor', label: 'Editor' },
  { value: 'Operator', label: 'Operator' },
]

type TabKey = 'details' | 'roles'

export function ResourceSetDetailPage() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()

  const resourceSet = resourceSetsData.find((rs) => rs.id === id)

  const [isEditing, setIsEditing] = useState(false)
  const [activeTab, setActiveTab] = useState<TabKey>('details')
  const [displayName, setDisplayName] = useState(resourceSet?.name ?? '')
  const [description, setDescription] = useState('')
  const [roleMode, setRoleMode] = useState<'add_new' | 'select_existing'>('add_new')
  const [createNewAdmin, setCreateNewAdmin] = useState(false)
  const [newRoleName, setNewRoleName] = useState('')
  const [selectedRole, setSelectedRole] = useState('')
  const [assignedRoles, setAssignedRoles] = useState<string[]>([])

  if (!resourceSet) {
    return (
      <div className="p-6">
        <PageHeader />
        <p className="mt-6 text-sm text-gray-500">Resource Set not found.</p>
        <button
          onClick={() => navigate('/administration')}
          className="mt-4 text-xs text-orange-500 hover:text-orange-600 font-medium"
        >
          ← Back to Administration
        </button>
      </div>
    )
  }

  const disabled = !isEditing

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
      console.info('[save-resource-set]', { displayName, description, assignedRoles, newRoleName })
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
      setNewRoleName('')
      setCreateNewAdmin(false)
      setIsEditing(false)
    } else {
      navigate('/administration')
    }
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
              <span>Resource Details</span>
            </button>
            <button
              type="button"
              onClick={() => setActiveTab('roles')}
              className={`w-full text-left px-5 py-2.5 text-xs font-medium transition-colors relative ${
                activeTab === 'roles' ? 'text-orange-600 bg-orange-50' : 'text-gray-500 hover:text-gray-800 hover:bg-gray-50'
              }`}
            >
              {activeTab === 'roles' && <span className="absolute left-0 top-0 bottom-0 w-0.5 bg-orange-500 rounded-r" />}
              <span>Resource Roles</span>
            </button>
          </div>
          <div className="px-4 py-3 border-t border-border space-y-2">
            <button
              type="button"
              onClick={handleEditToggle}
              className="w-full px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors"
            >
              {isEditing ? 'Save Resource Set' : 'Edit Resource Set'}
            </button>
            <button
              type="button"
              onClick={handleCancel}
              className="w-full px-4 py-2 border border-gray-300 text-xs font-medium text-gray-600 hover:text-gray-900 hover:border-gray-400 rounded-md transition-colors"
            >
              {isEditing ? 'Cancel' : 'Back'}
            </button>
          </div>
        </nav>

        {/* Main content */}
        <div className="flex-1 overflow-y-auto p-6">
          {activeTab === 'details' ? (
            <div>
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-sm font-semibold text-gray-900">Resource Set</h3>
                {isEditing && (
                  <button
                    type="button"
                    onClick={handleEditToggle}
                    className="px-4 py-1.5 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors"
                  >
                    Save
                  </button>
                )}
              </div>
              <div className="border-b border-border mb-6" />

              <div className="space-y-4">
                <div>
                  <label className="block text-xs text-gray-500 mb-1">Display Name</label>
                  <input
                    type="text"
                    value={displayName}
                    onChange={(e) => setDisplayName(e.target.value)}
                    disabled={disabled}
                    placeholder="Display Name"
                    className={`w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400 ${disabled ? 'bg-gray-50 text-gray-600' : ''}`}
                  />
                </div>
                <div>
                  <label className="block text-xs text-gray-500 mb-1">Description (Optional)</label>
                  <textarea
                    value={description}
                    onChange={(e) => setDescription(e.target.value)}
                    disabled={disabled}
                    placeholder="Description (Optional)"
                    rows={3}
                    className={`w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400 resize-y ${disabled ? 'bg-gray-50 text-gray-600' : ''}`}
                  />
                </div>
                <div>
                  <label className="block text-xs text-gray-500 mb-1">Resource Set ID</label>
                  <div className="px-3 py-2 border border-border rounded-md text-sm text-gray-500 bg-gray-50">{resourceSet.id}</div>
                </div>
              </div>
            </div>
          ) : (
            <div>
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-sm font-semibold text-gray-900">Role Assignments</h3>
                {isEditing && (
                  <button
                    type="button"
                    onClick={handleEditToggle}
                    className="px-4 py-1.5 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors"
                  >
                    Save
                  </button>
                )}
              </div>
              <div className="border-b border-border mb-6" />

              {disabled ? (
                <div className="px-3 py-2 border border-border rounded-md text-sm text-gray-600 bg-gray-50">
                  {assignedRoles.length > 0 ? assignedRoles.join(', ') : 'No roles assigned'}
                </div>
              ) : (
                <div className="border border-border rounded-lg p-5">
                  {/* Add New / Select Existing toggle */}
                  <div className="flex items-center gap-2 mb-5">
                    <button
                      type="button"
                      onClick={() => setRoleMode('add_new')}
                      className={`px-3 py-1 text-xs font-medium rounded-md transition-colors ${
                        roleMode === 'add_new' ? 'bg-orange-500 text-white' : 'text-orange-500 hover:bg-orange-50'
                      }`}
                    >
                      Add New
                    </button>
                    <button
                      type="button"
                      onClick={() => setRoleMode('select_existing')}
                      className={`px-3 py-1 text-xs font-medium rounded-md transition-colors ${
                        roleMode === 'select_existing' ? 'bg-orange-500 text-white' : 'text-orange-500 hover:bg-orange-50'
                      }`}
                    >
                      Select Existing
                    </button>
                  </div>

                  {roleMode === 'add_new' ? (
                    <div className="space-y-4">
                      <div className="flex items-center gap-2">
                        <input
                          type="checkbox"
                          checked={createNewAdmin}
                          onChange={(e) => setCreateNewAdmin(e.target.checked)}
                          className="w-4 h-4 accent-orange-500 cursor-pointer"
                        />
                        <span className="text-sm text-gray-700">Create New Admin</span>
                      </div>
                      {createNewAdmin && (
                        <div>
                          <label className="block text-xs text-gray-500 mb-1">Display Name</label>
                          <input
                            type="text"
                            value={newRoleName}
                            onChange={(e) => setNewRoleName(e.target.value)}
                            placeholder="Display Name"
                            className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400"
                          />
                        </div>
                      )}
                    </div>
                  ) : (
                    <div className="space-y-3">
                      <div className="flex items-center gap-2">
                        <div className="flex-1">
                          <Dropdown
                            options={AVAILABLE_ROLES}
                            value={selectedRole}
                            onChange={setSelectedRole}
                            placeholder="Select an existing role..."
                          />
                        </div>
                        <button
                          type="button"
                          onClick={handleAddRole}
                          disabled={!selectedRole}
                          className="px-3 py-2 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                        >
                          Add
                        </button>
                      </div>

                      {assignedRoles.length > 0 && (
                        <div className="border border-border rounded-md divide-y divide-border">
                          {assignedRoles.map((role) => (
                            <div key={role} className="flex items-center justify-between px-3 py-2">
                              <span className="text-sm text-gray-700">{role}</span>
                              <button
                                type="button"
                                onClick={() => handleRemoveRole(role)}
                                className="text-orange-400 hover:text-orange-600 transition-colors"
                              >
                                <X size={14} />
                              </button>
                            </div>
                          ))}
                        </div>
                      )}
                    </div>
                  )}
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

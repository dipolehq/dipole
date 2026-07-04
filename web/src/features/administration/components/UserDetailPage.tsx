import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { PageHeader } from '@/shared/components/PageHeader'
import { Dropdown } from '@/shared/components/Dropdown'
import { usersData } from '@/data/administration/users'

const ROLE_OPTIONS = [
  { value: 'SuperAdmin', label: 'SuperAdmin' },
  { value: 'Auditor', label: 'Auditor' },
]

const STATUS_OPTIONS = [
  { value: 'enabled', label: 'Enabled' },
  { value: 'disabled', label: 'Disabled' },
]

const COUNTRY_CODE_OPTIONS = [
  { value: '+91', label: '🇮🇳 +91' },
  { value: '+1', label: '🇺🇸 +1' },
  { value: '+44', label: '🇬🇧 +44' },
  { value: '+61', label: '🇦🇺 +61' },
  { value: '+49', label: '🇩🇪 +49' },
  { value: '+81', label: '🇯🇵 +81' },
  { value: '+65', label: '🇸🇬 +65' },
]

export function UserDetailPage() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()

  const user = usersData.find((u) => u.id === id)

  const [isEditing, setIsEditing] = useState(false)
  const [activeTab, setActiveTab] = useState<'details' | 'configurations'>('details')
  const [firstName, setFirstName] = useState(user?.firstName ?? '')
  const [lastName, setLastName] = useState(user?.lastName ?? '')
  const [email, setEmail] = useState(user?.email ?? '')
  const [countryCode, setCountryCode] = useState('+91')
  const [phone, setPhone] = useState('')
  const [role, setRole] = useState(user?.role ?? '')
  const [status, setStatus] = useState(user?.isActive ? 'enabled' : 'disabled')

  if (!user) {
    return (
      <div className="p-6">
        <PageHeader />
        <p className="mt-6 text-sm text-gray-500">User not found.</p>
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

  function handleEditToggle() {
    if (isEditing) {
      console.info('[save-user]', { firstName, lastName, email, countryCode, phone, role, status })
      setIsEditing(false)
    } else {
      setIsEditing(true)
    }
  }

  function handleCancel() {
    if (isEditing) {
      setFirstName(user?.firstName ?? '')
      setLastName(user?.lastName ?? '')
      setEmail(user?.email ?? '')
      setRole(user?.role ?? '')
      setStatus(user?.isActive ? 'enabled' : 'disabled')
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
              <span>User Details</span>
            </button>
            <button
              type="button"
              onClick={() => setActiveTab('configurations')}
              className={`w-full text-left px-5 py-2.5 text-xs font-medium transition-colors relative ${
                activeTab === 'configurations' ? 'text-orange-600 bg-orange-50' : 'text-gray-500 hover:text-gray-800 hover:bg-gray-50'
              }`}
            >
              {activeTab === 'configurations' && <span className="absolute left-0 top-0 bottom-0 w-0.5 bg-orange-500 rounded-r" />}
              <span>Configurations</span>
            </button>
          </div>
          <div className="px-4 py-3 border-t border-border space-y-2">
            <button
              type="button"
              onClick={handleEditToggle}
              className="w-full px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors"
            >
              {isEditing ? 'Save User' : 'Edit User'}
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
            {/* Header */}
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-sm font-semibold text-gray-900">Dipole User</h3>
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

            {/* Status */}
            <div className="mb-6">
              <label className="block text-xs font-medium text-gray-700 mb-1.5">Status</label>
              {disabled ? (
                <div className="px-3 py-2 border border-border rounded-md text-sm text-gray-600 bg-gray-50 capitalize">{status}</div>
              ) : (
                <Dropdown
                  options={STATUS_OPTIONS}
                  value={status}
                  onChange={setStatus}
                  placeholder="Select status..."
                />
              )}
            </div>

            {/* User Details */}
            <h4 className="text-[11px] font-bold text-gray-600 uppercase tracking-wider mb-4">User Details</h4>

            <div className="space-y-4 mb-8">
              <div>
                <label className="block text-xs text-gray-500 mb-1">First Name</label>
                <input
                  type="text"
                  value={firstName}
                  onChange={(e) => setFirstName(e.target.value)}
                  disabled={disabled}
                  placeholder="John"
                  className={`w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400 ${disabled ? 'bg-gray-50 text-gray-600' : ''}`}
                />
              </div>
              <div>
                <label className="block text-xs text-gray-500 mb-1">Last Name</label>
                <input
                  type="text"
                  value={lastName}
                  onChange={(e) => setLastName(e.target.value)}
                  disabled={disabled}
                  placeholder="Smith"
                  className={`w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400 ${disabled ? 'bg-gray-50 text-gray-600' : ''}`}
                />
              </div>
              <div>
                <label className="block text-xs text-gray-500 mb-1">Email</label>
                <input
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  disabled={disabled}
                  placeholder="john.smith@company.com"
                  className={`w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400 ${disabled ? 'bg-gray-50 text-gray-600' : ''}`}
                />
              </div>
              <div className="flex gap-3">
                <div className="w-32">
                  <label className="block text-xs text-gray-500 mb-1">Country Code</label>
                  {disabled ? (
                    <div className="px-3 py-2 border border-border rounded-md text-sm text-gray-600 bg-gray-50">{countryCode}</div>
                  ) : (
                    <Dropdown
                      options={COUNTRY_CODE_OPTIONS}
                      value={countryCode}
                      onChange={setCountryCode}
                      placeholder="+1"
                    />
                  )}
                </div>
                <div className="flex-1">
                  <label className="block text-xs text-gray-500 mb-1">Phone Number</label>
                  <input
                    type="tel"
                    value={phone}
                    onChange={(e) => setPhone(e.target.value)}
                    disabled={disabled}
                    placeholder="3105551212"
                    className={`w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400 ${disabled ? 'bg-gray-50 text-gray-600' : ''}`}
                  />
                </div>
              </div>
            </div>

            {/* Role Assignment */}
            <h4 className="text-[11px] font-bold text-gray-600 uppercase tracking-wider mb-4">Role Assignment</h4>

            <div className="mb-8">
              {disabled ? (
                <div className="px-3 py-2 border border-border rounded-md text-sm text-gray-600 bg-gray-50">{role}</div>
              ) : (
                <Dropdown
                  options={ROLE_OPTIONS}
                  value={role}
                  onChange={setRole}
                  placeholder="Select a role..."
                />
              )}
            </div>
          </div>
          ) : (
            <div>
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-sm font-semibold text-gray-900">Configurations</h3>
              </div>
              <div className="border-b border-border mb-6" />

              {/* MFA Status */}
              <div className="border border-border rounded-lg p-5 mb-6">
                <div className="flex items-center justify-between mb-4">
                  <span className="text-sm font-medium text-gray-700">MFA Status</span>
                  <button
                    type="button"
                    className="px-4 py-1.5 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors"
                  >
                    Reset MFA
                  </button>
                </div>
                <div className="border-b border-orange-400 mb-6" />

                <h4 className="text-sm font-semibold text-orange-600 mb-2">Multi-Factor Authentication Types</h4>
                <p className="text-xs text-gray-600 mb-4">
                  MFA Types must be registered so that you can provide a secondary means of verifying your identity, in addition to your password.{' '}
                  <span className="text-orange-500 font-medium cursor-pointer hover:text-orange-600">Learn more</span>
                </p>

                {/* MFA Types table */}
                <div className="border border-border rounded-lg overflow-hidden">
                  <div className="grid grid-cols-[1fr_auto] bg-gray-50 px-4 py-2 border-b border-border">
                    <span className="text-[11px] font-bold text-orange-600 uppercase tracking-wider">MFA Type</span>
                    <span className="text-[11px] font-bold text-gray-600 uppercase tracking-wider">Created</span>
                  </div>
                  <div className="divide-y divide-border">
                    <div className="flex items-center gap-3 px-4 py-3">
                      <svg className="w-5 h-5 text-gray-500" fill="none" stroke="currentColor" strokeWidth="1.5" viewBox="0 0 24 24">
                        <rect x="3" y="3" width="7" height="7" rx="1" />
                        <rect x="14" y="3" width="7" height="7" rx="1" />
                        <rect x="3" y="14" width="7" height="7" rx="1" />
                        <rect x="14" y="14" width="7" height="7" rx="1" />
                      </svg>
                      <span className="text-xs text-orange-500 font-medium">+ Add Authenticator App</span>
                    </div>
                    <div className="flex items-center gap-3 px-4 py-3">
                      <svg className="w-5 h-5 text-gray-500" fill="none" stroke="currentColor" strokeWidth="1.5" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" d="M15.75 5.25a3 3 0 013 3m3 0a6 6 0 01-7.029 5.912c-.563-.097-1.159.026-1.563.43L10.5 17.25H8.25v2.25H6v2.25H2.25v-2.818c0-.597.237-1.17.659-1.591l6.499-6.499c.404-.404.527-1 .43-1.563A6 6 0 1121.75 8.25z" />
                      </svg>
                      <span className="text-xs text-orange-500 font-medium">+ Add Backup Codes</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

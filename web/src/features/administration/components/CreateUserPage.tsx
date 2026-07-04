import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { PageHeader } from '@/shared/components/PageHeader'
import { Dropdown } from '@/shared/components/Dropdown'

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

export function CreateUserPage() {
  const navigate = useNavigate()
  const [status, setStatus] = useState('enabled')
  const [firstName, setFirstName] = useState('')
  const [lastName, setLastName] = useState('')
  const [email, setEmail] = useState('')
  const [countryCode, setCountryCode] = useState('+91')
  const [phone, setPhone] = useState('')
  const [selectedRole, setSelectedRole] = useState('')

  function handleSave() {
    console.info('[create-user] submitted', { firstName, lastName, email, countryCode, phone, role: selectedRole, status })
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
              className="w-full text-left px-5 py-2.5 text-xs font-medium text-orange-600 bg-orange-50 relative"
            >
              <span className="absolute left-0 top-0 bottom-0 w-0.5 bg-orange-500 rounded-r" />
              <span>User</span>
            </button>
          </div>
          <div className="px-4 py-3 border-t border-border space-y-2">
            <button
              type="button"
              onClick={handleSave}
              className="w-full px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors"
            >
              Save User
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
          <div>
            {/* Header */}
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-sm font-semibold text-gray-900">Dipole User</h3>
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

            {/* User Details */}
            <h4 className="text-[11px] font-bold text-gray-600 uppercase tracking-wider mb-4">User Details</h4>

            <div className="space-y-4 mb-8">
              <div>
                <label className="block text-xs text-gray-500 mb-1">First Name</label>
                <input
                  type="text"
                  value={firstName}
                  onChange={(e) => setFirstName(e.target.value)}
                  placeholder="John"
                  className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400"
                />
              </div>
              <div>
                <label className="block text-xs text-gray-500 mb-1">Last Name</label>
                <input
                  type="text"
                  value={lastName}
                  onChange={(e) => setLastName(e.target.value)}
                  placeholder="Smith"
                  className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400"
                />
              </div>
              <div>
                <label className="block text-xs text-gray-500 mb-1">Email</label>
                <input
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="john.smith@company.com"
                  className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400"
                />
              </div>
              <div className="flex gap-3">
                <div className="w-32">
                  <label className="block text-xs text-gray-500 mb-1">Country Code</label>
                  <Dropdown
                    options={COUNTRY_CODE_OPTIONS}
                    value={countryCode}
                    onChange={setCountryCode}
                    placeholder="+1"
                  />
                </div>
                <div className="flex-1">
                  <label className="block text-xs text-gray-500 mb-1">Phone Number</label>
                  <input
                    type="tel"
                    value={phone}
                    onChange={(e) => setPhone(e.target.value)}
                    placeholder="3105551212"
                    className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400"
                  />
                </div>
              </div>
            </div>

            {/* Role Assignments */}
            <h4 className="text-[11px] font-bold text-gray-600 uppercase tracking-wider mb-4">Role Assignment</h4>

            <div className="mb-8">
              <Dropdown
                options={ROLE_OPTIONS}
                value={selectedRole}
                onChange={setSelectedRole}
                placeholder="Select a role..."
              />
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { FileText } from 'lucide-react'
import { PageHeader } from '@/shared/components/PageHeader'
import { FloatingInput, FloatingDropdown } from '@/shared/components/FloatingInput'
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

const DETAIL_SECTIONS = [
  { value: 'details', label: 'Details' },
  { value: 'metrics', label: 'Metrics' },
  { value: 'logs', label: 'Logs' },
] as const

type DetailSection = (typeof DETAIL_SECTIONS)[number]['value']

export function UserDetailPage() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const user = usersData.find((u) => u.id === id)

  const [selectedSection, setSelectedSection] = useState<DetailSection>('details')
  const [sectionOpen, setSectionOpen] = useState(false)
  const [isEditing, setIsEditing] = useState(false)
  const [firstName, setFirstName] = useState(user?.firstName ?? '')
  const [lastName, setLastName] = useState(user?.lastName ?? '')
  const [email, setEmail] = useState(user?.email ?? '')
  const [countryCode, setCountryCode] = useState('+91')
  const [phone, setPhone] = useState('')
  const [role, setRole] = useState(user?.role ?? '')
  const [status, setStatus] = useState(user?.isActive ? 'enabled' : 'disabled')

  if (!user) {
    return (
      <div className="flex flex-col h-full">
        <div className="flex items-center justify-between px-4 py-3 shrink-0 min-h-[36px] border-b border-border bg-white"><PageHeader /></div>
        <div className="flex-1 flex items-center justify-center flex-col">
          <p className="text-sm text-gray-500">User not found.</p>
          <button onClick={() => navigate('/administration')} className="mt-4 text-xs text-orange-500 hover:text-orange-600 font-medium">← Back to Administration</button>
        </div>
      </div>
    )
  }

  const disabled = !isEditing
  const selectedLabel = DETAIL_SECTIONS.find((s) => s.value === selectedSection)?.label ?? ''

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
                  <h4 className="text-sm font-semibold text-gray-800 mb-4">Details</h4>
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

                <div className="border border-border rounded-lg p-5">
                  <h4 className="text-sm font-semibold text-gray-800 mb-4">Role Assignment</h4>
                  <div className="space-y-4">
                    <FloatingDropdown label="Role" options={ROLE_OPTIONS} value={role} onChange={setRole} disabled={disabled} />
                  </div>
                </div>
              </div>
            </div>

            {/* Right panel — 40% */}
            <div className="w-[40%] px-6 py-6 overflow-y-auto">
              <div className="border border-border rounded-lg p-5 mb-5">
                <h4 className="text-sm font-semibold text-gray-800 mb-4">Metadata</h4>
                <div className="space-y-4">
                  <FloatingInput label="ID" value={user.id} disabled />
                  <FloatingInput label="Email" value={user.email} disabled />
                  <FloatingInput label="MFA Enabled" value={user.mfaEnabled ? 'Yes' : 'No'} disabled />
                </div>
              </div>

              <div className="border border-border rounded-lg p-5">
                <p className="text-xs text-gray-700 leading-relaxed mb-3">
                  <span className="font-semibold text-orange-600">Users</span> are administrators who manage the Dipole platform. Each user is assigned a role that determines their level of access.
                </p>
                <p className="text-xs text-gray-700 leading-relaxed">
                  Assign a <span className="font-semibold">Role</span> to control what actions the user can perform across entities like Access Policies, Workloads, and Credential Providers.
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

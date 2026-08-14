import { useState } from 'react'
import { apiBaseUrl } from '@/shared/lib/platform-urls'
import { Copy } from 'lucide-react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { PageHeader } from '@/shared/components/PageHeader'
import { FloatingInput, FloatingDropdown } from '@/shared/components/FloatingInput'
import { apiClient } from '@/shared/lib/api-client'
import { MFAConfigTab } from '@/features/profile/components/MFAConfigTab'
import { APITokensSection } from '@/features/profile/components/APITokensSection'

function getUserRole(): string {
  try {
    const token = sessionStorage.getItem('dipole_token')
    if (!token) return ''
    const parts = token.split('.')
    const segment = parts[1]
    if (!segment) return ''
    const payload = JSON.parse(atob(segment.replace(/-/g, '+').replace(/_/g, '/')))
    return payload.role_name || ''
  } catch { return '' }
}

const COUNTRY_CODE_OPTIONS = [
  { value: '+91', label: '+91 (India)' },
  { value: '+1', label: '+1 (US/Canada)' },
  { value: '+44', label: '+44 (UK)' },
  { value: '+61', label: '+61 (Australia)' },
  { value: '+65', label: '+65 (Singapore)' },
  { value: '+971', label: '+971 (UAE)' },
]

const TABS = [
  { key: 'details', label: 'Details' },
  { key: 'password', label: 'Password' },
  { key: 'mfa', label: 'Configure MFA' },
  { key: 'preferences', label: 'Preferences' },
] as const

type TabKey = (typeof TABS)[number]['key']

interface ProfileData {
  firstName: string
  lastName: string
  email: string
  phone: string
}

export function ProfilePage() {
  const [activeTab, setActiveTab] = useState<TabKey>('details')

  const { data } = useQuery<ProfileData>({
    queryKey: ['profile'],
    queryFn: () => apiClient.get<ProfileData>('/profile'),
  })

  if (!data) {
    return <div className="flex-1" />
  }

  return <ProfileContent data={data} activeTab={activeTab} setActiveTab={setActiveTab} />
}

function ProfileContent({
  data,
  activeTab,
  setActiveTab,
}: {
  data: ProfileData
  activeTab: TabKey
  setActiveTab: (tab: TabKey) => void
}) {
  const qc = useQueryClient()
  const [isEditing, setIsEditing] = useState(false)

  const [firstName, setFirstName] = useState(data.firstName)
  const [lastName, setLastName] = useState(data.lastName)
  const [email, setEmail] = useState(data.email)
  const [countryCode, setCountryCode] = useState('+91')
  const [phone, setPhone] = useState(data.phone)

  const saveMutation = useMutation({
    mutationFn: () =>
      apiClient.put<ProfileData>('/profile', { firstName, lastName, email, phone }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['profile'] })
      setIsEditing(false)
    },
    onError: () => {
      setFirstName(data.firstName)
      setLastName(data.lastName)
      setEmail(data.email)
      setPhone(data.phone)
      setIsEditing(false)
    },
  })

  const disabled = !isEditing

  function handleEditToggle() {
    if (isEditing) {
      saveMutation.mutate()
    } else {
      setIsEditing(true)
    }
  }

  function handleCancel() {
    setFirstName(data.firstName)
    setLastName(data.lastName)
    setEmail(data.email)
    setPhone(data.phone)
    setIsEditing(false)
  }

  function copyToClipboard(text: string) {
    navigator.clipboard.writeText(text)
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-4 shrink-0 h-[60px] border-b border-border bg-white">
        <PageHeader />
        <div className="flex items-center gap-3">
          <button type="button" onClick={handleEditToggle} disabled={saveMutation.isPending} className="px-4 py-2 bg-gray-800 hover:bg-gray-900 text-white text-sm font-medium rounded-md transition-colors disabled:opacity-60">
            {saveMutation.isPending ? 'Saving…' : isEditing ? 'Save' : 'Edit'}
          </button>
          {isEditing && (
            <button type="button" onClick={handleCancel} className="px-4 py-2 border border-gray-300 text-sm font-medium text-gray-600 hover:text-gray-900 hover:border-gray-400 rounded-md transition-colors">
              Cancel
            </button>
          )}
        </div>
      </div>

      <div className="flex items-center bg-white px-6 pt-4 shrink-0">
        {TABS.map((tab, idx) => (
          <button
            key={tab.key}
            type="button"
            onClick={() => setActiveTab(tab.key)}
            className={`${idx === 0 ? 'pl-0 pr-3' : 'px-3'} py-2.5 text-xs font-medium transition-colors relative ${
              activeTab === tab.key ? 'text-gray-700' : 'text-gray-500 hover:text-gray-800'
            }`}
          >
            {tab.label}
            {activeTab === tab.key && (
              <span className="absolute bottom-0 left-0 right-0 h-0.5 bg-gray-800 rounded-t" />
            )}
          </button>
        ))}
      </div>

      <div className="flex-1 overflow-y-auto bg-white">
        <div className="flex h-full">
          <div className="w-[60%] px-6 py-6 overflow-y-auto">
            {activeTab === 'details' && (
              <div className="space-y-5">
                <div className={`space-y-5 ${disabled ? 'opacity-60 pointer-events-none' : ''}`}>
                  <div className="border border-border rounded-lg p-5">
                    <h4 className="text-sm font-semibold text-gray-800 mb-4">User Details</h4>
                    <div className="space-y-4">
                      <FloatingInput label="First Name" value={firstName} onChange={setFirstName} disabled={disabled} required />
                      <FloatingInput label="Last Name" value={lastName} onChange={setLastName} disabled={disabled} required />
                      <FloatingInput label="Email" value={email} onChange={setEmail} disabled={disabled} required />
                      <div className="flex gap-3">
                        <div className="w-36">
                          <FloatingDropdown label="Code" options={COUNTRY_CODE_OPTIONS} value={countryCode} onChange={setCountryCode} disabled={disabled} />
                        </div>
                        <div className="flex-1">
                          <FloatingInput label="Phone Number" value={phone} onChange={setPhone} disabled={disabled} />
                        </div>
                      </div>
                    </div>
                  </div>

                  <div className="border border-border rounded-lg p-5">
                    <h4 className="text-sm font-semibold text-gray-800 mb-4">Role Assignments</h4>
                    <FloatingInput label="Role" value={getUserRole()} disabled />
                  </div>
                </div>

                <div className="border border-border rounded-lg p-5">
                  <h4 className="text-sm font-semibold text-gray-800 mb-4">API Access</h4>
                  <div className="space-y-4">
                    <APITokensSection />
                    <CopyField label="Cloud API Base URL" value={apiBaseUrl() + '/'} onCopy={copyToClipboard} />
                    <CopyField label="Edge API Base URL" value={apiBaseUrl() + '/'} onCopy={copyToClipboard} />
                    <CopyField label="MCP Service URL" value={apiBaseUrl() + '/mcp'} onCopy={copyToClipboard} />
                  </div>
                </div>
              </div>
            )}

            {activeTab === 'password' && (
              <PasswordChangeForm />
            )}

            {activeTab === 'mfa' && (
              <MFAConfigTab />
            )}

            {activeTab === 'preferences' && (
              <div className="space-y-5">
                <div className="border border-border rounded-lg p-5">
                  <h4 className="text-sm font-semibold text-gray-800 mb-4">Preferences</h4>
                  <p className="text-xs text-gray-600 mb-4">Customize your Dipole experience.</p>
                  <div className="space-y-0">
                    <div className="flex items-center justify-between py-3 border-b border-border">
                      <span className="text-sm text-gray-700">Dark Mode</span>
                      <span className="text-xs text-gray-400">Coming soon</span>
                    </div>
                    <div className="flex items-center justify-between py-3 border-b border-border">
                      <span className="text-sm text-gray-700">Email Notifications</span>
                      <span className="text-xs text-gray-400">Coming soon</span>
                    </div>
                    <div className="flex items-center justify-between py-3">
                      <span className="text-sm text-gray-700">Timezone</span>
                      <span className="text-xs text-gray-400">Coming soon</span>
                    </div>
                  </div>
                </div>
              </div>
            )}
          </div>

          <div className="w-[40%] px-6 py-6 overflow-y-auto">
            <div className="border border-border rounded-lg p-5 sticky top-0">
              {activeTab === 'details' && (
                <>
                  <p className="text-xs text-gray-700 leading-relaxed mb-3">
                    <span className="font-semibold text-gray-700">User Profile</span> displays your account details and API access credentials.
                  </p>
                  <p className="text-xs text-gray-700 leading-relaxed">
                    Use the <span className="font-semibold">Generate</span> button to create a session access token for programmatic API access. Tokens are short-lived and scoped to your permissions.
                  </p>
                </>
              )}
              {activeTab === 'password' && (
                <p className="text-xs text-gray-700 leading-relaxed">
                  Update your password here. After changing your password, you'll remain signed in on this device.
                </p>
              )}
              {activeTab === 'mfa' && (
                <>
                  <p className="text-xs text-gray-700 leading-relaxed mb-3">
                    <span className="font-semibold text-gray-700">Multi-Factor Authentication</span> adds a second verification step to your sign-in process.
                  </p>
                  <p className="text-xs text-gray-700 leading-relaxed">
                    Use an authenticator app (Google Authenticator, Authy) for time-based one-time codes. Generate backup codes for emergency access.
                  </p>
                </>
              )}
              {activeTab === 'preferences' && (
                <p className="text-xs text-gray-700 leading-relaxed">
                  Preferences let you customize your Dipole experience including display settings and notification preferences.
                </p>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

function CopyField({ label, value, onCopy }: { label: string; value: string; onCopy: (v: string) => void }) {
  return (
    <div className="relative">
      <FloatingInput label={label} value={value} disabled />
      <button type="button" onClick={() => onCopy(value)} className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600"><Copy size={14} /></button>
    </div>
  )
}

function PasswordChangeForm() {
  const [currentPassword, setCurrentPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [loading, setLoading] = useState(false)

  async function handleSubmit() {
    setError('')
    setSuccess('')

    if (!currentPassword) { setError('Current password is required.'); return }
    if (newPassword.length < 6) { setError('New password must be at least 6 characters.'); return }
    if (newPassword !== confirmPassword) { setError('Passwords do not match.'); return }

    setLoading(true)
    try {
      await apiClient.post('/profile/password', { currentPassword, newPassword })
      sessionStorage.removeItem('dipole_token')
      sessionStorage.removeItem('dipole_user')
      window.location.href = '/login'
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Failed to update password.'
      setError(msg)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="space-y-5">
      <div className="border border-border rounded-lg p-5">
        <h4 className="text-sm font-semibold text-gray-800 mb-4">Change Password</h4>
        <div className="space-y-4">
          <FloatingInput label="Current Password" value={currentPassword} onChange={setCurrentPassword} type="password" />
          <FloatingInput label="New Password" value={newPassword} onChange={setNewPassword} type="password" />
          <FloatingInput label="Confirm New Password" value={confirmPassword} onChange={setConfirmPassword} type="password" />

          {error && <p className="text-xs text-red-600">{error}</p>}
          {success && <p className="text-xs text-green-600">{success}</p>}

          <button
            type="button"
            onClick={handleSubmit}
            disabled={loading}
            className="px-4 py-2 bg-gray-800 hover:bg-gray-900 text-white text-xs font-medium rounded-md transition-colors disabled:opacity-50"
          >
            {loading ? 'Updating…' : 'Update Password'}
          </button>
        </div>
      </div>
    </div>
  )
}

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

function parsePhone(fullPhone: string): { code: string; number: string } {
  if (!fullPhone) return { code: '+1', number: '' }
  // Match known country codes (longest first to avoid partial matches)
  const codes = ['+971', '+91', '+65', '+61', '+44', '+1']
  for (const c of codes) {
    if (fullPhone.startsWith(c)) {
      return { code: c, number: fullPhone.slice(c.length) }
    }
  }
  // If phone starts with + but doesn't match known codes, keep as-is
  if (fullPhone.startsWith('+')) {
    return { code: '+1', number: fullPhone }
  }
  return { code: '+1', number: fullPhone }
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
  const [saveError, setSaveError] = useState('')

  const [firstName, setFirstName] = useState(data.firstName)
  const [lastName, setLastName] = useState(data.lastName)
  const parsed = parsePhone(data.phone)
  const [countryCode, setCountryCode] = useState(parsed.code)
  const [phone, setPhone] = useState(parsed.number)

  const saveMutation = useMutation({
    mutationFn: () => {
      const fullPhone = phone ? `${countryCode}${phone}` : ''
      return apiClient.put<ProfileData>('/profile', { firstName, lastName, email: data.email, phone: fullPhone })
    },
    onSuccess: () => {
      setSaveError('')
      qc.invalidateQueries({ queryKey: ['profile'] })
      setIsEditing(false)

      // Update sessionStorage so the sidebar name reflects immediately.
      const displayName = [firstName, lastName].filter(Boolean).join(' ')
      if (displayName) {
        try {
          const raw = sessionStorage.getItem('dipole_user')
          const user = raw ? JSON.parse(raw) : {}
          user.name = displayName
          sessionStorage.setItem('dipole_user', JSON.stringify(user))
        } catch { /* ignore */ }
      }
    },
    onError: (err: unknown) => {
      const msg = err instanceof Error ? err.message : 'Failed to save profile.'
      setSaveError(msg)
    },
  })

  const disabled = !isEditing

  function handleEdit() {
    setIsEditing(true)
  }

  function handleSave() {
    saveMutation.mutate()
  }

  function handleCancel() {
    const p = parsePhone(data.phone)
    setFirstName(data.firstName)
    setLastName(data.lastName)
    setCountryCode(p.code)
    setPhone(p.number)
    setSaveError('')
    setIsEditing(false)
  }

  function copyToClipboard(text: string) {
    navigator.clipboard.writeText(text)
  }

  return (
    <div className="flex flex-col h-full">
      {/* Header bar */}
      <div className="flex items-center justify-between px-4 shrink-0 h-[44px] border-b border-border bg-white">
        <PageHeader />
      </div>

      {/* Toolbar — tabs on left, actions on right */}
      <div className="flex items-center justify-between px-4 py-3 shrink-0">
        {/* Tabs (clickable) */}
        <div className="flex items-center gap-1">
          {TABS.map((tab) => (
            <button
              key={tab.key}
              type="button"
              onClick={() => setActiveTab(tab.key)}
              className={`px-3 py-1.5 text-xs font-medium transition-colors ${
                tab.key === activeTab ? 'text-gray-900' : 'text-gray-400 hover:text-gray-700'
              }`}
            >
              {tab.label}
            </button>
          ))}
        </div>

        {/* Actions — only show Edit/Save/Cancel on Details tab */}
        <div className="flex items-center gap-2 h-9">
          {activeTab === 'details' && (
            <>
              {isEditing ? (
                <>
                  <button
                    type="button"
                    onClick={handleCancel}
                    className="inline-flex items-center gap-2 h-9 px-4 rounded-md border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors"
                  >
                    Cancel
                  </button>
                  <button
                    type="button"
                    onClick={handleSave}
                    disabled={saveMutation.isPending}
                    className="inline-flex items-center gap-2 h-9 px-4 rounded-md border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors disabled:opacity-50"
                  >
                    {saveMutation.isPending ? 'Saving…' : 'Save'}
                  </button>
                </>
              ) : (
                <button
                  type="button"
                  onClick={handleEdit}
                  className="inline-flex items-center gap-2 h-9 px-4 rounded-md border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors"
                >
                  Edit
                </button>
              )}
            </>
          )}
        </div>
      </div>

      {/* Content box */}
      <div className="flex-1 min-h-0 mx-4 border border-border bg-white overflow-y-auto">
        <div className="grid grid-cols-[1fr_380px] h-full">
          {/* Form area */}
          <div className="px-6 py-5 overflow-y-auto border-r border-border">
            {activeTab === 'details' && (
              <div className="space-y-6">
                {saveError && (
                  <div className="bg-red-50 border border-red-200 rounded-md px-3 py-2">
                    <p className="text-xs text-red-700">{saveError}</p>
                  </div>
                )}

                <div className={disabled ? 'opacity-60 pointer-events-none' : ''}>
                  <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider mb-4">User Details</h4>
                  <div className="space-y-4">
                    <FloatingInput label="First Name" value={firstName} onChange={setFirstName} disabled={disabled} required />
                    <FloatingInput label="Last Name" value={lastName} onChange={setLastName} disabled={disabled} required />
                    <FloatingInput label="Email" value={data.email} onChange={() => {}} disabled />
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

                <div className={disabled ? 'opacity-60 pointer-events-none' : ''}>
                  <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider mb-4">Role Assignments</h4>
                  <FloatingInput label="Role" value={getUserRole()} disabled />
                </div>

                <div>
                  <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider mb-4">API Access</h4>
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
              <div className="space-y-6">
                <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider mb-4">Preferences</h4>
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
            )}
          </div>

          {/* Help text — right panel */}
          <div className="px-5 py-5 overflow-y-auto">
            {activeTab === 'details' && (
              <div className="space-y-3">
                <p className="text-xs text-gray-600 leading-relaxed">
                  <span className="font-semibold text-gray-800">User Profile</span> displays your account details and API access credentials.
                </p>
                <p className="text-xs text-gray-600 leading-relaxed">
                  Use the <span className="font-semibold text-gray-800">Generate</span> button to create a session access token for programmatic API access. Tokens are short-lived and scoped to your permissions.
                </p>
              </div>
            )}
            {activeTab === 'password' && (
              <div className="space-y-3">
                <p className="text-xs text-gray-600 leading-relaxed">
                  Update your password here. After changing your password, you'll be signed out and redirected to the login page.
                </p>
              </div>
            )}
            {activeTab === 'mfa' && (
              <div className="space-y-3">
                <p className="text-xs text-gray-600 leading-relaxed">
                  <span className="font-semibold text-gray-800">Multi-Factor Authentication</span> adds a second verification step to your sign-in process.
                </p>
                <p className="text-xs text-gray-600 leading-relaxed">
                  Use an authenticator app (Google Authenticator, Authy) for time-based one-time codes. Generate backup codes for emergency access.
                </p>
              </div>
            )}
            {activeTab === 'preferences' && (
              <div className="space-y-3">
                <p className="text-xs text-gray-600 leading-relaxed">
                  Preferences let you customize your Dipole experience including display settings and notification preferences.
                </p>
              </div>
            )}
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
  const [loading, setLoading] = useState(false)

  async function handleSubmit() {
    setError('')

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
    <div className="space-y-6">
      <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider mb-4">Change Password</h4>
      <div className="space-y-4">
        <FloatingInput label="Current Password" value={currentPassword} onChange={setCurrentPassword} type="password" />
        <FloatingInput label="New Password" value={newPassword} onChange={setNewPassword} type="password" />
        <FloatingInput label="Confirm New Password" value={confirmPassword} onChange={setConfirmPassword} type="password" />

        {error && <p className="text-xs text-red-600">{error}</p>}

        <button
          type="button"
          onClick={handleSubmit}
          disabled={loading}
          className="inline-flex items-center gap-2 h-9 px-4 rounded-md border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors disabled:opacity-50"
        >
          {loading ? 'Updating…' : 'Update Password'}
        </button>
      </div>
    </div>
  )
}

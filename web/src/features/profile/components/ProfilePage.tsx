import { useState } from 'react'
import { Copy } from 'lucide-react'
import { PageHeader } from '@/shared/components/PageHeader'
import { FloatingInput, FloatingDropdown } from '@/shared/components/FloatingInput'
import { userProfileData, COUNTRY_CODE_OPTIONS } from '@/data/profile/userProfile'

const TABS = [
  { key: 'details', label: 'Details' },
  { key: 'password', label: 'Password' },
  { key: 'mfa', label: 'Configure MFA' },
  { key: 'preferences', label: 'Preferences' },
] as const

type TabKey = (typeof TABS)[number]['key']

export function ProfilePage() {
  const [activeTab, setActiveTab] = useState<TabKey>('details')
  const [isEditing, setIsEditing] = useState(false)

  // Details state
  const [firstName, setFirstName] = useState(userProfileData.firstName)
  const [lastName, setLastName] = useState(userProfileData.lastName)
  const [email, setEmail] = useState(userProfileData.email)
  const [countryCode, setCountryCode] = useState(userProfileData.countryCode)
  const [phone, setPhone] = useState(userProfileData.phone)

  // API Access state
  const [tokenLifetime, setTokenLifetime] = useState('3600')
  const [accessToken, setAccessToken] = useState('')

  const disabled = !isEditing

  function handleEditToggle() {
    if (isEditing) {
      console.info('[save-profile]', { firstName, lastName, email, countryCode, phone })
      setIsEditing(false)
    } else {
      setIsEditing(true)
    }
  }

  function handleCancel() {
    if (isEditing) {
      setFirstName(userProfileData.firstName)
      setLastName(userProfileData.lastName)
      setEmail(userProfileData.email)
      setCountryCode(userProfileData.countryCode)
      setPhone(userProfileData.phone)
      setIsEditing(false)
    }
  }

  function handleGenerateToken() {
    setAccessToken('dpl_' + Math.random().toString(36).substring(2, 15) + Math.random().toString(36).substring(2, 15))
  }

  function copyToClipboard(text: string) {
    navigator.clipboard.writeText(text)
  }

  return (
    <div className="flex flex-col h-full">
      {/* Header bar */}
      <div className="flex items-center justify-between px-4 py-3 shrink-0 min-h-[36px] border-b border-border bg-white">
        <PageHeader />
        <div className="flex items-center gap-3">
          <button type="button" onClick={handleEditToggle} className="px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white text-sm font-medium rounded-md transition-colors">
            {isEditing ? 'Save' : 'Edit'}
          </button>
          {isEditing && (
            <button type="button" onClick={handleCancel} className="px-4 py-2 border border-gray-300 text-sm font-medium text-gray-600 hover:text-gray-900 hover:border-gray-400 rounded-md transition-colors">
              Cancel
            </button>
          )}
        </div>
      </div>

      {/* Horizontal tabs */}
      <div className="flex items-center bg-white px-6 pt-4 shrink-0">
        {TABS.map((tab, idx) => (
          <button
            key={tab.key}
            type="button"
            onClick={() => setActiveTab(tab.key)}
            className={`${idx === 0 ? 'pl-0 pr-3' : 'px-3'} py-2.5 text-xs font-medium transition-colors relative ${
              activeTab === tab.key ? 'text-orange-600' : 'text-gray-500 hover:text-gray-800'
            }`}
          >
            {tab.label}
            {activeTab === tab.key && (
              <span className="absolute bottom-0 left-0 right-0 h-0.5 bg-orange-500 rounded-t" />
            )}
          </button>
        ))}
      </div>

      {/* Tab content */}
      <div className="flex-1 overflow-y-auto bg-white">
        <div className="flex h-full">
          {/* Left 60% */}
          <div className="w-[60%] px-6 py-6 overflow-y-auto">
            {activeTab === 'details' && (
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
                  <FloatingInput label="Role" value={userProfileData.role} disabled />
                </div>

                <div className="border border-border rounded-lg p-5">
                  <h4 className="text-sm font-semibold text-gray-800 mb-4">API Access</h4>
                  <div className="space-y-4">
                    <div className="flex items-end gap-3">
                      <div className="flex-1">
                        <div className="relative">
                          <FloatingInput label="Access Token" value={accessToken} disabled />
                          {accessToken && (
                            <button type="button" onClick={() => copyToClipboard(accessToken)} className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600"><Copy size={14} /></button>
                          )}
                        </div>
                      </div>
                      <div className="w-28">
                        <FloatingInput label="Lifetime (s)" value={tokenLifetime} onChange={setTokenLifetime} type="number" />
                      </div>
                      <button type="button" onClick={handleGenerateToken} className="px-4 h-[44px] bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors">
                        Generate
                      </button>
                    </div>
                    <p className="text-[10px] text-gray-400">Max lifetime: 604,800 seconds (7 days)</p>

                    <CopyField label="Cloud API Base URL" value={userProfileData.cloudApiBaseUrl} onCopy={copyToClipboard} />
                    <CopyField label="Edge API Base URL" value={userProfileData.edgeApiBaseUrl} onCopy={copyToClipboard} />
                    <CopyField label="MCP Service URL" value={userProfileData.mcpServiceUrl} onCopy={copyToClipboard} />
                  </div>
                </div>
              </div>
            )}

            {activeTab === 'password' && (
              <div className="space-y-5">
                <div className="border border-border rounded-lg p-5">
                  <h4 className="text-sm font-semibold text-gray-800 mb-4">Change Password</h4>
                  <div className="space-y-4">
                    <FloatingInput label="Current Password" value="" onChange={() => {}} type="password" />
                    <FloatingInput label="New Password" value="" onChange={() => {}} type="password" />
                    <FloatingInput label="Confirm New Password" value="" onChange={() => {}} type="password" />
                    <button type="button" className="px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors">
                      Update Password
                    </button>
                  </div>
                </div>
              </div>
            )}

            {activeTab === 'mfa' && (
              <div className="space-y-5">
                <div className="border border-border rounded-lg p-5">
                  <h4 className="text-sm font-semibold text-gray-800 mb-4">Multi-Factor Authentication</h4>
                  <p className="text-xs text-gray-600 mb-4">
                    Configure multi-factor authentication to add an extra layer of security to your account.
                  </p>
                  <div className="space-y-0">
                    <div className="flex items-center justify-between py-3 border-b border-border">
                      <div className="flex items-center gap-3">
                        <svg className="w-5 h-5 text-gray-500" fill="none" stroke="currentColor" strokeWidth="1.5" viewBox="0 0 24 24">
                          <rect x="3" y="3" width="7" height="7" rx="1" /><rect x="14" y="3" width="7" height="7" rx="1" /><rect x="3" y="14" width="7" height="7" rx="1" /><rect x="14" y="14" width="7" height="7" rx="1" />
                        </svg>
                        <span className="text-sm text-gray-700">Authenticator App</span>
                      </div>
                      <button type="button" className="text-xs text-orange-500 hover:text-orange-600 font-medium">+ Configure</button>
                    </div>
                    <div className="flex items-center justify-between py-3">
                      <div className="flex items-center gap-3">
                        <svg className="w-5 h-5 text-gray-500" fill="none" stroke="currentColor" strokeWidth="1.5" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" d="M15.75 5.25a3 3 0 013 3m3 0a6 6 0 01-7.029 5.912c-.563-.097-1.159.026-1.563.43L10.5 17.25H8.25v2.25H6v2.25H2.25v-2.818c0-.597.237-1.17.659-1.591l6.499-6.499c.404-.404.527-1 .43-1.563A6 6 0 1121.75 8.25z" />
                        </svg>
                        <span className="text-sm text-gray-700">Backup Codes</span>
                      </div>
                      <button type="button" className="text-xs text-orange-500 hover:text-orange-600 font-medium">+ Generate</button>
                    </div>
                  </div>
                </div>
              </div>
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

          {/* Right 40% — help */}
          <div className="w-[40%] px-6 py-6 overflow-y-auto">
            <div className="border border-border rounded-lg p-5 sticky top-0">
              {activeTab === 'details' && (
                <>
                  <p className="text-xs text-gray-700 leading-relaxed mb-3">
                    <span className="font-semibold text-orange-600">User Profile</span> displays your account details, role assignments, and API access credentials.
                  </p>
                  <p className="text-xs text-gray-700 leading-relaxed">
                    Use the <span className="font-semibold">Generate</span> button to create a session access token for programmatic API access. Tokens are short-lived and scoped to your permissions.
                  </p>
                </>
              )}
              {activeTab === 'password' && (
                <p className="text-xs text-gray-700 leading-relaxed">
                  Update your password here. Passwords must meet minimum complexity requirements. After changing your password, you'll remain signed in on this device.
                </p>
              )}
              {activeTab === 'mfa' && (
                <>
                  <p className="text-xs text-gray-700 leading-relaxed mb-3">
                    <span className="font-semibold text-orange-600">Multi-Factor Authentication</span> adds a second verification step to your sign-in process.
                  </p>
                  <p className="text-xs text-gray-700 leading-relaxed">
                    Use an authenticator app (Google Authenticator, Authy, etc.) for time-based one-time codes. Generate backup codes for emergency access.
                  </p>
                </>
              )}
              {activeTab === 'preferences' && (
                <p className="text-xs text-gray-700 leading-relaxed">
                  Preferences let you customize your Dipole experience including display settings, notification preferences, and timezone configuration.
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

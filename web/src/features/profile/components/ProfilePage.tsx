import { useState } from 'react'
import { Copy } from 'lucide-react'
import { PageHeader } from '@/shared/components/PageHeader'
import { Dropdown } from '@/shared/components/Dropdown'
import { userProfileData, COUNTRY_CODE_OPTIONS } from '@/data/profile/userProfile'

type TabKey = 'details' | 'password' | 'mfa' | 'preferences'

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
    } else {
      window.history.back()
    }
  }

  function handleGenerateToken() {
    setAccessToken('dpl_' + Math.random().toString(36).substring(2, 15) + Math.random().toString(36).substring(2, 15))
  }

  function copyToClipboard(text: string) {
    navigator.clipboard.writeText(text)
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
              className={`w-full text-left px-5 py-2.5 text-xs font-medium transition-colors relative ${activeTab === 'details' ? 'text-orange-600 bg-orange-50' : 'text-gray-500 hover:text-gray-800 hover:bg-gray-50'}`}
            >
              {activeTab === 'details' && <span className="absolute left-0 top-0 bottom-0 w-0.5 bg-orange-500 rounded-r" />}
              <span>Details</span>
            </button>
            <button
              type="button"
              onClick={() => setActiveTab('password')}
              className={`w-full text-left px-5 py-2.5 text-xs font-medium transition-colors relative ${activeTab === 'password' ? 'text-orange-600 bg-orange-50' : 'text-gray-500 hover:text-gray-800 hover:bg-gray-50'}`}
            >
              {activeTab === 'password' && <span className="absolute left-0 top-0 bottom-0 w-0.5 bg-orange-500 rounded-r" />}
              <span>Password</span>
            </button>
            <button
              type="button"
              onClick={() => setActiveTab('mfa')}
              className={`w-full text-left px-5 py-2.5 text-xs font-medium transition-colors relative ${activeTab === 'mfa' ? 'text-orange-600 bg-orange-50' : 'text-gray-500 hover:text-gray-800 hover:bg-gray-50'}`}
            >
              {activeTab === 'mfa' && <span className="absolute left-0 top-0 bottom-0 w-0.5 bg-orange-500 rounded-r" />}
              <span>Configure MFA</span>
            </button>
            <button
              type="button"
              onClick={() => setActiveTab('preferences')}
              className={`w-full text-left px-5 py-2.5 text-xs font-medium transition-colors relative ${activeTab === 'preferences' ? 'text-orange-600 bg-orange-50' : 'text-gray-500 hover:text-gray-800 hover:bg-gray-50'}`}
            >
              {activeTab === 'preferences' && <span className="absolute left-0 top-0 bottom-0 w-0.5 bg-orange-500 rounded-r" />}
              <span>Preferences</span>
            </button>
          </div>
          <div className="px-4 py-3 border-t border-border space-y-2">
            <button
              type="button"
              onClick={handleEditToggle}
              className="w-full px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors"
            >
              {isEditing ? 'Save Profile' : 'Edit Profile'}
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
          {activeTab === 'details' && (
            <div>
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-sm font-semibold text-gray-900">User Profile</h3>
                {isEditing && (
                  <button type="button" onClick={handleEditToggle} className="px-4 py-1.5 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors">Save</button>
                )}
              </div>
              <div className="border-b border-border mb-6" />

              <div className="space-y-8">
                {/* User Details */}
                <div>
                  <h4 className="text-[11px] font-bold text-gray-600 uppercase tracking-wider mb-4">User Details</h4>
                  <div className="space-y-4">
                    <div>
                      <label className="block text-xs text-gray-500 mb-1">First Name</label>
                      <input
                        type="text"
                        value={firstName}
                        onChange={(e) => setFirstName(e.target.value)}
                        disabled={disabled}
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
                        className={`w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400 ${disabled ? 'bg-gray-50 text-gray-600' : ''}`}
                      />
                    </div>
                    <div className="flex gap-3">
                      <div className="w-36">
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
                          placeholder="Phone Number"
                          className={`w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400 ${disabled ? 'bg-gray-50 text-gray-600' : ''}`}
                        />
                      </div>
                    </div>
                  </div>
                </div>

                {/* Role Assignments */}
                <div>
                  <h4 className="text-[11px] font-bold text-gray-600 uppercase tracking-wider mb-4">Role Assignments</h4>
                  <div className="px-4 py-3 border border-border rounded-md text-sm text-gray-700 bg-gray-50">
                    {userProfileData.role}
                  </div>
                </div>

                {/* API Access */}
                <div>
                  <h4 className="text-[11px] font-bold text-gray-600 uppercase tracking-wider mb-4">API Access</h4>
                  <div className="space-y-4">
                    <div className="flex items-start gap-4">
                      <div className="flex-1">
                        <label className="block text-xs text-gray-500 mb-1">Access Token</label>
                        <div className="relative">
                          <input
                            type="text"
                            value={accessToken}
                            readOnly
                            placeholder="Access Token"
                            className="w-full px-3 py-2 pr-9 border border-border rounded-md text-sm bg-gray-50 text-gray-600"
                          />
                          {accessToken && (
                            <button
                              type="button"
                              onClick={() => copyToClipboard(accessToken)}
                              className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 transition-colors"
                            >
                              <Copy size={14} />
                            </button>
                          )}
                        </div>
                      </div>
                      <div>
                        <label className="block text-xs text-gray-500 mb-1">Lifetime*</label>
                        <div className="flex items-center gap-2">
                          <input
                            type="number"
                            value={tokenLifetime}
                            onChange={(e) => setTokenLifetime(e.target.value)}
                            className="w-24 px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400"
                          />
                          <span className="text-sm text-gray-600">Secs.</span>
                        </div>
                        <p className="text-[10px] text-gray-400 mt-1">Max 604,800</p>
                      </div>
                      <button
                        type="button"
                        onClick={handleGenerateToken}
                        className="mt-5 px-5 py-2 bg-orange-500 hover:bg-orange-600 text-white text-sm font-medium rounded-md transition-colors"
                      >
                        Generate
                      </button>
                    </div>

                    <div>
                      <label className="block text-xs text-gray-500 mb-1">Cloud API Base URL</label>
                      <div className="relative">
                        <input type="text" value={userProfileData.cloudApiBaseUrl} readOnly className="w-full px-3 py-2 pr-9 border border-border rounded-md text-sm text-gray-700" />
                        <button type="button" onClick={() => copyToClipboard(userProfileData.cloudApiBaseUrl)} className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 transition-colors"><Copy size={14} /></button>
                      </div>
                    </div>

                    <div>
                      <label className="block text-xs text-gray-500 mb-1">Edge API Base URL</label>
                      <div className="relative">
                        <input type="text" value={userProfileData.edgeApiBaseUrl} readOnly className="w-full px-3 py-2 pr-9 border border-border rounded-md text-sm text-gray-700" />
                        <button type="button" onClick={() => copyToClipboard(userProfileData.edgeApiBaseUrl)} className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 transition-colors"><Copy size={14} /></button>
                      </div>
                    </div>

                    <div>
                      <label className="block text-xs text-gray-500 mb-1">MCP Service URL</label>
                      <div className="relative">
                        <input type="text" value={userProfileData.mcpServiceUrl} readOnly className="w-full px-3 py-2 pr-9 border border-border rounded-md text-sm text-gray-700" />
                        <button type="button" onClick={() => copyToClipboard(userProfileData.mcpServiceUrl)} className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 transition-colors"><Copy size={14} /></button>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          )}

          {activeTab === 'password' && (
            <div>
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-sm font-semibold text-gray-900">Change Password</h3>
              </div>
              <div className="border-b border-border mb-6" />
              <div className="space-y-4">
                <div>
                  <label className="block text-xs text-gray-500 mb-1">Current Password</label>
                  <input type="password" placeholder="Current Password" className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400" />
                </div>
                <div>
                  <label className="block text-xs text-gray-500 mb-1">New Password</label>
                  <input type="password" placeholder="New Password" className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400" />
                </div>
                <div>
                  <label className="block text-xs text-gray-500 mb-1">Confirm New Password</label>
                  <input type="password" placeholder="Confirm New Password" className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400" />
                </div>
                <button type="button" className="px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors">
                  Update Password
                </button>
              </div>
            </div>
          )}

          {activeTab === 'mfa' && (
            <div>
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-sm font-semibold text-gray-900">Configure MFA</h3>
              </div>
              <div className="border-b border-border mb-6" />
              <p className="text-xs text-gray-600 mb-4">
                Configure multi-factor authentication to add an extra layer of security to your account.
              </p>
              <div className="border border-border rounded-lg p-4 space-y-3">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <svg className="w-5 h-5 text-gray-500" fill="none" stroke="currentColor" strokeWidth="1.5" viewBox="0 0 24 24">
                      <rect x="3" y="3" width="7" height="7" rx="1" />
                      <rect x="14" y="3" width="7" height="7" rx="1" />
                      <rect x="3" y="14" width="7" height="7" rx="1" />
                      <rect x="14" y="14" width="7" height="7" rx="1" />
                    </svg>
                    <span className="text-sm text-gray-700">Authenticator App</span>
                  </div>
                  <button type="button" className="text-xs text-orange-500 hover:text-orange-600 font-medium">+ Configure</button>
                </div>
                <div className="border-t border-border" />
                <div className="flex items-center justify-between">
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
          )}

          {activeTab === 'preferences' && (
            <div>
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-sm font-semibold text-gray-900">Preferences</h3>
              </div>
              <div className="border-b border-border mb-6" />
              <p className="text-xs text-gray-600 mb-4">
                Customize your Dipole experience with display and notification preferences.
              </p>
              <div className="space-y-0">
                <div className="flex items-center justify-between py-3 border-b border-border">
                  <span className="text-sm text-gray-700">Dark Mode</span>
                  <span className="text-xs text-gray-400">Coming soon</span>
                </div>
                <div className="flex items-center justify-between py-3 border-b border-border">
                  <span className="text-sm text-gray-700">Email Notifications</span>
                  <span className="text-xs text-gray-400">Coming soon</span>
                </div>
                <div className="flex items-center justify-between py-3 border-b border-border">
                  <span className="text-sm text-gray-700">Timezone</span>
                  <span className="text-xs text-gray-400">Coming soon</span>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

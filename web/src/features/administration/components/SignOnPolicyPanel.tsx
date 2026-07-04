import { useState } from 'react'

export function SignOnPolicyPanel() {
  const [requireSSO, setRequireSSO] = useState(false)
  const [requireMFA, setRequireMFA] = useState(false)

  function handleSave() {
    console.info('[sign-on-policy] saved', { requireSSO, requireMFA })
  }

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-sm font-semibold text-gray-900">Sign-On Policy</h2>
        <button
          type="button"
          onClick={handleSave}
          className="px-4 py-1.5 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors"
        >
          Save
        </button>
      </div>

      <div className="space-y-6">
        {/* Require Single Sign-On */}
        <div className="border border-border rounded-lg p-5">
          <div className="flex items-center justify-between mb-2">
            <h3 className="text-sm font-semibold text-gray-900">Require Single Sign-On</h3>
            <Toggle enabled={requireSSO} onChange={setRequireSSO} />
          </div>
          <p className="text-xs text-gray-600 leading-relaxed">
            Mandates that users authenticate through a Single Sign-On provider. This simplifies the login process and enhances security by centralizing authentication management.
          </p>
          <p className="text-[11px] text-gray-400 mt-2">
            Note: Users with the SuperAdmin role can always use native sign-in (username and password) regardless of this setting.
          </p>
        </div>

        {/* Require Multi-Factor Authentication */}
        <div className="border border-border rounded-lg p-5">
          <div className="flex items-center justify-between mb-2">
            <h3 className="text-sm font-semibold text-gray-900">Require Multi-Factor Authentication</h3>
            <Toggle enabled={requireMFA} onChange={setRequireMFA} />
          </div>
          <p className="text-xs text-gray-600 leading-relaxed">
            Enforces the use of multi-factor authentication (MFA) for users logging in through native sign-in. Users must provide an MFA code in addition to their password.
          </p>
          <p className="text-[11px] text-gray-400 mt-2">
            Note: Users are given a 24-hour grace period to configure MFA after this setting is enabled. After this period, accounts without MFA configured will be locked.
          </p>
        </div>
      </div>
    </div>
  )
}

function Toggle({ enabled, onChange }: { enabled: boolean; onChange: (v: boolean) => void }) {
  return (
    <button
      type="button"
      onClick={() => onChange(!enabled)}
      className={`relative w-10 h-5 rounded-full transition-colors ${enabled ? 'bg-orange-500' : 'bg-gray-300'}`}
    >
      <span
        className={`absolute top-0.5 left-0.5 w-4 h-4 bg-white rounded-full shadow transition-transform ${enabled ? 'translate-x-5' : 'translate-x-0'}`}
      />
    </button>
  )
}

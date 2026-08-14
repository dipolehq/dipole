import { useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { QRCodeSVG } from 'qrcode.react'
import { apiClient } from '@/shared/lib/api-client'

interface MFAStatus {
  configured: boolean
  enabled: boolean
  hasBackupCodes: boolean
  createdAt?: string
}

interface SetupResponse {
  secret: string
  uri: string
}

export function MFAConfigTab() {
  const queryClient = useQueryClient()
  const [setupData, setSetupData] = useState<SetupResponse | null>(null)
  const [verifyCode, setVerifyCode] = useState('')
  const [backupCodes, setBackupCodes] = useState<string[] | null>(null)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const { data: status, isLoading } = useQuery({
    queryKey: ['mfa', 'status'],
    queryFn: () => apiClient.get<MFAStatus>('/profile/mfa/status'),
  })

  async function handleSetup() {
    setError('')
    setLoading(true)
    try {
      const res = await apiClient.post<SetupResponse>('/profile/mfa/setup', {})
      setSetupData(res)
    } catch {
      setError('Failed to set up authenticator.')
    } finally {
      setLoading(false)
    }
  }

  async function handleVerify() {
    setError('')
    if (verifyCode.length !== 6) {
      setError('Enter a 6-digit code from your authenticator app.')
      return
    }
    setLoading(true)
    try {
      await apiClient.post('/profile/mfa/verify', { code: verifyCode })
      setSetupData(null)
      setVerifyCode('')
      queryClient.invalidateQueries({ queryKey: ['mfa', 'status'] })
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Invalid code. Try again.')
    } finally {
      setLoading(false)
    }
  }

  async function handleGenerateBackupCodes() {
    setError('')
    setLoading(true)
    try {
      const res = await apiClient.post<{ codes: string[] }>('/profile/mfa/backup-codes', {})
      setBackupCodes(res.codes)
      queryClient.invalidateQueries({ queryKey: ['mfa', 'status'] })
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to generate backup codes.')
    } finally {
      setLoading(false)
    }
  }

  async function handleReset() {
    if (!confirm('This will remove all MFA configuration. You will no longer need a code to sign in. Continue?')) return
    setError('')
    setLoading(true)
    try {
      await apiClient.delete('/profile/mfa')
      setSetupData(null)
      setBackupCodes(null)
      queryClient.invalidateQueries({ queryKey: ['mfa', 'status'] })
    } catch {
      setError('Failed to reset MFA.')
    } finally {
      setLoading(false)
    }
  }

  if (isLoading) return <div className="p-5 text-sm text-gray-400">Loading...</div>

  // Setup flow — show QR code and verification input
  if (setupData) {
    return (
      <div className="space-y-5">
        <div className="border border-border rounded-lg p-5">
          <h4 className="text-sm font-semibold text-gray-800 mb-4">Set Up Authenticator App</h4>
          <p className="text-xs text-gray-600 mb-4">
            Scan this QR code with your authenticator app (Google Authenticator, Authy, 1Password, etc.)
          </p>

          <div className="flex justify-center mb-4">
            <div className="p-3 bg-white border border-gray-200 rounded-lg">
              <QRCodeSVG value={setupData.uri} size={180} />
            </div>
          </div>

          <div className="mb-4">
            <p className="text-xs text-gray-500 mb-1">Or enter this secret manually:</p>
            <code className="block text-xs bg-gray-50 border border-gray-200 rounded p-2 font-mono select-all break-all">
              {setupData.secret}
            </code>
          </div>

          <div className="space-y-3">
            <p className="text-xs text-gray-600 font-medium">Enter the 6-digit code from your app to verify:</p>
            <input
              type="text"
              maxLength={6}
              placeholder="000000"
              value={verifyCode}
              onChange={(e) => setVerifyCode(e.target.value.replace(/\D/g, '').slice(0, 6))}
              className="w-full px-4 py-3 border border-gray-300 rounded-md text-sm text-center tracking-[0.5em] font-mono focus:outline-none focus:ring-2 focus:ring-gray-400 focus:border-transparent"
            />

            {error && <p className="text-xs text-red-600">{error}</p>}

            <div className="flex gap-2">
              <button
                type="button"
                onClick={handleVerify}
                disabled={loading || verifyCode.length !== 6}
                className="flex-1 py-2 bg-gray-800 hover:bg-gray-900 text-white text-xs font-medium rounded-md transition-colors disabled:opacity-50"
              >
                {loading ? 'Verifying…' : 'Verify & Enable'}
              </button>
              <button
                type="button"
                onClick={() => { setSetupData(null); setVerifyCode(''); setError('') }}
                className="px-4 py-2 border border-gray-300 text-gray-600 text-xs font-medium rounded-md hover:bg-gray-50"
              >
                Cancel
              </button>
            </div>
          </div>
        </div>
      </div>
    )
  }

  // Backup codes display (one-time view after generation)
  if (backupCodes) {
    return (
      <div className="space-y-5">
        <div className="border border-border rounded-lg p-5">
          <h4 className="text-sm font-semibold text-gray-800 mb-2">Backup Codes</h4>
          <div className="bg-yellow-50 border border-yellow-200 rounded-md p-3 mb-4">
            <p className="text-xs text-yellow-800">
              Save these codes in a safe place. Each code can only be used once. They will not be shown again.
            </p>
          </div>

          <div className="grid grid-cols-2 gap-2 mb-4">
            {backupCodes.map((code, i) => (
              <code key={i} className="text-xs bg-gray-50 border border-gray-200 rounded p-2 font-mono text-center select-all">
                {code}
              </code>
            ))}
          </div>

          <button
            type="button"
            onClick={() => setBackupCodes(null)}
            className="w-full py-2 bg-gray-800 hover:bg-gray-900 text-white text-xs font-medium rounded-md transition-colors"
          >
            I've saved these codes
          </button>
        </div>
      </div>
    )
  }

  // Main status view
  return (
    <div className="space-y-5">
      <div className="border border-border rounded-lg p-5">
        <div className="flex items-center justify-between mb-4">
          <h4 className="text-sm font-semibold text-gray-800">MFA Status</h4>
          <span className={`text-xs font-medium px-2 py-0.5 rounded ${status?.enabled ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-500'}`}>
            {status?.enabled ? 'CONFIGURED' : 'NOT CONFIGURED'}
          </span>
        </div>

        {status?.enabled && (
          <button
            type="button"
            onClick={handleReset}
            disabled={loading}
            className="mb-4 px-3 py-1.5 border border-red-300 text-red-600 text-xs font-medium rounded-md hover:bg-red-50 transition-colors disabled:opacity-50"
          >
            Reset MFA
          </button>
        )}

        <hr className="border-border mb-4" />

        <h4 className="text-xs font-semibold text-gray-600 uppercase tracking-wider mb-3">Multi-Factor Authentication Types</h4>
        <p className="text-xs text-gray-500 mb-4">
          MFA Types must be registered so that you can provide a secondary means of verifying your identity, in addition to your password.
        </p>

        <table className="w-full text-xs">
          <thead>
            <tr className="bg-gray-50 border-y border-border">
              <th className="px-3 py-2 text-left text-gray-600 font-semibold">MFA Type</th>
              <th className="px-3 py-2 text-right text-gray-600 font-semibold">Status</th>
            </tr>
          </thead>
          <tbody>
            <tr className="border-b border-border">
              <td className="px-3 py-3 text-gray-700">Authenticator App</td>
              <td className="px-3 py-3 text-right">
                {status?.enabled ? (
                  <span className="text-green-600 font-medium">Active</span>
                ) : (
                  <button type="button" onClick={handleSetup} disabled={loading} className="text-gray-600 hover:text-gray-700 font-medium">
                    + Add Authenticator App
                  </button>
                )}
              </td>
            </tr>
            <tr>
              <td className="px-3 py-3 text-gray-700">Backup Codes</td>
              <td className="px-3 py-3 text-right">
                {status?.hasBackupCodes ? (
                  <span className="text-green-600 font-medium">Generated</span>
                ) : status?.enabled ? (
                  <button type="button" onClick={handleGenerateBackupCodes} disabled={loading} className="text-gray-600 hover:text-gray-700 font-medium">
                    + Add Backup Codes
                  </button>
                ) : (
                  <span className="text-gray-400">Set up authenticator first</span>
                )}
              </td>
            </tr>
          </tbody>
        </table>

        {error && <p className="mt-3 text-xs text-red-600">{error}</p>}
      </div>
    </div>
  )
}

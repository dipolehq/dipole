import { useState } from 'react'
import { Download } from 'lucide-react'
import { Dropdown } from '@/shared/components/Dropdown'

const LEAF_CERT_LIFETIME_OPTIONS = [
  { value: '1_hour', label: '1 Hour' },
  { value: '1_day', label: '1 Day' },
  { value: '1_week', label: '1 Week' },
]

export function TlsDecryptPanel() {
  const [leafLifetime, setLeafLifetime] = useState('1_day')

  function handleSave() {
    console.info('[tls-decrypt] saved', { leafLifetime })
  }

  function handleDownloadRootCA() {
    console.info('[tls-decrypt] download root CA')
  }

  return (
    <div className="p-6">
      <div className="border border-border rounded-lg p-6">
        <div className="flex items-start justify-between">
          <div>
            <h2 className="text-base font-semibold text-gray-900 mb-3">TLS Decrypt Root CA</h2>
            <p className="text-xs text-gray-600 mb-2">
              Dipole provides the Root CA certificate for your tenant in support of the TLS Decrypt feature.
            </p>
            <p className="text-xs text-gray-600 max-w-3xl">
              When you enable TLS Decrypt for your Dipole-enabled Client Workloads, Dipole Agent Proxy can decrypt and manage encrypted traffic (such as HTTPS) to and from the configured Client and Server Workloads.
            </p>
          </div>

          <a href="#" className="flex items-center gap-1.5 text-xs text-orange-500 hover:text-orange-600 font-medium shrink-0">
            <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" /></svg>
            Documentation
          </a>
        </div>

        <div className="mt-6 flex items-end gap-4">
          <div>
            <label className="block text-xs text-gray-600 mb-1.5 font-medium">Leaf Certificate Lifetime options</label>
            <div className="w-40">
              <Dropdown
                options={LEAF_CERT_LIFETIME_OPTIONS}
                value={leafLifetime}
                onChange={setLeafLifetime}
                placeholder="Select lifetime..."
              />
            </div>
          </div>

          <button
            type="button"
            onClick={handleSave}
            className="px-4 py-2 border border-orange-500 text-orange-500 hover:bg-orange-50 text-xs font-medium rounded-md transition-colors"
          >
            Save
          </button>

          <button
            type="button"
            onClick={handleDownloadRootCA}
            className="flex items-center gap-1.5 px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors"
          >
            <Download size={12} />
            Download Tenant Root CA
          </button>
        </div>
      </div>
    </div>
  )
}

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
    <>
      {/* Toolbar */}
      <div className="flex items-center justify-between gap-3 px-4 py-3 shrink-0">
        <h2 className="text-sm font-medium text-gray-900">TLS Decrypt</h2>
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={handleSave}
            className="inline-flex items-center gap-2 h-9 px-4 rounded-md border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors"
          >
            Save
          </button>
          <button
            type="button"
            onClick={handleDownloadRootCA}
            className="inline-flex items-center gap-2 h-9 px-4 rounded-md border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors"
          >
            <Download size={14} strokeWidth={2} className="opacity-60" />
            Download Root CA
          </button>
        </div>
      </div>

      {/* Content box */}
      <div className="flex-1 min-h-0 mx-4 border border-border bg-white overflow-y-auto">
        <div className="grid grid-cols-[1fr_380px] h-full">
          <div className="px-6 py-5 overflow-y-auto border-r border-border">
            <div className="space-y-6">
              <div>
                <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider mb-4">Root CA Certificate</h4>
                <p className="text-xs text-gray-600 leading-relaxed mb-4">
                  Dipole provides the Root CA certificate for your tenant in support of the TLS Decrypt feature. When you enable TLS Decrypt, the Agent Proxy can decrypt and manage encrypted traffic (such as HTTPS) to and from configured workloads.
                </p>
              </div>

              <div>
                <h4 className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider mb-4">Leaf Certificate Lifetime</h4>
                <div className="w-48">
                  <Dropdown
                    options={LEAF_CERT_LIFETIME_OPTIONS}
                    value={leafLifetime}
                    onChange={setLeafLifetime}
                    placeholder="Select lifetime..."
                  />
                </div>
                <p className="text-[11px] text-gray-400 mt-2">
                  Controls how long generated leaf certificates remain valid before requiring renewal.
                </p>
              </div>
            </div>
          </div>

          <div className="px-5 py-5 overflow-y-auto">
            <div className="space-y-3">
              <p className="text-xs text-gray-600 leading-relaxed">
                <span className="font-semibold text-gray-800">TLS Decrypt</span> enables the Agent Proxy to terminate TLS connections from client workloads, inspect/modify HTTP requests (e.g., inject credentials), and re-encrypt to the server.
              </p>
              <p className="text-xs text-gray-600 leading-relaxed">
                The <span className="font-semibold text-gray-800">Root CA</span> must be trusted by your client workloads. It is automatically injected into pods by the Agent Injector's init container.
              </p>
              <p className="text-xs text-gray-600 leading-relaxed">
                <span className="font-semibold text-gray-800">Leaf certificates</span> are generated per-hostname on the fly. Shorter lifetimes are more secure but may increase CPU usage on the proxy.
              </p>
            </div>
          </div>
        </div>
      </div>
    </>
  )
}

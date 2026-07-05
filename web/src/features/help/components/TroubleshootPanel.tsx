import { useState } from 'react'
import { CheckCircle, Info, XCircle, ArrowLeftRight, Monitor, Shield, Key, Server } from 'lucide-react'
import { Dropdown } from '@/shared/components/Dropdown'
import { CLIENT_WORKLOAD_OPTIONS, SERVER_WORKLOAD_OPTIONS, getTroubleshootResult } from '@/data/help/troubleshoot'
import type { TroubleshootResult } from '@/data/help/troubleshoot'

export function TroubleshootPanel() {
  const [clientWorkload, setClientWorkload] = useState('')
  const [serverWorkload, setServerWorkload] = useState('')
  const [result, setResult] = useState<TroubleshootResult | null>(null)

  function handleAnalyze() {
    if (!clientWorkload || !serverWorkload) return
    setResult(getTroubleshootResult(clientWorkload, serverWorkload))
  }

  return (
    <div className="p-5 space-y-5">
      {/* Selection area */}
      <div className="flex items-end gap-4">
        <div className="flex-1">
          <label className="block text-xs text-gray-500 mb-1">Client Workload</label>
          <Dropdown
            options={CLIENT_WORKLOAD_OPTIONS}
            value={clientWorkload}
            onChange={setClientWorkload}
            placeholder="Select Client Workload..."
          />
        </div>
        <div className="flex-1">
          <label className="block text-xs text-gray-500 mb-1">Server Workload</label>
          <Dropdown
            options={SERVER_WORKLOAD_OPTIONS}
            value={serverWorkload}
            onChange={setServerWorkload}
            placeholder="Select Server Workload..."
          />
        </div>
        <button
          type="button"
          onClick={handleAnalyze}
          disabled={!clientWorkload || !serverWorkload}
          className="px-5 py-2 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
        >
          Analyze
        </button>
      </div>

      {/* Results — two column layout */}
      {result && (
        <div className="grid grid-cols-[2fr_3fr] gap-5">
          {/* Left column — Troubleshooting Checks */}
          <div>
            <h4 className="text-xs font-bold text-gray-600 uppercase tracking-wider mb-3">Troubleshooting Checks</h4>
            <div className="space-y-3">
              {/* Access Policy Checks */}
              <div className="border border-border rounded-lg p-4">
                <div className="flex items-center gap-2 mb-3">
                  <ArrowLeftRight size={14} className="text-gray-500" />
                  <span className="text-sm font-semibold text-gray-900">Access Policy Checks</span>
                </div>
                <div className="flex items-center gap-3 mb-3">
                  <span className="text-xs text-gray-700 font-medium">{result.accessPolicy.clientName}</span>
                  <span className="text-gray-400">↔</span>
                  <span className="text-xs text-gray-700 font-medium">{result.accessPolicy.serverName}</span>
                </div>
                <div className="flex items-center gap-2 mb-1">
                  {result.accessPolicy.isActive ? (
                    <CheckCircle size={12} className="text-green-500" />
                  ) : (
                    <XCircle size={12} className="text-red-500" />
                  )}
                  <span className="text-xs text-gray-700">{result.accessPolicy.isActive ? 'Active' : 'Inactive'}</span>
                </div>
                <div className="flex items-center gap-2">
                  <Info size={12} className="text-blue-500" />
                  <span className="text-xs text-gray-500">Last Update</span>
                  <span className="text-xs text-gray-700 ml-auto">{result.accessPolicy.lastUpdate}</span>
                </div>
              </div>

              {/* Client Workload Checks */}
              <div className="border border-border rounded-lg p-4">
                <div className="flex items-center gap-2 mb-3">
                  <Monitor size={14} className="text-gray-500" />
                  <span className="text-sm font-semibold text-gray-900">Client Workload Checks</span>
                </div>
                <p className="text-xs font-medium text-gray-800 mb-0.5">{result.clientWorkload.name}</p>
                <p className="text-xs text-gray-500 mb-3">{result.clientWorkload.description}</p>
                <div className="flex items-center gap-2 mb-1">
                  {result.clientWorkload.isActive ? (
                    <CheckCircle size={12} className="text-green-500" />
                  ) : (
                    <XCircle size={12} className="text-red-500" />
                  )}
                  <span className="text-xs text-gray-700">{result.clientWorkload.isActive ? 'Active' : 'Inactive'}</span>
                </div>
                <div className="flex items-center gap-2">
                  <Info size={12} className="text-blue-500" />
                  <span className="text-xs text-gray-500">Last Update</span>
                  <span className="text-xs text-gray-700 ml-auto">{result.clientWorkload.lastUpdate}</span>
                </div>
              </div>

              {/* Trust Provider Checks */}
              <div className="border border-border rounded-lg p-4">
                <div className="flex items-center gap-2 mb-3">
                  <Shield size={14} className="text-gray-500" />
                  <span className="text-sm font-semibold text-gray-900">Trust Provider Checks</span>
                </div>
                {result.trustProvider.exists ? (
                  <>
                    <div className="flex items-center gap-2 mb-3">
                      <CheckCircle size={12} className="text-green-500" />
                      <span className="text-xs text-gray-700">Exists</span>
                    </div>
                    <p className="text-xs font-medium text-gray-800 mb-0.5">{result.trustProvider.name}</p>
                    <p className="text-[11px] text-gray-500 uppercase tracking-wide mb-0.5">{result.trustProvider.type}</p>
                    <p className="text-xs text-gray-500 mb-3">{result.trustProvider.description}</p>
                    <div className="flex items-center gap-2 mb-1">
                      {result.trustProvider.isActive ? (
                        <CheckCircle size={12} className="text-green-500" />
                      ) : (
                        <XCircle size={12} className="text-red-500" />
                      )}
                      <span className="text-xs text-gray-700">{result.trustProvider.isActive ? 'Active' : 'Inactive'}</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <Info size={12} className="text-blue-500" />
                      <span className="text-xs text-gray-500">Last Update</span>
                      <span className="text-xs text-gray-700 ml-auto">{result.trustProvider.lastUpdate}</span>
                    </div>
                  </>
                ) : (
                  <div className="flex items-center gap-2">
                    <XCircle size={12} className="text-red-500" />
                    <span className="text-xs text-gray-700">No Trust Provider configured for this policy.</span>
                  </div>
                )}
              </div>

              {/* Credential Provider Checks */}
              <div className="border border-border rounded-lg p-4">
                <div className="flex items-center gap-2 mb-3">
                  <Key size={14} className="text-gray-500" />
                  <span className="text-sm font-semibold text-gray-900">Credential Provider Checks</span>
                </div>
                {result.credentialProvider.exists ? (
                  <>
                    <div className="flex items-center gap-2 mb-3">
                      <CheckCircle size={12} className="text-green-500" />
                      <span className="text-xs text-gray-700">Exists</span>
                    </div>
                    <p className="text-xs font-medium text-gray-800 mb-0.5">{result.credentialProvider.name}</p>
                    <p className="text-[11px] text-gray-500 uppercase tracking-wide mb-0.5">{result.credentialProvider.type}</p>
                    <p className="text-xs text-gray-500 mb-3">{result.credentialProvider.description}</p>
                    <div className="flex items-center gap-2 mb-1">
                      {result.credentialProvider.isActive ? (
                        <CheckCircle size={12} className="text-green-500" />
                      ) : (
                        <XCircle size={12} className="text-red-500" />
                      )}
                      <span className="text-xs text-gray-700">{result.credentialProvider.isActive ? 'Active' : 'Inactive'}</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <Info size={12} className="text-blue-500" />
                      <span className="text-xs text-gray-500">Last Update</span>
                      <span className="text-xs text-gray-700 ml-auto">{result.credentialProvider.lastUpdate}</span>
                    </div>
                  </>
                ) : (
                  <div className="flex items-center gap-2">
                    <XCircle size={12} className="text-red-500" />
                    <span className="text-xs text-gray-700">No Credential Provider configured.</span>
                  </div>
                )}
              </div>

              {/* Server Workload Checks */}
              <div className="border border-border rounded-lg p-4">
                <div className="flex items-center gap-2 mb-3">
                  <Server size={14} className="text-gray-500" />
                  <span className="text-sm font-semibold text-gray-900">Server Workload Checks</span>
                </div>
                <p className="text-xs font-medium text-gray-800 mb-0.5">{result.serverWorkload.name}</p>
                <p className="text-xs text-gray-500 mb-1">{result.serverWorkload.description}</p>
                <p className="text-xs text-gray-500 mb-3">{result.serverWorkload.host}:{result.serverWorkload.port}</p>
                <div className="flex items-center gap-2 mb-1">
                  {result.serverWorkload.isActive ? (
                    <CheckCircle size={12} className="text-green-500" />
                  ) : (
                    <XCircle size={12} className="text-red-500" />
                  )}
                  <span className="text-xs text-gray-700">{result.serverWorkload.isActive ? 'Active' : 'Inactive'}</span>
                </div>
                <div className="flex items-center gap-2">
                  <Info size={12} className="text-blue-500" />
                  <span className="text-xs text-gray-500">Last Update</span>
                  <span className="text-xs text-gray-700 ml-auto">{result.serverWorkload.lastUpdate}</span>
                </div>
              </div>
            </div>
          </div>

          {/* Right column — Event Traffic Details */}
          <div>
            <div className="flex items-center justify-between mb-3">
              <h4 className="text-xs font-bold text-gray-600 uppercase tracking-wider">Event Traffic Details (Last 24 hrs)</h4>
              <span className="text-[11px] text-gray-400">a few seconds ago</span>
            </div>
            <div className="border border-border rounded-lg overflow-hidden">
              <table className="w-full text-xs">
                <thead>
                  <tr className="bg-gray-50 border-b border-border">
                    <th className="px-4 py-2 text-left font-semibold text-gray-600">Outcome Reason</th>
                    <th className="px-4 py-2 text-center font-semibold text-gray-600">Count</th>
                    <th className="px-4 py-2 text-center font-semibold text-gray-600">Severity</th>
                  </tr>
                </thead>
                <tbody>
                  {result.events.length === 0 ? (
                    <tr>
                      <td colSpan={3} className="px-4 py-4">
                        <div className="flex items-center gap-2">
                          <Info size={12} className="text-blue-500" />
                          <span className="text-xs text-gray-500">No events produced between selected workloads.</span>
                        </div>
                      </td>
                    </tr>
                  ) : (
                    result.events.map((event, idx) => (
                      <tr key={idx} className="border-b border-border last:border-b-0">
                        <td className="px-4 py-2 text-gray-700">{event.outcomeReason}</td>
                        <td className="px-4 py-2 text-center text-gray-700">{event.count}</td>
                        <td className="px-4 py-2 text-center">
                          <SeverityBadge severity={event.severity} />
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      )}

      {/* Empty state before analysis */}
      {!result && (
        <div className="flex items-center justify-center py-16">
          <p className="text-sm text-gray-400">Select workloads above and click Analyze to run diagnostics.</p>
        </div>
      )}
    </div>
  )
}

function SeverityBadge({ severity }: { severity: string }) {
  const config: Record<string, { color: string }> = {
    Info: { color: 'text-green-600' },
    Warn: { color: 'text-yellow-600' },
    Error: { color: 'text-red-600' },
  }
  const c = config[severity] ?? { color: 'text-gray-600' }
  return <span className={`text-xs font-medium ${c.color}`}>{severity}</span>
}

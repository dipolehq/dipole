import { Info } from 'lucide-react'

export function TroubleshootPanel() {
  return (
    <div className="p-5 flex flex-col items-center justify-center py-16">
      <Info size={32} className="text-gray-300 mb-3" />
      <h3 className="text-sm font-medium text-gray-600 mb-1">Troubleshooting</h3>
      <p className="text-xs text-gray-400">Coming soon — this diagnostic tool will be wired to live data.</p>
    </div>
  )
}

import { useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Search, Filter } from 'lucide-react'
import { PageHeader } from '@/shared/components/PageHeader'
import { Dropdown } from '@/shared/components/Dropdown'
import { GlobalPolicyComplianceTable } from '@/features/reporting/components/GlobalPolicyComplianceTable'
import { COMPLIANCE_STATUS_OPTIONS, RESOURCE_SET_OPTIONS } from '@/shared/constants/reporting'

export function ReportingCompliancePage() {
  const [params, setParams] = useSearchParams()

  const searchQuery = params.get('search') || ''
  const complianceStatus = params.get('complianceStatus') || ''
  const resourceSet = params.get('resourceSet') || ''

  function setParam(key: string, value: string) {
    setParams((prev) => {
      const next = new URLSearchParams(prev)
      if (value) next.set(key, value)
      else next.delete(key)
      return next
    }, { replace: true })
  }

  const [showFilter, setShowFilter] = useState(false)

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-4 shrink-0 h-[60px] border-b border-border bg-white">
        <PageHeader />
        <div className="flex items-center gap-3">
          <div className="relative">
            <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
            <input type="text" placeholder="Search" value={searchQuery} onChange={(e) => setParam('search', e.target.value)} className="pl-9 pr-3 py-2 border border-border rounded-md text-sm w-48 focus:outline-none focus:ring-1 focus:ring-gray-300 focus:border-gray-400" />
          </div>
          <div className="relative">
            <button type="button" onClick={() => setShowFilter(!showFilter)} className={`flex items-center gap-1.5 px-4 py-2 border rounded-md text-sm font-medium transition-colors ${showFilter ? 'border-orange-400 text-orange-600 bg-orange-50' : 'border-border text-gray-600 hover:border-gray-400'}`}>
              <Filter size={14} /> Filter
            </button>
            {showFilter && (
              <div className="absolute top-full right-0 mt-1 w-64 p-3 bg-white border border-border rounded-lg shadow-lg z-50">
                <div className="space-y-3">
                  <div>
                    <label className="block text-[11px] text-gray-500 mb-1">Resource Set</label>
                    <Dropdown options={RESOURCE_SET_OPTIONS.map((r) => ({ value: r.value, label: r.label }))} value={resourceSet} onChange={(v) => setParam('resourceSet', v)} placeholder="Resource Set" />
                  </div>
                  <div>
                    <label className="block text-[11px] text-gray-500 mb-1">Compliance Status</label>
                    <Dropdown options={COMPLIANCE_STATUS_OPTIONS.map((c) => ({ value: c.value, label: c.label }))} value={complianceStatus} onChange={(v) => setParam('complianceStatus', v)} placeholder="Status" />
                  </div>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto min-h-0">
        <GlobalPolicyComplianceTable searchQuery={searchQuery} complianceStatus={complianceStatus} resourceSet={resourceSet} />
      </div>
    </div>
  )
}

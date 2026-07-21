import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Plus, Search, FileText } from 'lucide-react'
import { PageHeader } from '@/shared/components/PageHeader'
import { AgentControllersTable } from '@/features/edge-components/components/AgentControllersTable'
import { DeployEdgePanel } from '@/features/edge-components/components/DeployEdgePanel'
import { TlsDecryptPanel } from '@/features/edge-components/components/TlsDecryptPanel'
import { useAgentControllers } from '@/features/edge-components/hooks/useAgentControllers'
import { PageSizeSelect } from '@/shared/components/PageSizeSelect'

const EDGE_SECTIONS = [
  { value: 'agent_controllers', label: 'Agent Controllers' },
  { value: 'deploy_edge', label: 'Deploy Dipole Edge' },
  { value: 'tls_decrypt', label: 'TLS Decrypt' },
] as const

type EdgeSection = (typeof EDGE_SECTIONS)[number]['value']

const PAGE_SIZE_OPTIONS = [10, 25, 50] as const

export function EdgeComponentsPage() {
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedSection, setSelectedSection] = useState<EdgeSection>('agent_controllers')
  const [sectionOpen, setSectionOpen] = useState(false)
  const [pageSize, setPageSize] = useState<number>(PAGE_SIZE_OPTIONS[0])
  const [currentPage, setCurrentPage] = useState(1)

  const { data, isLoading } = useAgentControllers(currentPage, pageSize, searchQuery)
  const items = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const selectedLabel = EDGE_SECTIONS.find((s) => s.value === selectedSection)?.label ?? ''
  const showPagination = selectedSection === 'agent_controllers'
  const showNewButton = selectedSection === 'agent_controllers'

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-4 shrink-0 h-[60px] border-b border-border bg-white">
        <PageHeader />
        <div className="flex items-center gap-3">
          <div className="relative">
            <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
            <input
              type="text"
              placeholder="Search"
              value={searchQuery}
              onChange={(e) => { setSearchQuery(e.target.value); setCurrentPage(1) }}
              className="pl-9 pr-3 py-2 border border-border rounded-md text-sm w-48 focus:outline-none focus:ring-1 focus:ring-gray-300 focus:border-gray-400"
            />
          </div>
          {showNewButton && (
            <Link to="/edge-components/agent_controllers/new" className="flex items-center gap-1.5 px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white text-sm font-medium rounded-md transition-colors">
              <Plus size={14} /> New
            </Link>
          )}
          <div className="relative">
            <button type="button" onClick={() => setSectionOpen(!sectionOpen)} className="flex items-center gap-1.5 w-[180px] px-4 py-2 bg-[#28282B] hover:bg-[#333336] text-white text-sm font-medium rounded-md transition-colors">
              <FileText size={14} className="shrink-0" />
              <span className="truncate">{selectedLabel}</span>
              <svg className={`w-3.5 h-3.5 ml-auto shrink-0 transition-transform ${sectionOpen ? 'rotate-180' : ''}`} fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" /></svg>
            </button>
            {sectionOpen && (
              <div className="absolute top-full right-0 mt-1 bg-white border border-border rounded-md shadow-lg z-50 w-[220px] overflow-hidden">
                {EDGE_SECTIONS.map((opt) => (
                  <button key={opt.value} type="button" onClick={() => { setSelectedSection(opt.value); setSectionOpen(false) }} className={`w-full text-left px-4 py-2.5 text-sm transition-colors ${selectedSection === opt.value ? 'bg-orange-50 text-orange-600 font-medium' : 'text-gray-600 hover:bg-gray-50'}`}>{opt.label}</button>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto min-h-0">
        {selectedSection === 'agent_controllers' ? (
          <AgentControllersTable items={items} isLoading={isLoading} />
        ) : selectedSection === 'deploy_edge' ? (
          <DeployEdgePanel />
        ) : selectedSection === 'tls_decrypt' ? (
          <TlsDecryptPanel />
        ) : null}
      </div>

      {showPagination && (
        <div className="flex items-center justify-between px-4 py-2 border-t border-border bg-gray-50 shrink-0">
          <div className="flex items-center gap-2 text-xs text-gray-600">
            <PageSizeSelect value={pageSize} options={PAGE_SIZE_OPTIONS} onChange={(v) => { setPageSize(v); setCurrentPage(1) }} />
            <span>Showing {total === 0 ? 0 : (currentPage - 1) * pageSize + 1} - {Math.min(currentPage * pageSize, total)} of {total} Results</span>
          </div>
          <div className="flex items-center gap-1">
            <button onClick={() => setCurrentPage((p) => Math.max(1, p - 1))} disabled={currentPage === 1} className="px-2 py-1 text-xs text-gray-500 hover:text-gray-700 disabled:opacity-30">←</button>
            {Array.from({ length: totalPages }, (_, i) => (
              <button key={i + 1} onClick={() => setCurrentPage(i + 1)} className={`w-7 h-7 rounded text-xs font-medium transition-colors ${currentPage === i + 1 ? 'bg-orange-500 text-white' : 'text-gray-600 hover:bg-gray-100'}`}>{i + 1}</button>
            ))}
            <button onClick={() => setCurrentPage((p) => Math.min(totalPages, p + 1))} disabled={currentPage === totalPages} className="px-2 py-1 text-xs text-gray-500 hover:text-gray-700 disabled:opacity-30">→</button>
          </div>
        </div>
      )}
    </div>
  )
}

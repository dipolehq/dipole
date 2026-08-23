import { useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { Plus, Search, FileText } from 'lucide-react'
import { usePageHeaderActions } from '@/shared/context/PageHeaderActionsContext'
import { AgentControllersTable } from '@/features/edge-components/components/AgentControllersTable'
import { AgentProxiesTable } from '@/features/edge-components/components/AgentProxiesTable'
import { DeployEdgePanel } from '@/features/edge-components/components/DeployEdgePanel'
import { TlsDecryptPanel } from '@/features/edge-components/components/TlsDecryptPanel'
import { useAgentControllers } from '@/features/edge-components/hooks/useAgentControllers'
import { useAgentProxies } from '@/features/edge-components/hooks/useAgentProxies'
import { PageSizeSelect } from '@/shared/components/PageSizeSelect'

const EDGE_SECTIONS = [
  { value: 'agent_controllers', label: 'Agent Controllers' },
  { value: 'agent_proxies', label: 'Agent Proxies' },
  { value: 'deploy_edge', label: 'Deploy Dipole Edge' },
  { value: 'tls_decrypt', label: 'TLS Decrypt' },
] as const

type EdgeSection = (typeof EDGE_SECTIONS)[number]['value']

const PAGE_SIZE_OPTIONS = [10, 25, 50] as const

export function EdgeComponentsPage() {
  const [params, setParams] = useSearchParams()

  const [searchQuery, setSearchQuery] = useState(params.get('search') || '')
  const [selectedSection, setSelectedSectionState] = useState<EdgeSection>((params.get('tab') as EdgeSection) || 'agent_controllers')
  const [sectionOpen, setSectionOpen] = useState(false)
  const [pageSize, setPageSizeState] = useState<number>(Number(params.get('perPage')) || PAGE_SIZE_OPTIONS[0])
  const [currentPage, setCurrentPageState] = useState(Number(params.get('page')) || 1)

  function setSelectedSection(v: EdgeSection) { setSelectedSectionState(v); setCurrentPageState(1); setParams((prev) => { const next = new URLSearchParams(prev); next.set('tab', v); next.set('page', '1'); return next }, { replace: true }) }
  function setCurrentPage(p: number) { setCurrentPageState(p); setParams((prev) => { const next = new URLSearchParams(prev); next.set('page', String(p)); return next }, { replace: true }) }
  function setPageSize(pp: number) { setPageSizeState(pp); setCurrentPageState(1); setParams((prev) => { const next = new URLSearchParams(prev); next.set('perPage', String(pp)); next.set('page', '1'); return next }, { replace: true }) }

  const { data, isLoading } = useAgentControllers(currentPage, pageSize, searchQuery)
  const items = data?.items ?? []
  const total = data?.total ?? 0

  const { data: proxyData, isLoading: proxyLoading } = useAgentProxies(currentPage, pageSize, searchQuery)
  const proxyItems = proxyData?.items ?? []
  const proxyTotal = proxyData?.total ?? 0

  const activeTotal = selectedSection === 'agent_proxies' ? proxyTotal : total
  const totalPages = Math.max(1, Math.ceil(activeTotal / pageSize))

  const selectedLabel = EDGE_SECTIONS.find((s) => s.value === selectedSection)?.label ?? ''
  const showPagination = selectedSection === 'agent_controllers' || selectedSection === 'agent_proxies'
  const showNewButton = selectedSection === 'agent_controllers'

  usePageHeaderActions(
    <div className="flex items-center gap-3">
      <div className="relative">
        <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
        <input
          type="text"
          placeholder="Search"
          value={searchQuery}
          onChange={(e) => { setSearchQuery(e.target.value); setCurrentPageState(1); setParams((prev) => { const next = new URLSearchParams(prev); if (e.target.value) next.set('search', e.target.value); else next.delete('search'); next.set('page', '1'); return next }, { replace: true }) }}
          className="pl-9 pr-3 py-2 border border-border rounded-md text-sm w-48 focus:outline-none focus:ring-1 focus:ring-gray-300 focus:border-gray-400"
        />
      </div>
      {showNewButton && (
        <Link to="/edge-components/agent_controllers/new" className="flex items-center gap-1.5 px-4 py-2 bg-gray-800 hover:bg-gray-900 text-white text-sm font-medium rounded-md transition-colors">
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
              <button key={opt.value} type="button" onClick={() => { setSelectedSection(opt.value); setSectionOpen(false) }} className={`w-full text-left px-4 py-2.5 text-sm transition-colors ${selectedSection === opt.value ? 'bg-gray-50 text-gray-700 font-medium' : 'text-gray-600 hover:bg-gray-50'}`}>{opt.label}</button>
            ))}
          </div>
        )}
      </div>
    </div>,
  )

  return (
    <div className="flex flex-col h-full">
      <div className="flex-1 overflow-y-auto min-h-0">
        {selectedSection === 'agent_controllers' ? (
          <AgentControllersTable items={items} isLoading={isLoading} />
        ) : selectedSection === 'agent_proxies' ? (
          <AgentProxiesTable items={proxyItems} isLoading={proxyLoading} />
        ) : selectedSection === 'deploy_edge' ? (
          <DeployEdgePanel />
        ) : selectedSection === 'tls_decrypt' ? (
          <TlsDecryptPanel />
        ) : null}
      </div>

      {showPagination && (
        <div className="flex items-center justify-between px-4 py-2 border-t border-border bg-gray-50 shrink-0">
          <div className="flex items-center gap-2 text-xs text-gray-600">
            <PageSizeSelect value={pageSize} options={PAGE_SIZE_OPTIONS} onChange={(v) => { setPageSize(v) }} />
            <span>Showing {activeTotal === 0 ? 0 : (currentPage - 1) * pageSize + 1} - {Math.min(currentPage * pageSize, activeTotal)} of {activeTotal} Results</span>
          </div>
          <div className="flex items-center gap-1">
            <button onClick={() => setCurrentPage(Math.max(1, currentPage - 1))} disabled={currentPage === 1} className="px-2 py-1 text-xs text-gray-500 hover:text-gray-700 disabled:opacity-30">←</button>
            {Array.from({ length: totalPages }, (_, i) => (
              <button key={i + 1} onClick={() => setCurrentPage(i + 1)} className={`w-7 h-7 rounded text-xs font-medium transition-colors ${currentPage === i + 1 ? 'bg-gray-800 text-white' : 'text-gray-600 hover:bg-gray-100'}`}>{i + 1}</button>
            ))}
            <button onClick={() => setCurrentPage(Math.min(totalPages, currentPage + 1))} disabled={currentPage === totalPages} className="px-2 py-1 text-xs text-gray-500 hover:text-gray-700 disabled:opacity-30">→</button>
          </div>
        </div>
      )}
    </div>
  )
}

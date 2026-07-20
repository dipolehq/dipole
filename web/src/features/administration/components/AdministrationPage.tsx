import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Plus, Search, FileText } from 'lucide-react'
import { PageHeader } from '@/shared/components/PageHeader'
import { UsersTable } from '@/features/administration/components/UsersTable'
import { RolesTable } from '@/features/administration/components/RolesTable'
import { ResourceSetsTable } from '@/features/administration/components/ResourceSetsTable'
import { IdentityProvidersTable } from '@/features/administration/components/IdentityProvidersTable'
import { LogStreamsTable } from '@/features/administration/components/LogStreamsTable'
import { GlobalPolicySettingsPanel } from '@/features/administration/components/GlobalPolicySettingsPanel'
import { SignOnPolicyPanel } from '@/features/administration/components/SignOnPolicyPanel'
import { useUsers } from '@/features/administration/hooks/useUsers'
import { useRoles } from '@/features/administration/hooks/useRoles'
import { useResourceSets } from '@/features/administration/hooks/useResourceSets'
import { useIdentityProviders } from '@/features/administration/hooks/useIdentityProviders'
import { logStreamsData } from '@/data/administration/logStreams'

const ADMIN_SECTIONS = [
  { value: 'users', label: 'Users' },
  { value: 'roles', label: 'Roles' },
  { value: 'resource_sets', label: 'Resource Sets' },
  { value: 'identity_providers', label: 'Identity Providers' },
  { value: 'sign_on_policy', label: 'Sign-On Policy' },
  { value: 'log_streams', label: 'Log Streams' },
  { value: 'global_policy_compliance', label: 'Global Policy Compliance' },
] as const

type AdminSection = (typeof ADMIN_SECTIONS)[number]['value']

export function AdministrationPage() {
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedSection, setSelectedSection] = useState<AdminSection>('users')
  const [sectionOpen, setSectionOpen] = useState(false)
  const [pageSize, setPageSize] = useState(10)
  const [currentPage, setCurrentPage] = useState(1)

  const { data: usersData, isLoading: usersLoading } = useUsers(currentPage, pageSize, searchQuery)
  const usersItems = usersData?.items ?? []
  const usersTotal = usersData?.total ?? 0

  const { data: rolesApiData, isLoading: rolesLoading } = useRoles(currentPage, pageSize, searchQuery)
  const rolesItems = rolesApiData?.items ?? []
  const rolesTotal = rolesApiData?.total ?? 0

  const { data: resourceSetsData, isLoading: resourceSetsLoading } = useResourceSets(currentPage, pageSize, searchQuery)
  const resourceSetsItems = resourceSetsData?.items ?? []
  const resourceSetsTotal = resourceSetsData?.total ?? 0

  const { data: idpData, isLoading: idpLoading } = useIdentityProviders(currentPage, pageSize, searchQuery)
  const idpItems = idpData?.items ?? []
  const idpTotal = idpData?.total ?? 0

  const selectedLabel = ADMIN_SECTIONS.find((s) => s.value === selectedSection)?.label ?? ''

  const totalItems = selectedSection === 'users'
    ? usersTotal
    : selectedSection === 'roles'
    ? rolesTotal
    : selectedSection === 'resource_sets'
    ? resourceSetsTotal
    : selectedSection === 'identity_providers'
    ? idpTotal
    : selectedSection === 'log_streams'
    ? logStreamsData.filter((ls) => {
        if (!searchQuery) return true
        const q = searchQuery.toLowerCase()
        return ls.name.toLowerCase().includes(q) || ls.description.toLowerCase().includes(q)
      }).length
    : 0
  const totalPages = Math.max(1, Math.ceil(totalItems / pageSize))
  const showPagination = selectedSection !== 'global_policy_compliance' && selectedSection !== 'sign_on_policy'
  const showNewButton = selectedSection !== 'global_policy_compliance' && selectedSection !== 'sign_on_policy'

  return (
    <div className="flex flex-col h-full">
      {/* Header bar */}
      <div className="flex items-center justify-between px-4 py-3 shrink-0 min-h-[36px] border-b border-border bg-white">
        <PageHeader />
        <div className="flex items-center gap-3">
          <div className="relative">
            <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
            <input type="text" placeholder="Search" value={searchQuery} onChange={(e) => setSearchQuery(e.target.value)} className="pl-9 pr-3 py-2 border border-border rounded-md text-sm w-48 focus:outline-none focus:ring-1 focus:ring-gray-300 focus:border-gray-400" />
          </div>
          {showNewButton && (
            <Link to={`/administration/${selectedSection}/new`} className="flex items-center gap-1.5 px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white text-sm font-medium rounded-md transition-colors">
              <Plus size={14} /> New
            </Link>
          )}
          <div className="relative">
            <button type="button" onClick={() => setSectionOpen(!sectionOpen)} className="flex items-center gap-1.5 w-[160px] px-4 py-2 bg-[#28282B] hover:bg-[#333336] text-white text-sm font-medium rounded-md transition-colors">
              <FileText size={14} className="shrink-0" />
              <span className="truncate">{selectedLabel}</span>
              <svg className={`w-3.5 h-3.5 ml-auto shrink-0 transition-transform ${sectionOpen ? 'rotate-180' : ''}`} fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" /></svg>
            </button>
            {sectionOpen && (
              <div className="absolute top-full right-0 mt-1 bg-white border border-border rounded-md shadow-lg z-50 w-[220px] overflow-hidden">
                {ADMIN_SECTIONS.map((opt) => (
                  <button key={opt.value} type="button" onClick={() => { setSelectedSection(opt.value); setSectionOpen(false); setCurrentPage(1); setSearchQuery('') }} className={`w-full text-left px-4 py-2.5 text-sm transition-colors ${selectedSection === opt.value ? 'bg-orange-50 text-orange-600 font-medium' : 'text-gray-600 hover:bg-gray-50'}`}>{opt.label}</button>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Content — full bleed */}
      <div className="flex-1 overflow-y-auto min-h-0">
        {selectedSection === 'users' ? (
          <UsersTable items={usersItems} isLoading={usersLoading} />
        ) : selectedSection === 'roles' ? (
          <RolesTable items={rolesItems} isLoading={rolesLoading} />
        ) : selectedSection === 'resource_sets' ? (
          <ResourceSetsTable items={resourceSetsItems} isLoading={resourceSetsLoading} />
        ) : selectedSection === 'identity_providers' ? (
          <IdentityProvidersTable items={idpItems} isLoading={idpLoading} />
        ) : selectedSection === 'log_streams' ? (
          <LogStreamsTable searchQuery={searchQuery} />
        ) : selectedSection === 'global_policy_compliance' ? (
          <GlobalPolicySettingsPanel />
        ) : selectedSection === 'sign_on_policy' ? (
          <SignOnPolicyPanel />
        ) : null}
      </div>

      {/* Pagination footer */}
      {showPagination && (
        <div className="flex items-center justify-between px-4 py-2 border-t border-border bg-gray-50 shrink-0">
          <div className="flex items-center gap-2 text-xs text-gray-600">
            <select value={pageSize} onChange={(e) => { setPageSize(Number(e.target.value)); setCurrentPage(1) }} className="border border-border rounded px-2 py-1 text-xs">
              {[10, 25, 50].map((size) => <option key={size} value={size}>{size}</option>)}
            </select>
            <span>Showing {totalItems === 0 ? 0 : (currentPage - 1) * pageSize + 1} - {Math.min(currentPage * pageSize, totalItems)} of {totalItems} Results</span>
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

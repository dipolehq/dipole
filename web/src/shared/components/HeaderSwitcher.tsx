import { useState, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { Check, ChevronsUpDown, Plus } from 'lucide-react'
import { useClickOutside } from '@/shared/hooks/useClickOutside'
import { useResourceSetContext } from '@/shared/context/ResourceSetContext'
import { apiClient } from '@/shared/lib/api-client'
import { setCurrentTenantSubdomain } from '@/shared/lib/platform-urls'
import { useQuery } from '@tanstack/react-query'

interface TenantSummary {
  id: string
  name: string
  subdomain: string
}

function getCurrentTenantId(): string {
  try {
    const token = sessionStorage.getItem('dipole_token')
    if (!token) return ''
    const parts = token.split('.')
    const segment = parts[1]
    if (!segment) return ''
    const payload = JSON.parse(atob(segment.replace(/-/g, '+').replace(/_/g, '/')))
    return payload.tenant_id || ''
  } catch { return '' }
}

function getCurrentRoleName(): string {
  try {
    const token = sessionStorage.getItem('dipole_token')
    if (!token) return ''
    const parts = token.split('.')
    const segment = parts[1]
    if (!segment) return ''
    const payload = JSON.parse(atob(segment.replace(/-/g, '+').replace(/_/g, '/')))
    return payload.role_name || ''
  } catch { return '' }
}

/**
 * Neon-style breadcrumb switcher: ResourceSet / Tenant
 * Renders in the top-right header area.
 */
export function HeaderSwitcher() {
  return (
    <div className="flex items-center gap-0">
      <ResourceSetSwitcher />
      <span className="text-gray-300 mx-1 text-sm select-none">/</span>
      <TenantSwitcherInline />
    </div>
  )
}

// --- Resource Set (left side) ---
function ResourceSetSwitcher() {
  const { current, all, setCurrent } = useResourceSetContext()
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)
  const navigate = useNavigate()

  useClickOutside(ref, () => setOpen(false))

  return (
    <div ref={ref} className="relative">
      <button
        onClick={() => setOpen(!open)}
        className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-md text-[13px] font-medium text-gray-700 hover:bg-gray-100 transition-colors"
      >
        <span className="truncate max-w-[140px]">{current.name}</span>
        <ChevronsUpDown size={13} className="text-gray-400 shrink-0" />
      </button>

      {open && (
        <>
          <div className="fixed inset-0 z-40" onClick={() => setOpen(false)} />
          <div className="absolute top-full left-0 mt-1 z-50 min-w-[200px] bg-white rounded-lg shadow-xl border border-gray-200 py-1 max-h-64 overflow-y-auto">
            <div className="px-3 py-2 text-[11px] font-semibold text-gray-400 uppercase tracking-wider">
              Resource Sets
            </div>
            {all.map((rs) => (
              <button
                key={rs.id}
                onClick={() => { setCurrent(rs); setOpen(false) }}
                className={`flex items-center justify-between gap-2 w-full px-3 py-2 text-left text-[13px] transition-colors ${
                  current.id === rs.id
                    ? 'bg-gray-50 text-gray-900 font-medium'
                    : 'text-gray-700 hover:bg-gray-50'
                }`}
              >
                <span className="truncate">{rs.name}</span>
                {current.id === rs.id && <Check size={14} className="text-gray-600 shrink-0" />}
              </button>
            ))}
            <div className="border-t border-gray-100 mt-1 pt-1">
              <button
                onClick={() => { setOpen(false); navigate('/administration/resource_sets/new') }}
                className="flex items-center gap-2 w-full px-3 py-2 text-left text-[13px] text-gray-400 hover:text-gray-600 hover:bg-gray-50 transition-colors"
              >
                <Plus size={13} />
                <span>New Resource Set</span>
              </button>
            </div>
          </div>
        </>
      )}
    </div>
  )
}

// --- Tenant (right side) ---
function TenantSwitcherInline() {
  const [open, setOpen] = useState(false)
  const [switching, setSwitching] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useClickOutside(ref, () => setOpen(false))

  const { data } = useQuery({
    queryKey: ['user', 'tenants'],
    queryFn: () => apiClient.get<{ tenants: TenantSummary[] }>('/user/tenants'),
    staleTime: 120_000,
  })

  const tenants = data?.tenants ?? []
  const currentTenantId = getCurrentTenantId()
  const currentTenant = tenants.find((t) => t.id === currentTenantId)

  // Display tenant name or fallback
  const displayName = currentTenant?.name ?? 'Tenant'
  const displayRole = getCurrentRoleName() || 'Member'

  async function handleSwitch(tenantId: string) {
    if (tenantId === currentTenantId || switching) return
    setSwitching(true)
    setOpen(false)

    try {
      const res = await apiClient.post<{ token: string; tenantSubdomain?: string; user: { id: string; email: string; name: string } }>(
        '/auth/switch-tenant',
        { tenantId },
      )
      sessionStorage.setItem('dipole_token', res.token)
      sessionStorage.setItem('dipole_user', JSON.stringify(res.user))
      sessionStorage.removeItem('dipole_resource_set')
      setCurrentTenantSubdomain(res.tenantSubdomain ?? '')
      window.location.replace('/')
    } catch {
      setSwitching(false)
    }
  }

  return (
    <div ref={ref} className="relative">
      <button
        onClick={() => setOpen(!open)}
        className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-md text-[13px] font-medium text-gray-700 hover:bg-gray-100 transition-colors"
      >
        <span className="truncate max-w-[140px]">{displayName}</span>
        <span className="text-[11px] font-normal text-gray-400 bg-gray-100 px-1.5 py-0.5 rounded">{displayRole}</span>
        <ChevronsUpDown size={13} className="text-gray-400 shrink-0" />
      </button>

      {open && (
        <>
          <div className="fixed inset-0 z-40" onClick={() => setOpen(false)} />
          <div className="absolute top-full right-0 mt-1 z-50 min-w-[220px] bg-white rounded-lg shadow-xl border border-gray-200 py-1 max-h-64 overflow-y-auto">
            <div className="px-3 py-2 text-[11px] font-semibold text-gray-400 uppercase tracking-wider">
              Tenants
            </div>
            {tenants.length === 0 && (
              <div className="px-3 py-2 text-[12px] text-gray-400">
                {displayName}
              </div>
            )}
            {tenants.map((t) => (
              <button
                key={t.id}
                onClick={() => handleSwitch(t.id)}
                disabled={switching}
                className={`flex items-center justify-between gap-2 w-full px-3 py-2 text-left text-[13px] transition-colors ${
                  t.id === currentTenantId
                    ? 'bg-gray-50 text-gray-900 font-medium'
                    : 'text-gray-700 hover:bg-gray-50'
                }`}
              >
                <div className="min-w-0">
                  <div className="truncate">{t.name}</div>
                  <div className="text-[11px] text-gray-400 truncate">{t.subdomain}</div>
                </div>
                {t.id === currentTenantId && <Check size={14} className="text-gray-600 shrink-0" />}
              </button>
            ))}
          </div>
        </>
      )}
    </div>
  )
}

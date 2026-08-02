import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Building2, ChevronDown } from 'lucide-react'
import { apiClient } from '@/shared/lib/api-client'

interface TenantSummary {
  id: string
  name: string
  subdomain: string
}

export function TenantSwitcher() {
  const [open, setOpen] = useState(false)
  const [switching, setSwitching] = useState(false)

  const { data } = useQuery({
    queryKey: ['user', 'tenants'],
    queryFn: () => apiClient.get<{ tenants: TenantSummary[] }>('/user/tenants'),
    staleTime: 120_000,
  })

  const tenants = data?.tenants ?? []

  // Don't render if user belongs to only one tenant.
  if (tenants.length <= 1) return null

  // Determine current tenant from JWT claims in sessionStorage.
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

  const currentTenantId = getCurrentTenantId()
  const currentTenant = tenants.find((t) => t.id === currentTenantId)

  async function handleSwitch(tenantId: string) {
    if (tenantId === currentTenantId || switching) return
    setSwitching(true)
    setOpen(false)

    try {
      const res = await apiClient.post<{ token: string; user: { id: string; email: string; name: string } }>(
        '/auth/switch-tenant',
        { tenantId },
      )
      sessionStorage.setItem('dipole_token', res.token)
      sessionStorage.setItem('dipole_user', JSON.stringify(res.user))
      // Full reload to clear all cached data scoped to the old tenant.
      window.location.replace('/')
    } catch {
      setSwitching(false)
    }
  }

  return (
    <div className="relative">
      <div className="px-3 mb-1.5 text-[10px] font-semibold text-gray-500 uppercase tracking-wider">
        Tenant
      </div>
      <button
        type="button"
        onClick={() => setOpen(!open)}
        className="flex items-center gap-2 w-full px-3 py-2 rounded-md text-sm text-gray-300 hover:bg-white/5 hover:text-white transition-colors"
      >
        <Building2 size={14} className="shrink-0 text-gray-500" />
        <span className="truncate flex-1 text-left text-xs">{currentTenant?.name ?? 'Tenant'}</span>
        <ChevronDown size={12} className={`shrink-0 text-gray-500 transition-transform ${open ? 'rotate-180' : ''}`} />
      </button>

      {open && (
        <div className="absolute top-full left-0 right-0 mt-1 bg-[#333336] rounded-lg shadow-lg z-50 py-1 max-h-48 overflow-y-auto">
          {tenants.map((t) => (
            <button
              key={t.id}
              type="button"
              onClick={() => handleSwitch(t.id)}
              disabled={switching}
              className={`w-full text-left px-4 py-2 text-xs transition-colors ${
                t.id === currentTenantId
                  ? 'text-orange-400 font-medium bg-white/5'
                  : 'text-gray-300 hover:text-white hover:bg-white/5'
              }`}
            >
              <div className="truncate">{t.name}</div>
              <div className="text-[10px] text-gray-500">{t.subdomain}</div>
            </button>
          ))}
        </div>
      )}
    </div>
  )
}

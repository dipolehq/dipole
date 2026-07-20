import { Fingerprint } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import type { IdpListItem } from '@/features/administration/api/identityProvidersApi'

interface IdentityProvidersTableProps {
  items: IdpListItem[]
  isLoading?: boolean
}

export function IdentityProvidersTable({ items, isLoading }: IdentityProvidersTableProps) {
  const navigate = useNavigate()

  return (
    <table className="w-full text-xs table-fixed">
      <colgroup>
        <col className="w-[4%]" />
        <col className="w-[22%]" />
        <col className="w-[14%]" />
        <col className="w-[48%]" />
        <col className="w-[12%]" />
      </colgroup>
      <thead className="sticky top-0 bg-gray-50 z-10">
        <tr className="border-b border-border">
          <th className="px-3 py-3" />
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Name</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Type</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Description</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-center">Status</th>
        </tr>
      </thead>
      <tbody className="divide-y divide-border">
        {isLoading && (
          <tr>
            <td colSpan={5} className="px-4 py-12 text-center text-gray-400 text-sm" />
          </tr>
        )}
        {!isLoading && items.length === 0 && (
          <tr>
            <td colSpan={5} className="px-4 py-12 text-center text-gray-400 text-sm">
              No identity providers found.
            </td>
          </tr>
        )}
        {!isLoading &&
          items.map((idp) => (
            <tr
              key={idp.id}
              className="hover:bg-gray-50 transition-colors cursor-pointer h-[41px]"
              onClick={() => navigate(`/administration/identity_providers/${idp.id}`)}
            >
              <td className="px-3 py-3 text-center">
                <Fingerprint size={14} className="text-gray-400" />
              </td>
              <td className="px-4 py-3 text-gray-900 font-medium truncate">{idp.name}</td>
              <td className="px-4 py-3 text-gray-700">{idp.type === 'saml_2.0' ? 'SAML 2.0' : 'OIDC 1.0'}</td>
              <td className="px-4 py-3 text-gray-700 truncate">{idp.description || '—'}</td>
              <td className="px-4 py-3 text-center">
                {idp.isActive ? (
                  <span className="text-xs font-medium text-green-600">Active</span>
                ) : (
                  <span className="text-xs font-medium text-gray-400">Inactive</span>
                )}
              </td>
            </tr>
          ))}
      </tbody>
    </table>
  )
}

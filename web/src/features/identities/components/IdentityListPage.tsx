import { useIdentities } from '@/features/identities/hooks/useIdentities'

export function IdentityListPage() {
  const { data, isLoading, error } = useIdentities()

  if (isLoading) {
    return <div className="p-6 text-sm text-gray-500">Loading identities...</div>
  }

  if (error) {
    return <div className="p-6 text-sm text-red-600">Failed to load identities.</div>
  }

  return (
    <div className="p-6">
      <h1 className="text-2xl font-semibold text-gray-900 mb-4">Client Workloads</h1>
      {data?.items.length === 0 ? (
        <p className="text-sm text-gray-500">No identities configured yet.</p>
      ) : (
        <ul className="divide-y divide-border">
          {data?.items.map((identity) => (
            <li key={identity.id} className="py-3">
              <p className="font-medium text-gray-900">{identity.name}</p>
              <p className="text-sm text-gray-500">
                {identity.identifiers.length} identifier
                {identity.identifiers.length !== 1 ? 's' : ''}
              </p>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

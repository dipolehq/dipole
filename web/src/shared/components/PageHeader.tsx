import { useLocation } from 'react-router-dom'

export function PageHeader() {
  const { pathname } = useLocation()

  const segments = pathname === '/'
    ? ['Dashboard']
    : pathname.split('/').filter(Boolean).map(formatSegment)

  return (
    <h1 className="text-sm font-mono font-semibold text-gray-800 tracking-tight">
      <span className="text-gray-400">/</span>
      {segments.map((segment, idx) => (
        <span key={idx}>
          <span className="text-gray-900">{segment}</span>
          {idx < segments.length - 1 && <span className="text-gray-400">/</span>}
        </span>
      ))}
    </h1>
  )
}

function formatSegment(segment: string): string {
  // Don't format UUIDs — leave them as-is
  if (/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(segment)) {
    return segment
  }
  return segment
    .split('-')
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ')
}

import { useLocation, Link } from 'react-router-dom'

export function PageHeader() {
  const { pathname } = useLocation()

  const segments = pathname === '/'
    ? [{ label: 'Dashboard', path: '/' }]
    : pathname.split('/').filter(Boolean).map((seg, idx, arr) => ({
        label: formatSegment(seg),
        path: '/' + arr.slice(0, idx + 1).join('/'),
      }))

  return (
    <h1 className="text-[15px] font-semibold tracking-tight leading-[36px]">
      {segments.map((segment, idx) => (
        <span key={idx}>
          {idx < segments.length - 1 ? (
            <Link to={segment.path} className="text-gray-400 hover:text-orange-500 transition-colors">{segment.label}</Link>
          ) : (
            <span className="text-gray-900">{segment.label}</span>
          )}
          {idx < segments.length - 1 && <span className="text-gray-300 mx-1.5">/</span>}
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
    .split(/[-_]/)
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ')
}

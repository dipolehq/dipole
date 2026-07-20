import { Radio } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { LOG_STREAM_EVENT_TYPE_OPTIONS, LOG_STREAM_DESTINATION_OPTIONS } from '@/data/administration/logStreams'
import type { LogStreamListItem } from '@/features/administration/api/logStreamsApi'

interface LogStreamsTableProps {
  items: LogStreamListItem[]
  isLoading?: boolean
}

function getEventTypeLabel(value: string) {
  return LOG_STREAM_EVENT_TYPE_OPTIONS.find((o) => o.value === value)?.label ?? value
}

function getDestinationLabel(value: string) {
  return LOG_STREAM_DESTINATION_OPTIONS.find((o) => o.value === value)?.label ?? value
}

export function LogStreamsTable({ items, isLoading }: LogStreamsTableProps) {
  const navigate = useNavigate()

  return (
    <table className="w-full text-xs table-fixed">
      <colgroup>
        <col className="w-[4%]" />
        <col className="w-[18%]" />
        <col className="w-[28%]" />
        <col className="w-[18%]" />
        <col className="w-[22%]" />
        <col className="w-[10%]" />
      </colgroup>
      <thead className="sticky top-0 bg-gray-50 z-10">
        <tr className="border-b border-border">
          <th className="px-3 py-3" />
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Stream Name</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Description</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Stream Type</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-left">Destination Type</th>
          <th className="px-4 py-3 text-[11px] font-bold text-gray-600 uppercase tracking-wider text-center">Status</th>
        </tr>
      </thead>
      <tbody className="divide-y divide-border">
        {isLoading && (
          <tr>
            <td colSpan={6} className="px-4 py-12 text-center text-gray-400 text-sm" />
          </tr>
        )}
        {!isLoading && items.length === 0 && (
          <tr>
            <td colSpan={6} className="px-4 py-12 text-center text-gray-400 text-sm">
              No log streams found.
            </td>
          </tr>
        )}
        {!isLoading &&
          items.map((ls) => (
            <tr
              key={ls.id}
              className="hover:bg-gray-50 transition-colors cursor-pointer h-[41px]"
              onClick={() => navigate(`/administration/log_streams/${ls.id}`)}
            >
              <td className="px-3 py-3 text-center">
                <Radio size={14} className="text-gray-400" />
              </td>
              <td className="px-4 py-3 text-gray-900 font-medium truncate">{ls.name}</td>
              <td className="px-4 py-3 text-gray-700 truncate">{ls.description || '—'}</td>
              <td className="px-4 py-3 text-gray-700 truncate">{getEventTypeLabel(ls.eventType)}</td>
              <td className="px-4 py-3 text-gray-700 truncate">{getDestinationLabel(ls.destinationType)}</td>
              <td className="px-4 py-3 text-center">
                {ls.isActive ? (
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

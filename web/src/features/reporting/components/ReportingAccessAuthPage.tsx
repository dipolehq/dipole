import { useState } from 'react'
import { useQuery, useQueryClient, keepPreviousData } from '@tanstack/react-query'
import { AccessAuthorizationTable } from '@/features/reporting/components/AccessAuthorizationTable'
import { fetchAuthEvents } from '@/features/reporting/api/eventsApi'

const PAGE_SIZE = 50

export function ReportingAccessAuthPage() {
  const queryClient = useQueryClient()
  const [page, setPage] = useState(1)
  const [search, setSearch] = useState('')
  const [severity, setSeverity] = useState('')

  const { data } = useQuery({
    queryKey: ['events', 'auth', page, PAGE_SIZE, severity, search],
    queryFn: () => fetchAuthEvents({ page, perPage: PAGE_SIZE, severity, search, eventType: '' }),
    placeholderData: keepPreviousData,
  })

  return (
    <div className="flex flex-col h-full">
      <div className="flex-1 min-h-0">
        <AccessAuthorizationTable
          data={data?.items}
          total={data?.total ?? 0}
          page={page}
          pageSize={PAGE_SIZE}
          onPageChange={setPage}
          onSearch={(q) => { setSearch(q); setPage(1) }}
          searchValue={search}
          severity={severity}
          onSeverityChange={(v) => { setSeverity(v); setPage(1) }}
          onRefresh={() => queryClient.invalidateQueries({ queryKey: ['events', 'auth'] })}
        />
      </div>
    </div>
  )
}

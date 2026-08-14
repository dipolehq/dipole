import { useRef, useState } from 'react'
import {
  type ColumnDef,
  flexRender,
  getCoreRowModel,
  useReactTable,
} from '@tanstack/react-table'
import {
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  ChevronsLeft,
  ChevronsRight,
  CircleX,
  Filter,
  ListFilter,
  RefreshCw,
} from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/shared/components/ui/Table'

// --- Props ---
interface LogTableProps<TData> {
  columns: ColumnDef<TData, unknown>[]
  data: TData[]
  /** Total number of rows across all pages (from the backend) */
  total: number
  /** Current page (1-indexed) */
  page: number
  /** Rows per page */
  pageSize: number
  /** Called when the user navigates to a different page */
  onPageChange: (page: number) => void
  /** Called when search input changes */
  onSearch?: (query: string) => void
  /** Current search value (controlled) */
  searchValue?: string
  searchPlaceholder?: string
  /** Keys to search client-side within current page data */
  searchKeys?: string[]
  /** Severity filter value (server-side) */
  severity?: string
  /** Called when severity filter changes */
  onSeverityChange?: (value: string) => void
  renderExpanded: (row: TData) => React.ReactNode
  onRefresh?: () => void
  emptyMessage?: string
}

export function LogTable<TData>({
  columns,
  data,
  total,
  page,
  pageSize,
  onPageChange,
  onSearch,
  searchValue = '',
  searchPlaceholder = 'Filter...',
  searchKeys = [],
  severity = '',
  onSeverityChange,
  renderExpanded,
  onRefresh,
  emptyMessage = 'No events found.',
}: LogTableProps<TData>) {
  const inputRef = useRef<HTMLInputElement>(null)
  const [expandedRowId, setExpandedRowId] = useState<string | null>(null)

  // Client-side search filtering within current page data
  const filteredData = searchValue && searchKeys.length > 0
    ? data.filter((row) => {
        const search = searchValue.toLowerCase()
        return searchKeys.some((key) => {
          const val = (row as Record<string, unknown>)[key]
          return typeof val === 'string' && val.toLowerCase().includes(search)
        })
      })
    : data

  const table = useReactTable({
    data: filteredData,
    columns,
    getCoreRowModel: getCoreRowModel(),
    // No client-side pagination/sorting/filtering — server handles it
    manualPagination: true,
    pageCount: Math.ceil(total / pageSize),
  })

  const totalPages = Math.max(1, Math.ceil(total / pageSize))
  const canPrev = page > 1
  const canNext = page < totalPages
  const colSpan = columns.length + 1 // +1 for chevron column


  return (
    <div className="flex flex-col h-full">
      {/* Toolbar */}
      <div className="flex flex-wrap items-center justify-between gap-3 px-4 py-3 shrink-0">
        <div className="flex items-center gap-3">
          {/* Search input */}
          <div className="relative">
            <input
              ref={inputRef}
              className={cn(
                'h-9 min-w-[240px] rounded-md border border-border bg-white pl-9 pr-9 text-sm placeholder:text-gray-400 focus:outline-none focus:ring-1 focus:ring-gray-300 focus:border-gray-400',
                !searchValue && 'pr-3',
              )}
              value={searchValue}
              onChange={(e) => onSearch?.(e.target.value)}
              placeholder={searchPlaceholder}
              type="text"
            />
            <div className="pointer-events-none absolute inset-y-0 start-0 flex items-center ps-3 text-gray-400">
              <ListFilter size={16} strokeWidth={2} />
            </div>
            {searchValue && (
              <button
                className="absolute inset-y-0 end-0 flex h-full w-9 items-center justify-center text-gray-400 hover:text-gray-600"
                onClick={() => {
                  onSearch?.('')
                  inputRef.current?.focus()
                }}
              >
                <CircleX size={16} strokeWidth={2} />
              </button>
            )}
          </div>

          {/* Filter slot (severity, time range, etc.) */}
          {onSeverityChange && <SeverityFilter severity={severity} onChange={onSeverityChange} />}
        </div>
      </div>

      {/* Table container */}
      <div className="flex flex-col flex-1 min-h-0 mx-4 border border-border bg-white">
        {/* Scrollable table */}
        <div className="flex-1 overflow-y-auto min-h-0">
          <Table className="table-fixed">
            <TableHeader>
              {table.getHeaderGroups().map((headerGroup) => (
                <TableRow key={headerGroup.id} className="hover:bg-transparent">
                  {/* Refresh / chevron column header */}
                  <TableHead className="h-11 w-[40px] min-w-[40px] max-w-[40px]">
                    {onRefresh && (
                      <button
                        type="button"
                        onClick={onRefresh}
                        className="text-gray-400 hover:text-gray-600 transition-colors"
                        title="Refresh"
                      >
                        <RefreshCw size={15} strokeWidth={2.5} />
                      </button>
                    )}
                  </TableHead>
                  {headerGroup.headers.map((header) => (
                    <TableHead
                      key={header.id}
                      style={header.getSize() !== 150 ? { width: `${header.getSize()}px`, minWidth: `${header.getSize()}px` } : undefined}
                      className="h-11"
                    >
                      {header.isPlaceholder
                        ? null
                        : flexRender(header.column.columnDef.header, header.getContext())}
                    </TableHead>
                  ))}
                </TableRow>
              ))}
            </TableHeader>
            <TableBody>
              {table.getRowModel().rows?.length ? (
                table.getRowModel().rows.map((row) => {
                  const rowId = row.id
                  const isExpanded = expandedRowId === rowId
                  return (
                    <LogExpandableRow
                      key={rowId}
                      row={row}
                      isExpanded={isExpanded}
                      onToggle={() => setExpandedRowId(isExpanded ? null : rowId)}
                      renderExpanded={renderExpanded}
                      colSpan={colSpan}
                    />
                  )
                })
              ) : (
                <TableRow>
                  <TableCell colSpan={colSpan} className="h-24 text-center text-gray-400">
                    {emptyMessage}
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </div>

        {/* Pagination footer — server-side */}
        <div className="flex items-center justify-end gap-4 px-4 h-10 border-t border-border bg-gray-50 shrink-0">
          <span className="text-xs text-gray-500 whitespace-nowrap">
            <span className="text-gray-700">
              {total === 0
                ? '0'
                : `${(page - 1) * pageSize + 1}-${Math.min(page * pageSize, total)}`}
            </span>
            {' '}of{' '}
            <span className="text-gray-700">{total}</span>
          </span>
          <div className="flex items-center gap-0.5">
            <button
              onClick={() => onPageChange(1)}
              disabled={!canPrev}
              className="inline-flex items-center justify-center w-7 h-7 rounded border border-border text-gray-400 hover:bg-white disabled:opacity-30 disabled:pointer-events-none transition-colors"
              aria-label="First page"
            >
              <ChevronsLeft size={14} strokeWidth={2} />
            </button>
            <button
              onClick={() => onPageChange(page - 1)}
              disabled={!canPrev}
              className="inline-flex items-center justify-center w-7 h-7 rounded border border-border text-gray-400 hover:bg-white disabled:opacity-30 disabled:pointer-events-none transition-colors"
              aria-label="Previous page"
            >
              <ChevronLeft size={14} strokeWidth={2} />
            </button>
            <button
              onClick={() => onPageChange(page + 1)}
              disabled={!canNext}
              className="inline-flex items-center justify-center w-7 h-7 rounded border border-border text-gray-400 hover:bg-white disabled:opacity-30 disabled:pointer-events-none transition-colors"
              aria-label="Next page"
            >
              <ChevronRight size={14} strokeWidth={2} />
            </button>
            <button
              onClick={() => onPageChange(totalPages)}
              disabled={!canNext}
              className="inline-flex items-center justify-center w-7 h-7 rounded border border-border text-gray-400 hover:bg-white disabled:opacity-30 disabled:pointer-events-none transition-colors"
              aria-label="Last page"
            >
              <ChevronsRight size={14} strokeWidth={2} />
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

// --- Expandable Row ---
function LogExpandableRow<TData>({
  row,
  isExpanded,
  onToggle,
  renderExpanded,
  colSpan,
}: {
  row: ReturnType<ReturnType<typeof useReactTable<TData>>['getRowModel']>['rows'][number]
  isExpanded: boolean
  onToggle: () => void
  renderExpanded: (row: TData) => React.ReactNode
  colSpan: number
}) {
  return (
    <>
      <TableRow className="cursor-pointer" onClick={onToggle}>
        <TableCell className="w-[40px] min-w-[40px] max-w-[40px] text-center">
          <ChevronDown
            size={14}
            className={cn('text-gray-400 transition-transform', isExpanded && 'rotate-180')}
          />
        </TableCell>
        {row.getVisibleCells().map((cell) => (
          <TableCell key={cell.id}>
            {flexRender(cell.column.columnDef.cell, cell.getContext())}
          </TableCell>
        ))}
      </TableRow>
      {isExpanded && (
        <tr>
          <td colSpan={colSpan} className="p-0">
            {renderExpanded(row.original)}
          </td>
        </tr>
      )}
    </>
  )
}

// --- Severity Filter (same design as DataTable's Status filter) ---
const SEVERITY_VALUES = ['Info', 'Warning', 'Error'] as const

function SeverityFilter({ severity, onChange }: { severity: string; onChange: (v: string) => void }) {
  const [open, setOpen] = useState(false)
  const selected = severity ? severity.split(',') : []

  function handleChange(value: string, checked: boolean) {
    const current = [...selected]
    if (checked) {
      current.push(value)
    } else {
      const idx = current.indexOf(value)
      if (idx > -1) current.splice(idx, 1)
    }
    onChange(current.join(','))
  }

  return (
    <div className="relative">
      <button
        onClick={() => setOpen(!open)}
        className="inline-flex items-center gap-2 h-9 px-4 rounded-md border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors"
      >
        <Filter size={16} strokeWidth={2} className="opacity-60" />
        Severity
        {selected.length > 0 && (
          <span className="ml-2 inline-flex h-5 items-center rounded border border-border bg-gray-50 px-1.5 text-[10px] font-medium text-gray-500">
            {selected.length}
          </span>
        )}
      </button>
      {open && (
        <>
          <div className="fixed inset-0 z-40" onClick={() => setOpen(false)} />
          <div className="absolute top-full left-0 mt-1 z-50 min-w-[160px] rounded-lg border border-border bg-white p-3 shadow-lg">
            <div className="space-y-3">
              <div className="text-xs font-medium text-gray-500">Filters</div>
              <div className="space-y-2">
                {SEVERITY_VALUES.map((value) => (
                  <label key={value} className="flex items-center gap-2 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={selected.includes(value)}
                      onChange={(e) => handleChange(value, e.target.checked)}
                      className="h-4 w-4 rounded border-gray-300 text-gray-600 focus:ring-gray-400"
                    />
                    <span className="text-sm text-gray-700">{value}</span>
                  </label>
                ))}
              </div>
            </div>
          </div>
        </>
      )}
    </div>
  )
}

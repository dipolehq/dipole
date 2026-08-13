import { useId, useMemo, useRef, useState } from 'react'
import {
  type ColumnDef,
  type ColumnFiltersState,
  type FilterFn,
  type PaginationState,
  type SortingState,
  type VisibilityState,
  flexRender,
  getCoreRowModel,
  getFacetedUniqueValues,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
} from '@tanstack/react-table'
import {
  ChevronDown,
  ChevronUp,
  ChevronLeft,
  ChevronRight,
  ChevronsLeft,
  ChevronsRight,
  CircleX,
  Columns3,
  Filter,
  ListFilter,
  Plus,
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

// --- Filter function for multi-column text search ---
function createMultiColumnFilter<T>(searchKeys: string[]): FilterFn<T> {
  return (row, _columnId, filterValue) => {
    if (!filterValue) return true
    const search = (filterValue as string).toLowerCase()
    return searchKeys.some((key) => {
      const val = (row.original as Record<string, unknown>)[key]
      return typeof val === 'string' && val.toLowerCase().includes(search)
    })
  }
}

// --- Faceted filter config ---
export interface FacetedFilterConfig {
  columnId: string
  label: string
}

// --- Main DataTable props ---
interface DataTableProps<TData> {
  columns: ColumnDef<TData, unknown>[]
  data: TData[]
  searchPlaceholder?: string
  searchKeys?: string[]
  facetedFilters?: FacetedFilterConfig[]
  actionLabel?: string
  onAction?: () => void
  onRowClick?: (row: TData) => void
  emptyMessage?: string
}

export function DataTable<TData>({
  columns,
  data,
  searchPlaceholder = 'Filter...',
  searchKeys = [],
  facetedFilters = [],
  actionLabel,
  onAction,
  onRowClick,
  emptyMessage = 'No results.',
}: DataTableProps<TData>) {
  const id = useId()
  const inputRef = useRef<HTMLInputElement>(null)

  const [sorting, setSorting] = useState<SortingState>([])
  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([])
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({})
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 50 })
  const [globalFilter, setGlobalFilter] = useState('')

  const globalFilterFn = useMemo(() => {
    if (searchKeys.length === 0) return undefined
    return createMultiColumnFilter<TData>(searchKeys)
  }, [searchKeys])

  const table = useReactTable({
    data,
    columns,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getFacetedUniqueValues: getFacetedUniqueValues(),
    onSortingChange: setSorting,
    onColumnFiltersChange: setColumnFilters,
    onColumnVisibilityChange: setColumnVisibility,
    onPaginationChange: setPagination,
    onGlobalFilterChange: setGlobalFilter,
    globalFilterFn,
    enableSortingRemoval: false,
    state: {
      sorting,
      columnFilters,
      columnVisibility,
      pagination,
      globalFilter,
    },
  })

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
                !globalFilter && 'pr-3',
              )}
              value={globalFilter}
              onChange={(e) => setGlobalFilter(e.target.value)}
              placeholder={searchPlaceholder}
              type="text"
            />
            <div className="pointer-events-none absolute inset-y-0 start-0 flex items-center ps-3 text-gray-400">
              <ListFilter size={16} strokeWidth={2} />
            </div>
            {globalFilter && (
              <button
                className="absolute inset-y-0 end-0 flex h-full w-9 items-center justify-center text-gray-400 hover:text-gray-600"
                onClick={() => {
                  setGlobalFilter('')
                  inputRef.current?.focus()
                }}
              >
                <CircleX size={16} strokeWidth={2} />
              </button>
            )}
          </div>

          {/* Faceted filters */}
          {facetedFilters.map((filter) => (
            <FacetedFilter key={filter.columnId} table={table} config={filter} id={id} />
          ))}

          {/* Column visibility toggle */}
          <ColumnVisibilityToggle table={table} />
        </div>

        {/* Action button */}
        {actionLabel && onAction && (
          <button
            onClick={onAction}
            className="inline-flex items-center gap-2 h-9 px-4 rounded-md border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors"
          >
            <Plus size={16} strokeWidth={2} className="opacity-60" />
            {actionLabel}
          </button>
        )}
      </div>

      {/* Table container with border — holds scrollable rows + pinned pagination footer */}
      <div className="flex flex-col flex-1 min-h-0 mx-4 border border-border bg-white">
        {/* Scrollable table rows */}
        <div className="flex-1 overflow-y-auto min-h-0">
          <Table className="table-fixed">
            <TableHeader>
              {table.getHeaderGroups().map((headerGroup) => (
                <TableRow key={headerGroup.id} className="hover:bg-transparent">
                  {headerGroup.headers.map((header) => (
                    <TableHead
                      key={header.id}
                      style={{ width: header.getSize() !== 150 ? `${header.getSize()}px` : undefined }}
                      className="h-11"
                    >
                      {header.isPlaceholder ? null : header.column.getCanSort() ? (
                        <div
                          className="flex h-full cursor-pointer select-none items-center justify-between gap-2"
                          onClick={header.column.getToggleSortingHandler()}
                          onKeyDown={(e) => {
                            if (e.key === 'Enter' || e.key === ' ') {
                              e.preventDefault()
                              header.column.getToggleSortingHandler()?.(e)
                            }
                          }}
                          tabIndex={0}
                          role="button"
                        >
                          {flexRender(header.column.columnDef.header, header.getContext())}
                          {{
                            asc: <ChevronUp size={16} strokeWidth={2} className="shrink-0 opacity-60" />,
                            desc: <ChevronDown size={16} strokeWidth={2} className="shrink-0 opacity-60" />,
                          }[header.column.getIsSorted() as string] ?? null}
                        </div>
                      ) : (
                        flexRender(header.column.columnDef.header, header.getContext())
                      )}
                    </TableHead>
                  ))}
                </TableRow>
              ))}
            </TableHeader>
            <TableBody>
              {table.getRowModel().rows?.length ? (
                table.getRowModel().rows.map((row) => (
                  <TableRow
                    key={row.id}
                    data-state={row.getIsSelected() && 'selected'}
                    className={onRowClick ? 'cursor-pointer' : ''}
                    onClick={() => onRowClick?.(row.original)}
                  >
                    {row.getVisibleCells().map((cell) => (
                      <TableCell key={cell.id}>
                        {flexRender(cell.column.columnDef.cell, cell.getContext())}
                      </TableCell>
                    ))}
                  </TableRow>
                ))
              ) : (
                <TableRow>
                  <TableCell colSpan={columns.length} className="h-24 text-center text-gray-400">
                    {emptyMessage}
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </div>

        {/* Pagination — pinned at bottom of the table container, always visible */}
        <div className="flex items-center justify-end gap-4 px-4 h-10 border-t border-border bg-gray-50 shrink-0">
          <span className="text-xs text-gray-500 whitespace-nowrap">
            <span className="text-gray-700">
              {table.getRowCount() === 0
                ? '0'
                : `${table.getState().pagination.pageIndex * table.getState().pagination.pageSize + 1}-${Math.min(
                    (table.getState().pagination.pageIndex + 1) * table.getState().pagination.pageSize,
                    table.getRowCount(),
                  )}`}
            </span>
            {' '}of{' '}
            <span className="text-gray-700">{table.getRowCount()}</span>
          </span>
          <div className="flex items-center gap-0.5">
            <button
              onClick={() => table.firstPage()}
              disabled={!table.getCanPreviousPage()}
              className="inline-flex items-center justify-center w-7 h-7 rounded border border-border text-gray-400 hover:bg-white disabled:opacity-30 disabled:pointer-events-none transition-colors"
              aria-label="First page"
            >
              <ChevronsLeft size={14} strokeWidth={2} />
            </button>
            <button
              onClick={() => table.previousPage()}
              disabled={!table.getCanPreviousPage()}
              className="inline-flex items-center justify-center w-7 h-7 rounded border border-border text-gray-400 hover:bg-white disabled:opacity-30 disabled:pointer-events-none transition-colors"
              aria-label="Previous page"
            >
              <ChevronLeft size={14} strokeWidth={2} />
            </button>
            <button
              onClick={() => table.nextPage()}
              disabled={!table.getCanNextPage()}
              className="inline-flex items-center justify-center w-7 h-7 rounded border border-border text-gray-400 hover:bg-white disabled:opacity-30 disabled:pointer-events-none transition-colors"
              aria-label="Next page"
            >
              <ChevronRight size={14} strokeWidth={2} />
            </button>
            <button
              onClick={() => table.lastPage()}
              disabled={!table.getCanNextPage()}
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

// --- Faceted Filter (Status-style popover) ---
function FacetedFilter<TData>({
  table,
  config,
  id,
}: {
  table: ReturnType<typeof useReactTable<TData>>
  config: FacetedFilterConfig
  id: string
}) {
  const [open, setOpen] = useState(false)
  const column = table.getColumn(config.columnId)
  if (!column) return null

  const uniqueValues = Array.from(column.getFacetedUniqueValues().keys()).sort()
  const counts = column.getFacetedUniqueValues()
  const filterValue = (column.getFilterValue() as string[]) ?? []

  function handleChange(value: string, checked: boolean) {
    const current = [...filterValue]
    if (checked) {
      current.push(value)
    } else {
      const idx = current.indexOf(value)
      if (idx > -1) current.splice(idx, 1)
    }
    column?.setFilterValue(current.length ? current : undefined)
  }

  return (
    <div className="relative">
      <button
        onClick={() => setOpen(!open)}
        className="inline-flex items-center gap-2 h-9 px-4 rounded-md border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors"
      >
        <Filter size={16} strokeWidth={2} className="opacity-60" />
        {config.label}
        {filterValue.length > 0 && (
          <span className="ml-2 inline-flex h-5 items-center rounded border border-border bg-gray-50 px-1.5 text-[10px] font-medium text-gray-500">
            {filterValue.length}
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
                {uniqueValues.map((value, i) => (
                  <label key={value} className="flex items-center gap-2 cursor-pointer" htmlFor={`${id}-facet-${config.columnId}-${i}`}>
                    <input
                      id={`${id}-facet-${config.columnId}-${i}`}
                      type="checkbox"
                      checked={filterValue.includes(value)}
                      onChange={(e) => handleChange(value, e.target.checked)}
                      className="h-4 w-4 rounded border-gray-300 text-orange-500 focus:ring-orange-500"
                    />
                    <span className="flex grow justify-between gap-2 text-sm text-gray-700">
                      {value}
                      <span className="text-xs text-gray-400">{counts.get(value)}</span>
                    </span>
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

// --- Column Visibility Toggle ---
function ColumnVisibilityToggle<TData>({ table }: { table: ReturnType<typeof useReactTable<TData>> }) {
  const [open, setOpen] = useState(false)

  return (
    <div className="relative">
      <button
        onClick={() => setOpen(!open)}
        className="inline-flex items-center gap-2 h-9 px-4 rounded-md border border-border bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors"
      >
        <Columns3 size={16} strokeWidth={2} className="opacity-60" />
        View
      </button>
      {open && (
        <>
          <div className="fixed inset-0 z-40" onClick={() => setOpen(false)} />
          <div className="absolute top-full right-0 mt-1 z-50 min-w-[160px] rounded-lg border border-border bg-white py-1 shadow-lg">
            <div className="px-3 py-2 text-xs font-medium text-gray-500">Toggle columns</div>
            {table
              .getAllColumns()
              .filter((column) => column.getCanHide())
              .map((column) => (
                <label
                  key={column.id}
                  className="flex items-center gap-2 px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-50 cursor-pointer capitalize"
                >
                  <input
                    type="checkbox"
                    checked={column.getIsVisible()}
                    onChange={(e) => column.toggleVisibility(e.target.checked)}
                    className="h-4 w-4 rounded border-gray-300 text-orange-500 focus:ring-orange-500"
                  />
                  {column.id}
                </label>
              ))}
          </div>
        </>
      )}
    </div>
  )
}

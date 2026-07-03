import { useState, useRef } from 'react'
import { ChevronDown, ChevronLeft, ChevronRight } from 'lucide-react'
import { useClickOutside } from '@/shared/hooks/useClickOutside'

type TimespanMode = 'relative' | 'absolute'

const RELATIVE_PRESETS = [
  { unit: 'Minutes', values: [10, 15, 30, 45] },
  { unit: 'Hours', values: [3, 6, 12, 24] },
  { unit: 'Days', values: [2, 3, 4, 5] },
  { unit: 'Weeks', values: [1, 2, 3, 4] },
  { unit: 'Months', values: [3, 6, 9, 12] },
] as const

const UNIT_OPTIONS = ['Minutes', 'Hours', 'Days', 'Weeks', 'Months'] as const

interface TimespanPickerProps {
  value: string
  onChange: (value: string) => void
}

export function TimespanPicker({ value, onChange }: TimespanPickerProps) {
  const [open, setOpen] = useState(false)
  const [mode, setMode] = useState<TimespanMode>('relative')
  const [customDuration, setCustomDuration] = useState('1')
  const [customUnit, setCustomUnit] = useState<string>('Days')
  const [unitOpen, setUnitOpen] = useState(false)

  // Absolute state
  const [startDate, setStartDate] = useState('')
  const [startTime, setStartTime] = useState('00:00')
  const [endDate, setEndDate] = useState('')
  const [endTime, setEndTime] = useState('00:00')
  const [leftMonth, setLeftMonth] = useState(() => {
    const now = new Date()
    return new Date(now.getFullYear(), now.getMonth() - 1, 1)
  })
  const [rightMonth, setRightMonth] = useState(() => new Date())

  const ref = useRef<HTMLDivElement>(null)
  useClickOutside(ref, () => { setOpen(false); setUnitOpen(false) })

  const displayLabel = value || 'Timespan'

  function handlePresetClick(val: number, unit: string) {
    const label = `${val} ${unit}`
    onChange(label)
    setOpen(false)
  }

  function handleCustomApply() {
    const label = `${customDuration} ${customUnit}`
    onChange(label)
    setOpen(false)
  }

  function handleAbsoluteApply() {
    if (startDate && endDate) {
      onChange(`${startDate} ${startTime} — ${endDate} ${endTime}`)
    }
    setOpen(false)
  }

  function navigateMonth(side: 'left' | 'right', direction: -1 | 1) {
    if (side === 'left') {
      setLeftMonth(new Date(leftMonth.getFullYear(), leftMonth.getMonth() + direction, 1))
    } else {
      setRightMonth(new Date(rightMonth.getFullYear(), rightMonth.getMonth() + direction, 1))
    }
  }

  return (
    <div ref={ref} className="relative">
      <button
        type="button"
        onClick={() => setOpen(!open)}
        className={`w-full flex items-center justify-between px-3 py-2 border rounded-md text-sm transition-colors ${
          open ? 'border-orange-400 ring-2 ring-orange-100' : 'border-border'
        } ${value ? 'text-gray-900' : 'text-gray-400'}`}
      >
        <span className="truncate">{displayLabel}</span>
        <ChevronDown size={14} className={`shrink-0 text-orange-500 transition-transform ${open ? 'rotate-180' : ''}`} />
      </button>

      {open && (
        <div className="absolute top-full left-0 mt-1 bg-white border border-border rounded-lg shadow-lg z-50 p-4" style={{ width: mode === 'absolute' ? '580px' : '280px' }}>
          {/* Mode toggle */}
          <div className="flex items-center gap-2 mb-4">
            <button
              type="button"
              onClick={() => setMode('relative')}
              className={`px-3 py-1 text-xs font-medium rounded-md transition-colors ${
                mode === 'relative' ? 'bg-orange-500 text-white' : 'text-orange-500 hover:bg-orange-50'
              }`}
            >
              Relative
            </button>
            <button
              type="button"
              onClick={() => setMode('absolute')}
              className={`px-3 py-1 text-xs font-medium rounded-md transition-colors ${
                mode === 'absolute' ? 'bg-orange-500 text-white' : 'text-orange-500 hover:bg-orange-50'
              }`}
            >
              Absolute
            </button>
          </div>

          {mode === 'relative' ? (
            <div>
              {/* Presets grid */}
              <div className="space-y-2 mb-4">
                {RELATIVE_PRESETS.map((row) => (
                  <div key={row.unit} className="flex items-center">
                    <span className="text-xs font-medium text-gray-700 w-16">{row.unit}</span>
                    <div className="flex gap-2">
                      {row.values.map((val) => (
                        <button
                          key={val}
                          type="button"
                          onClick={() => handlePresetClick(val, row.unit)}
                          className="text-xs text-orange-500 hover:text-orange-600 font-medium w-8 text-center"
                        >
                          {val}
                        </button>
                      ))}
                    </div>
                  </div>
                ))}
              </div>

              {/* Custom duration */}
              <div className="flex items-end gap-2 pt-3 border-t border-border">
                <div>
                  <label className="block text-[10px] text-gray-500 mb-1">Duration</label>
                  <input
                    type="number"
                    min="1"
                    value={customDuration}
                    onChange={(e) => setCustomDuration(e.target.value)}
                    className="w-14 px-2 py-1.5 border border-border rounded-md text-xs focus:outline-none focus:ring-1 focus:ring-orange-300 focus:border-orange-400"
                  />
                </div>
                <div className="relative flex-1">
                  <label className="block text-[10px] text-gray-500 mb-1">Unit of Time</label>
                  <button
                    type="button"
                    onClick={() => setUnitOpen(!unitOpen)}
                    className="w-full flex items-center justify-between px-2 py-1.5 border border-border rounded-md text-xs"
                  >
                    <span>{customUnit}</span>
                    <ChevronDown size={12} className={`text-gray-400 transition-transform ${unitOpen ? 'rotate-180' : ''}`} />
                  </button>
                  {unitOpen && (
                    <div className="absolute top-full left-0 right-0 mt-1 bg-white border border-border rounded-md shadow-lg z-50">
                      {UNIT_OPTIONS.map((u) => (
                        <button
                          key={u}
                          type="button"
                          onClick={() => { setCustomUnit(u); setUnitOpen(false) }}
                          className={`w-full text-left px-3 py-2 text-xs transition-colors ${
                            customUnit === u ? 'bg-gray-100 text-orange-600 font-medium' : 'text-gray-600 hover:bg-gray-50'
                          }`}
                        >
                          {u}
                        </button>
                      ))}
                    </div>
                  )}
                </div>
                <button
                  type="button"
                  onClick={handleCustomApply}
                  className="px-3 py-1.5 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors"
                >
                  Apply
                </button>
              </div>
            </div>
          ) : (
            <div>
              {/* Dual calendar */}
              <div className="flex gap-4 mb-4">
                <CalendarMonth
                  month={leftMonth}
                  onPrev={() => navigateMonth('left', -1)}
                  onNext={() => navigateMonth('left', 1)}
                  onSelectDate={setStartDate}
                  selectedDate={startDate}
                />
                <CalendarMonth
                  month={rightMonth}
                  onPrev={() => navigateMonth('right', -1)}
                  onNext={() => navigateMonth('right', 1)}
                  onSelectDate={setEndDate}
                  selectedDate={endDate}
                />
              </div>

              {/* Date/Time inputs */}
              <div className="grid grid-cols-4 gap-2 pt-3 border-t border-border">
                <div>
                  <label className="block text-[10px] text-gray-500 mb-1">Start Date</label>
                  <input
                    type="text"
                    placeholder="DD/MM/YYYY"
                    value={startDate}
                    onChange={(e) => setStartDate(e.target.value)}
                    className="w-full px-2 py-1.5 border border-border rounded-md text-[11px] focus:outline-none focus:ring-1 focus:ring-orange-300"
                  />
                </div>
                <div>
                  <label className="block text-[10px] text-gray-500 mb-1">Start Time</label>
                  <input
                    type="text"
                    placeholder="00:00"
                    value={startTime}
                    onChange={(e) => setStartTime(e.target.value)}
                    className="w-full px-2 py-1.5 border border-border rounded-md text-[11px] focus:outline-none focus:ring-1 focus:ring-orange-300"
                  />
                </div>
                <div>
                  <label className="block text-[10px] text-gray-500 mb-1">End Date</label>
                  <input
                    type="text"
                    placeholder="DD/MM/YYYY"
                    value={endDate}
                    onChange={(e) => setEndDate(e.target.value)}
                    className="w-full px-2 py-1.5 border border-border rounded-md text-[11px] focus:outline-none focus:ring-1 focus:ring-orange-300"
                  />
                </div>
                <div>
                  <label className="block text-[10px] text-gray-500 mb-1">End Time</label>
                  <input
                    type="text"
                    placeholder="00:00"
                    value={endTime}
                    onChange={(e) => setEndTime(e.target.value)}
                    className="w-full px-2 py-1.5 border border-border rounded-md text-[11px] focus:outline-none focus:ring-1 focus:ring-orange-300"
                  />
                </div>
              </div>
              <div className="flex justify-end mt-3">
                <button
                  type="button"
                  onClick={handleAbsoluteApply}
                  className="px-3 py-1.5 bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium rounded-md transition-colors"
                >
                  Apply
                </button>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  )
}

// --- Calendar Month sub-component ---

function CalendarMonth({
  month,
  onPrev,
  onNext,
  onSelectDate,
  selectedDate,
}: {
  month: Date
  onPrev: () => void
  onNext: () => void
  onSelectDate: (date: string) => void
  selectedDate: string
}) {
  const year = month.getFullYear()
  const mo = month.getMonth()
  const monthName = month.toLocaleString('en', { month: 'short' }).toUpperCase()

  const firstDay = new Date(year, mo, 1).getDay()
  const daysInMonth = new Date(year, mo + 1, 0).getDate()

  const days: (number | null)[] = []
  for (let i = 0; i < firstDay; i++) days.push(null)
  for (let i = 1; i <= daysInMonth; i++) days.push(i)

  function formatDay(day: number): string {
    const d = String(day).padStart(2, '0')
    const m = String(mo + 1).padStart(2, '0')
    return `${d}/${m}/${year}`
  }

  return (
    <div className="flex-1">
      <div className="flex items-center justify-between mb-2">
        <span className="text-xs font-semibold text-gray-700">{monthName} {year}</span>
        <div className="flex items-center gap-1">
          <button type="button" onClick={onPrev} className="text-orange-500 hover:text-orange-600">
            <ChevronLeft size={14} />
          </button>
          <button type="button" onClick={onNext} className="text-orange-500 hover:text-orange-600">
            <ChevronRight size={14} />
          </button>
        </div>
      </div>
      <div className="grid grid-cols-7 gap-0 text-center">
        {['S', 'M', 'T', 'W', 'T', 'F', 'S'].map((d, i) => (
          <span key={i} className="text-[10px] font-medium text-gray-500 py-1">{d}</span>
        ))}
        {days.map((day, i) => (
          <button
            key={i}
            type="button"
            disabled={!day}
            onClick={() => day && onSelectDate(formatDay(day))}
            className={`text-[11px] py-1 rounded transition-colors ${
              !day
                ? ''
                : selectedDate === formatDay(day)
                  ? 'bg-orange-500 text-white font-medium'
                  : 'text-orange-500 hover:bg-orange-50'
            }`}
          >
            {day ?? ''}
          </button>
        ))}
      </div>
    </div>
  )
}

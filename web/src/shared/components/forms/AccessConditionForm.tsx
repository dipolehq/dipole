import { type ReactNode } from 'react'
import { TagsField } from '@/shared/components/TagsField'
import {
  FloatingInput,
  FloatingTextarea,
  FloatingDropdown,
} from '@/shared/components/FloatingInput'
import { COUNTRIES, SUBDIVISIONS, TIMEZONES, CONDITION_INTEGRATIONS } from '@/shared/constants'
import type { AccessConditionFormValues } from '@/shared/components/forms/types'

const INTEGRATION_TYPES = CONDITION_INTEGRATIONS

interface AccessConditionFormProps {
  values: AccessConditionFormValues
  onChange: (values: AccessConditionFormValues) => void
  disabled?: boolean
  statusSlot?: ReactNode
}

export function AccessConditionForm({
  values,
  onChange,
  disabled = false,
  statusSlot,
}: AccessConditionFormProps) {
  function update(partial: Partial<AccessConditionFormValues>) {
    onChange({ ...values, ...partial })
  }

  const disabledClass = disabled ? 'opacity-60 pointer-events-none' : ''

  return (
    <fieldset disabled={disabled} className={disabledClass}>
      <div className="space-y-5">
        <div className="border border-border rounded-lg p-5">
          <h4 className="text-sm font-semibold text-gray-800 mb-4">Details</h4>
          <div className="space-y-4">
            {statusSlot}
            <FloatingInput
              label="Name"
              value={values.name}
              onChange={(v) => update({ name: v })}
              disabled={disabled}
              required
            />
            <FloatingTextarea
              label="Description (Optional)"
              value={values.description}
              onChange={(v) => update({ description: v })}
              disabled={disabled}
              rows={2}
            />
          </div>
        </div>

        <div className="border border-border rounded-lg p-5">
          <h4 className="text-sm font-semibold text-gray-800 mb-4">Condition Configuration</h4>
          <div className="space-y-4">
            <FloatingDropdown
              label="Integration Type"
              options={INTEGRATION_TYPES}
              value={values.integration}
              onChange={(v) => update({ integration: v })}
              disabled={disabled}
            />

            {values.integration && (
              <div className="border border-border rounded-md p-4 mt-4">
                <h5 className="text-xs font-semibold text-gray-800 mb-3">Conditions</h5>
                {values.integration === 'geoip' && (
                  <GeoIPConditionForm
                    countries={values.countries}
                    onCountriesChange={(c) => update({ countries: c })}
                  />
                )}
                {values.integration === 'time' && (
                  <TimeConditionForm
                    timezone={values.timezone}
                    onTimezoneChange={(v) => update({ timezone: v })}
                    days={values.days}
                    onDaysChange={(d) => update({ days: d })}
                  />
                )}
              </div>
            )}
          </div>
        </div>

        <div className="border border-border rounded-lg p-5">
          <h4 className="text-sm font-semibold text-gray-800 mb-4">Tags</h4>
          <TagsField
            value={values.tags}
            onChange={(tags) => update({ tags })}
            disabled={disabled}
          />
        </div>
      </div>
    </fieldset>
  )
}

function GeoIPConditionForm({
  countries,
  onCountriesChange,
}: {
  countries: { country: string; subdivisions: string[] }[]
  onCountriesChange: (c: { country: string; subdivisions: string[] }[]) => void
}) {
  function updateCountry(idx: number, country: string) {
    onCountriesChange(countries.map((c, i) => (i === idx ? { country, subdivisions: [''] } : c)))
  }
  function updateSubdivision(idx: number, subIdx: number, value: string) {
    onCountriesChange(
      countries.map((c, i) => {
        if (i !== idx) return c
        const subs = [...c.subdivisions]
        subs[subIdx] = value
        return { ...c, subdivisions: subs }
      }),
    )
  }
  function addSubdivision(idx: number) {
    onCountriesChange(
      countries.map((c, i) => (i === idx ? { ...c, subdivisions: [...c.subdivisions, ''] } : c)),
    )
  }
  function removeCountry(idx: number) {
    if (countries.length > 1) onCountriesChange(countries.filter((_, i) => i !== idx))
  }
  function removeSubdivision(idx: number, subIdx: number) {
    onCountriesChange(
      countries.map((c, i) => {
        if (i !== idx) return c
        const subs = c.subdivisions.filter((_, si) => si !== subIdx)
        return { ...c, subdivisions: subs.length === 0 ? [''] : subs }
      }),
    )
  }
  function addCountry() {
    onCountriesChange([...countries, { country: '', subdivisions: [''] }])
  }

  return (
    <div>
      <p className="text-[11px] text-gray-500 mb-4">
        Add or remove locations that will be allowed by this Access Condition.
      </p>
      {countries.map((entry, idx) => (
        <div key={idx} className="mb-4">
          <div className="grid grid-cols-2 gap-3">
            <div>
              <div className="flex items-center gap-1">
                <div className="flex-1">
                  <FloatingDropdown
                    label="Country"
                    options={COUNTRIES}
                    value={entry.country}
                    onChange={(v) => updateCountry(idx, v)}
                  />
                </div>
                <button
                  onClick={() => removeCountry(idx)}
                  className="text-orange-400 hover:text-red-500 text-sm shrink-0"
                >
                  ×
                </button>
              </div>
            </div>
            <div className="space-y-2">
              {entry.subdivisions.map((sub, subIdx) => (
                <div key={subIdx} className="flex items-center gap-1">
                  <div className="flex-1">
                    <FloatingDropdown
                      label="Subdivision"
                      options={SUBDIVISIONS[entry.country] || []}
                      value={sub}
                      onChange={(v) => updateSubdivision(idx, subIdx, v)}
                    />
                  </div>
                  <button
                    onClick={() => removeSubdivision(idx, subIdx)}
                    className="text-orange-400 hover:text-red-500 text-sm shrink-0"
                  >
                    ×
                  </button>
                </div>
              ))}
              <button
                onClick={() => addSubdivision(idx)}
                className="text-orange-500 hover:text-orange-600 text-lg font-bold"
              >
                +
              </button>
            </div>
          </div>
        </div>
      ))}
      <div className="text-center">
        <button
          onClick={addCountry}
          className="text-xs text-orange-500 hover:text-orange-600 font-medium"
        >
          + Add Country
        </button>
      </div>
    </div>
  )
}

function TimeConditionForm({
  timezone,
  onTimezoneChange,
  days,
  onDaysChange,
}: {
  timezone: string
  onTimezoneChange: (v: string) => void
  days: { day: string; startTime: string; endTime: string; enabled: boolean }[]
  onDaysChange: (d: { day: string; startTime: string; endTime: string; enabled: boolean }[]) => void
}) {
  return (
    <div className="space-y-4">
      <FloatingDropdown
        label="Timezone"
        options={TIMEZONES}
        value={timezone}
        onChange={onTimezoneChange}
      />
      <div className="space-y-2">
        {days.map((day, idx) => (
          <div key={day.day} className="flex items-center gap-3">
            <span className="text-xs text-gray-700 w-24">{day.day}</span>
            {day.enabled ? (
              <div className="flex items-center gap-2 flex-1">
                <input
                  type="time"
                  value={day.startTime}
                  onChange={(e) =>
                    onDaysChange(
                      days.map((d, i) => (i === idx ? { ...d, startTime: e.target.value } : d)),
                    )
                  }
                  className="px-2 py-1.5 border border-border rounded-md text-xs focus:outline-none focus:ring-1 focus:ring-orange-500"
                />
                <span className="text-xs text-gray-400">to</span>
                <input
                  type="time"
                  value={day.endTime}
                  onChange={(e) =>
                    onDaysChange(
                      days.map((d, i) => (i === idx ? { ...d, endTime: e.target.value } : d)),
                    )
                  }
                  className="px-2 py-1.5 border border-border rounded-md text-xs focus:outline-none focus:ring-1 focus:ring-orange-500"
                />
                <button
                  onClick={() =>
                    onDaysChange(days.map((d, i) => (i === idx ? { ...d, enabled: false } : d)))
                  }
                  className="text-red-400 hover:text-red-500 text-sm"
                >
                  ×
                </button>
              </div>
            ) : (
              <button
                onClick={() =>
                  onDaysChange(days.map((d, i) => (i === idx ? { ...d, enabled: true } : d)))
                }
                className="text-orange-500 hover:text-orange-600 text-lg font-bold"
              >
                +
              </button>
            )}
          </div>
        ))}
      </div>
      <p className="text-[10px] text-gray-400">At least one time condition is required.</p>
    </div>
  )
}

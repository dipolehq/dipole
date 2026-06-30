import { useState } from 'react'
import { Dropdown } from '@/shared/components/Dropdown'
import { TagsField } from '@/shared/components/TagsField'

const INTEGRATION_TYPES = [
  { value: 'geoip', label: 'Dipole GeoIP Condition' },
  { value: 'time', label: 'Dipole Time Condition' },
]

export function AccessConditionForm() {
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [integration, setIntegration] = useState('')

  return (
    <div className="px-4 py-4 pb-8 space-y-4">
      <div>
        <label className="block text-[11px] text-gray-500 mb-1">Display Name</label>
        <input
          type="text"
          placeholder="Display Name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          className="w-full px-3 py-2 border border-border rounded-md text-sm focus:outline-none focus:ring-1 focus:ring-gray-300"
        />
      </div>

      <div>
        <label className="block text-[11px] text-gray-500 mb-1">Description (Optional)</label>
        <textarea
          placeholder="Description (Optional)"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          rows={2}
          className="w-full px-3 py-2 border border-border rounded-md text-sm resize-y focus:outline-none focus:ring-1 focus:ring-gray-300"
        />
      </div>

      <div>
        <h4 className="text-xs font-semibold text-gray-800 mb-2">Select Integration</h4>
        <Dropdown
          options={INTEGRATION_TYPES}
          value={integration}
          onChange={setIntegration}
          placeholder="Select an integration..."
        />
      </div>

      {integration && (
        <div className="border border-border rounded-md p-4">
          <h4 className="text-xs font-semibold text-gray-800 mb-1">Conditions</h4>
          {integration === 'geoip' && <GeoIPConditionForm />}
          {integration === 'time' && <TimeConditionForm />}
        </div>
      )}

      <div>
        <label className="block text-xs font-medium text-gray-700 mb-1.5">Tags</label>
        <TagsField />
      </div>
    </div>
  )
}

function GeoIPConditionForm() {
  const [countries, setCountries] = useState<{ country: string; subdivisions: string[] }[]>([
    { country: '', subdivisions: [''] },
  ])

  const COUNTRIES = [
    { value: 'IN', label: 'India' },
    { value: 'US', label: 'United States of America (the)' },
    { value: 'GB', label: 'United Kingdom' },
    { value: 'DE', label: 'Germany' },
    { value: 'FR', label: 'France' },
    { value: 'JP', label: 'Japan' },
    { value: 'AU', label: 'Australia' },
    { value: 'CA', label: 'Canada' },
    { value: 'SG', label: 'Singapore' },
    { value: 'AE', label: 'United Arab Emirates' },
  ]

  const SUBDIVISIONS: Record<string, { value: string; label: string }[]> = {
    IN: [
      { value: 'KA', label: 'Karnataka' },
      { value: 'MH', label: 'Maharashtra' },
      { value: 'DL', label: 'Delhi' },
      { value: 'TN', label: 'Tamil Nadu' },
      { value: 'TG', label: 'Telangana' },
      { value: 'UP', label: 'Uttar Pradesh' },
      { value: 'GJ', label: 'Gujarat' },
    ],
    US: [
      { value: 'WA', label: 'Washington' },
      { value: 'CA', label: 'California' },
      { value: 'NY', label: 'New York' },
      { value: 'TX', label: 'Texas' },
      { value: 'VA', label: 'Virginia' },
      { value: 'OR', label: 'Oregon' },
    ],
    GB: [
      { value: 'ENG', label: 'England' },
      { value: 'SCT', label: 'Scotland' },
      { value: 'WLS', label: 'Wales' },
    ],
  }

  function updateCountry(idx: number, country: string) {
    setCountries(countries.map((c, i) => i === idx ? { country, subdivisions: [''] } : c))
  }

  function updateSubdivision(idx: number, subIdx: number, value: string) {
    setCountries(countries.map((c, i) => {
      if (i !== idx) return c
      const subs = [...c.subdivisions]
      subs[subIdx] = value
      return { ...c, subdivisions: subs }
    }))
  }

  function addSubdivision(idx: number) {
    setCountries(countries.map((c, i) => i === idx ? { ...c, subdivisions: [...c.subdivisions, ''] } : c))
  }

  function removeCountry(idx: number) {
    if (countries.length > 1) {
      setCountries(countries.filter((_, i) => i !== idx))
    }
  }

  function removeSubdivision(idx: number, subIdx: number) {
    setCountries(countries.map((c, i) => {
      if (i !== idx) return c
      const subs = c.subdivisions.filter((_, si) => si !== subIdx)
      return { ...c, subdivisions: subs.length === 0 ? [''] : subs }
    }))
  }

  function addCountry() {
    setCountries([...countries, { country: '', subdivisions: [''] }])
  }

  return (
    <div>
      <h5 className="text-xs font-medium text-gray-700 mb-1">Location</h5>
      <p className="text-[11px] text-gray-500 mb-4">
        Add or remove locations that will be allowed by this Access Condition. Requests from locations not identified here will be prohibited. If no Subdivisions are selected for a Country, then all Subdivisions will be allowed for that Country.
      </p>

      {countries.map((entry, idx) => (
        <div key={idx} className="mb-4">
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-[10px] font-semibold text-gray-600 uppercase mb-1">Country</label>
              <div className="flex items-center gap-1">
                <div className="flex-1">
                  <Dropdown
                    options={COUNTRIES}
                    value={entry.country}
                    onChange={(v) => updateCountry(idx, v)}
                    placeholder="Select country..."
                  />
                </div>
                <button onClick={() => removeCountry(idx)} className="text-orange-400 hover:text-red-500 text-sm shrink-0">×</button>
              </div>
            </div>
            <div>
              <label className="block text-[10px] font-semibold text-gray-600 uppercase mb-1">Subdivision</label>
              {entry.subdivisions.map((sub, subIdx) => (
                <div key={subIdx} className="flex items-center gap-1 mb-1.5">
                  <div className="flex-1">
                    <Dropdown
                      options={SUBDIVISIONS[entry.country] || []}
                      value={sub}
                      onChange={(v) => updateSubdivision(idx, subIdx, v)}
                      placeholder="Select subdivision..."
                    />
                  </div>
                  <button onClick={() => removeSubdivision(idx, subIdx)} className="text-orange-400 hover:text-red-500 text-sm shrink-0">×</button>
                </div>
              ))}
              <button onClick={() => addSubdivision(idx)} className="text-orange-500 hover:text-orange-600 text-lg font-bold">+</button>
            </div>
          </div>
        </div>
      ))}

      <div className="text-center">
        <button onClick={addCountry} className="text-xs text-orange-500 hover:text-orange-600 font-medium">
          + Add Country
        </button>
      </div>
    </div>
  )
}

function TimeConditionForm() {
  const [timezone, setTimezone] = useState('')
  const [days, setDays] = useState<{ day: string; startTime: string; endTime: string; enabled: boolean }[]>([
    { day: 'Monday', startTime: '09:00', endTime: '18:00', enabled: false },
    { day: 'Tuesday', startTime: '09:00', endTime: '18:00', enabled: false },
    { day: 'Wednesday', startTime: '09:00', endTime: '18:00', enabled: false },
    { day: 'Thursday', startTime: '09:00', endTime: '18:00', enabled: false },
    { day: 'Friday', startTime: '09:00', endTime: '18:00', enabled: false },
    { day: 'Saturday', startTime: '09:00', endTime: '18:00', enabled: false },
    { day: 'Sunday', startTime: '09:00', endTime: '18:00', enabled: false },
  ])

  const TIMEZONES = [
    { value: 'Asia/Kolkata', label: 'Asia/Kolkata (IST)' },
    { value: 'America/New_York', label: 'America/New_York (EST)' },
    { value: 'America/Chicago', label: 'America/Chicago (CST)' },
    { value: 'America/Los_Angeles', label: 'America/Los_Angeles (PST)' },
    { value: 'Europe/London', label: 'Europe/London (GMT)' },
    { value: 'Europe/Berlin', label: 'Europe/Berlin (CET)' },
    { value: 'Asia/Tokyo', label: 'Asia/Tokyo (JST)' },
    { value: 'Asia/Singapore', label: 'Asia/Singapore (SGT)' },
    { value: 'Australia/Sydney', label: 'Australia/Sydney (AEST)' },
    { value: 'UTC', label: 'UTC' },
  ]

  function toggleDay(idx: number) {
    setDays(days.map((d, i) => i === idx ? { ...d, enabled: !d.enabled } : d))
  }

  function updateTime(idx: number, field: 'startTime' | 'endTime', val: string) {
    setDays(days.map((d, i) => i === idx ? { ...d, [field]: val } : d))
  }

  return (
    <div className="space-y-4">
      <div>
        <label className="block text-[11px] text-gray-500 mb-1">Timezone</label>
        <Dropdown
          options={TIMEZONES}
          value={timezone}
          onChange={setTimezone}
          placeholder="Select timezone..."
        />
      </div>

      <div className="space-y-2">
        {days.map((day, idx) => (
          <div key={day.day} className="flex items-center gap-3">
            <span className="text-xs text-gray-700 w-24">{day.day}</span>
            {day.enabled ? (
              <div className="flex items-center gap-2 flex-1">
                <input
                  type="time"
                  value={day.startTime}
                  onChange={(e) => updateTime(idx, 'startTime', e.target.value)}
                  className="px-2 py-1.5 border border-border rounded-md text-xs focus:outline-none focus:ring-1 focus:ring-gray-300"
                />
                <span className="text-xs text-gray-400">to</span>
                <input
                  type="time"
                  value={day.endTime}
                  onChange={(e) => updateTime(idx, 'endTime', e.target.value)}
                  className="px-2 py-1.5 border border-border rounded-md text-xs focus:outline-none focus:ring-1 focus:ring-gray-300"
                />
                <button onClick={() => toggleDay(idx)} className="text-red-400 hover:text-red-500 text-sm">×</button>
              </div>
            ) : (
              <button onClick={() => toggleDay(idx)} className="text-orange-500 hover:text-orange-600 text-lg font-bold">+</button>
            )}
          </div>
        ))}
      </div>

      <p className="text-[10px] text-gray-400">At least one time condition is required.</p>
    </div>
  )
}

import { PageHeader } from '@/shared/components/PageHeader'

export function SupportPage() {
  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-4 shrink-0 h-[44px] border-b border-border bg-white">
        <PageHeader />
      </div>
      <div className="flex-1 min-h-0 mx-4 mt-3 border border-border bg-white overflow-hidden rounded-lg">
        <iframe
          src="https://calendar.google.com/calendar/appointments/schedules/AcZssZ0N0iLlhp8Fe3A0fL4nysUU5ThEz6C7pK435UoWKt5b0tX8uUGA0StrHC6oHIsl29V3q9KLpHSK?gv=true"
          className="w-full h-full border-0"
          title="Schedule support with Dipole"
        />
      </div>
    </div>
  )
}

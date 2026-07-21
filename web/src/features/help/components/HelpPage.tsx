import { useState } from 'react'
import { FileText } from 'lucide-react'
import { PageHeader } from '@/shared/components/PageHeader'
import { GettingStartedPanel } from '@/features/help/components/GettingStartedPanel'
import { TroubleshootPanel } from '@/features/help/components/TroubleshootPanel'

const HELP_SECTIONS = [
  { value: 'getting_started', label: 'Getting Started' },
  { value: 'troubleshoot', label: 'Troubleshoot' },
] as const

type HelpSection = (typeof HELP_SECTIONS)[number]['value']

export function HelpPage() {
  const [selectedSection, setSelectedSection] = useState<HelpSection>('getting_started')
  const [sectionOpen, setSectionOpen] = useState(false)

  const selectedLabel = HELP_SECTIONS.find((s) => s.value === selectedSection)?.label ?? ''

  return (
    <div className="flex flex-col h-full">
      {/* Header bar */}
      <div className="flex items-center justify-between px-4 shrink-0 h-[60px] border-b border-border bg-white">
        <PageHeader />
        <div className="flex items-center gap-3">
          <div className="relative">
            <button
              type="button"
              onClick={() => setSectionOpen(!sectionOpen)}
              className="flex items-center gap-1.5 w-[160px] px-4 py-2 bg-[#28282B] hover:bg-[#333336] text-white text-sm font-medium rounded-md transition-colors"
            >
              <FileText size={14} className="shrink-0" />
              <span className="truncate">{selectedLabel}</span>
              <svg className={`w-3.5 h-3.5 ml-auto shrink-0 transition-transform ${sectionOpen ? 'rotate-180' : ''}`} fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
              </svg>
            </button>
            {sectionOpen && (
              <div className="absolute top-full right-0 mt-1 bg-white border border-border rounded-md shadow-lg z-50 w-[200px] overflow-hidden">
                {HELP_SECTIONS.map((opt) => (
                  <button key={opt.value} type="button" onClick={() => { setSelectedSection(opt.value); setSectionOpen(false) }} className={`w-full text-left px-4 py-2.5 text-sm transition-colors ${selectedSection === opt.value ? 'bg-orange-50 text-orange-600 font-medium' : 'text-gray-600 hover:bg-gray-50'}`}>{opt.label}</button>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto min-h-0 px-4 py-3">
        {selectedSection === 'getting_started' ? (
          <GettingStartedPanel />
        ) : selectedSection === 'troubleshoot' ? (
          <TroubleshootPanel />
        ) : null}
      </div>
    </div>
  )
}

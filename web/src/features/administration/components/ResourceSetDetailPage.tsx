import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { FileText } from 'lucide-react'
import { PageHeader } from '@/shared/components/PageHeader'
import { FloatingInput, FloatingTextarea } from '@/shared/components/FloatingInput'
import { useResourceSet, useSaveResourceSet, useDeleteResourceSet } from '@/features/administration/hooks/useResourceSets'
import { apiErrorMessage } from '@/features/administration/api/resourceSetsApi'

const DETAIL_SECTIONS = [
  { value: 'details', label: 'Details' },
  { value: 'metrics', label: 'Metrics' },
  { value: 'logs', label: 'Logs' },
] as const

type DetailSection = (typeof DETAIL_SECTIONS)[number]['value']

export function ResourceSetDetailPage() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()

  const { data: resourceSet, isLoading } = useResourceSet(id)
  const saveMutation = useSaveResourceSet(id ?? '')
  const deleteMutation = useDeleteResourceSet()

  const [selectedSection, setSelectedSection] = useState<DetailSection>('details')
  const [sectionOpen, setSectionOpen] = useState(false)
  const [isEditing, setIsEditing] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)

  const [displayName, setDisplayName] = useState('')
  const [description, setDescription] = useState('')
  const [initialized, setInitialized] = useState(false)

  if (resourceSet && !initialized) {
    setDisplayName(resourceSet.name)
    setDescription(resourceSet.description)
    setInitialized(true)
  }

  if (initialized && resourceSet && resourceSet.id !== id) {
    setInitialized(false)
  }

  if (!id || (!isLoading && resourceSet === null)) {
    return (
      <div className="flex flex-col h-full">
        <div className="flex items-center justify-between px-4 py-3 shrink-0 border-b border-border bg-white">
          <PageHeader />
        </div>
        <div className="flex-1 flex items-center justify-center flex-col">
          <p className="text-sm text-gray-500">Resource Set not found.</p>
          <button onClick={() => navigate('/administration?tab=resource_sets')} className="mt-4 text-xs text-orange-500 hover:text-orange-600 font-medium">← Back to Administration</button>
        </div>
      </div>
    )
  }

  const disabled = !isEditing
  const selectedLabel = DETAIL_SECTIONS.find((s) => s.value === selectedSection)?.label ?? ''

  async function handleEditToggle() {
    if (isEditing) {
      setError(null)
      try {
        await saveMutation.mutateAsync({ name: displayName, description })
        setIsEditing(false)
      } catch (err) {
        setError(apiErrorMessage(err, 'Failed to save resource set'))
      }
    } else {
      setIsEditing(true)
    }
  }

  function handleCancel() {
    if (isEditing) {
      if (resourceSet) {
        setDisplayName(resourceSet.name)
        setDescription(resourceSet.description)
      }
      setIsEditing(false)
    } else {
      navigate('/administration?tab=resource_sets')
    }
  }

  async function handleDelete() {
    if (!id) return
    try {
      await deleteMutation.mutateAsync(id)
      navigate('/administration?tab=resource_sets')
    } catch (err) {
      setError(apiErrorMessage(err, 'Failed to delete resource set'))
    }
  }

  return (
    <div className="flex flex-col h-full">
      {error && (
        <div className="px-6 py-2 bg-red-50 border-b border-red-200 text-sm text-red-700 shrink-0">
          {error}
        </div>
      )}

      <div className="flex items-center justify-between px-4 shrink-0 h-[60px] border-b border-border bg-white">
        <PageHeader />
        <div className="flex items-center gap-3">
          <button type="button" onClick={handleEditToggle} className="px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white text-sm font-medium rounded-md transition-colors">{isEditing ? 'Save' : 'Edit'}</button>
          <button type="button" onClick={handleCancel} className="px-4 py-2 border border-gray-300 text-sm font-medium text-gray-600 hover:text-gray-900 hover:border-gray-400 rounded-md transition-colors">{isEditing ? 'Cancel' : 'Back'}</button>
          {!isEditing && (
            <button type="button" onClick={() => setShowDeleteConfirm(true)} className="px-4 py-2 border border-red-300 text-sm font-medium text-red-600 hover:bg-red-50 hover:border-red-400 rounded-md transition-colors">Delete</button>
          )}
          <div className="relative">
            <button type="button" onClick={() => setSectionOpen(!sectionOpen)} className="flex items-center gap-1.5 w-[160px] px-4 py-2 bg-[#28282B] hover:bg-[#333336] text-white text-sm font-medium rounded-md transition-colors">
              <FileText size={14} className="shrink-0" />
              <span className="truncate">{selectedLabel}</span>
              <svg className={`w-3.5 h-3.5 ml-auto shrink-0 transition-transform ${sectionOpen ? 'rotate-180' : ''}`} fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" /></svg>
            </button>
            {sectionOpen && (
              <div className="absolute top-full right-0 mt-1 bg-white border border-border rounded-md shadow-lg z-50 w-[180px] overflow-hidden">
                {DETAIL_SECTIONS.map((opt) => (
                  <button key={opt.value} type="button" onClick={() => { setSelectedSection(opt.value); setSectionOpen(false) }} className={`w-full text-left px-4 py-2.5 text-sm transition-colors ${selectedSection === opt.value ? 'bg-orange-50 text-orange-600 font-medium' : 'text-gray-600 hover:bg-gray-50'}`}>{opt.label}</button>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto bg-white">
        {selectedSection === 'details' && (
          <div className="flex h-full">
            <div className="w-[60%] px-6 py-6 overflow-y-auto">
              <div className={`space-y-5 ${disabled ? 'opacity-60 pointer-events-none' : ''}`}>
                <div className="border border-border rounded-lg p-5">
                  <h4 className="text-sm font-semibold text-gray-800 mb-4">Resource Set Details</h4>
                  <div className="space-y-4">
                    <FloatingInput label="Display Name" value={displayName} onChange={setDisplayName} disabled={disabled} required />
                    <FloatingTextarea label="Description (Optional)" value={description} onChange={setDescription} disabled={disabled} rows={3} />
                  </div>
                </div>
              </div>
            </div>

            <div className="w-[40%] px-6 py-6 overflow-y-auto">
              <div className="sticky top-0 space-y-5">
                {resourceSet && (
                  <div className="border border-border rounded-lg p-5">
                    <h4 className="text-sm font-semibold text-gray-800 mb-4">Metadata</h4>
                    <div className="space-y-4">
                      <div>
                        <label className="block text-[11px] text-gray-500 mb-1">ID</label>
                        <input type="text" readOnly value={resourceSet.id} className="w-full px-3 py-2 border border-border rounded-md text-sm text-gray-700 bg-gray-50" />
                      </div>
                      <div>
                        <label className="block text-[11px] text-gray-500 mb-1">Created At</label>
                        <input type="text" readOnly value={new Date(resourceSet.createdAt).toLocaleString()} className="w-full px-3 py-2 border border-border rounded-md text-sm text-gray-700 bg-gray-50" />
                      </div>
                      <div>
                        <label className="block text-[11px] text-gray-500 mb-1">Last Modified</label>
                        <input type="text" readOnly value={new Date(resourceSet.updatedAt).toLocaleString()} className="w-full px-3 py-2 border border-border rounded-md text-sm text-gray-700 bg-gray-50" />
                      </div>
                    </div>
                  </div>
                )}

                <div className="border border-border rounded-lg p-5">
                  <p className="text-xs text-gray-700 leading-relaxed mb-3">
                    <span className="font-semibold text-orange-600">Resource Sets</span> provide multi-tenancy within your organization. Each Resource Set is an isolated group of resources.
                  </p>
                  <p className="text-xs text-gray-700 leading-relaxed">
                    Assign <span className="font-semibold">Roles</span> to control which users can manage resources within this Resource Set.
                  </p>
                </div>
              </div>
            </div>
          </div>
        )}

        {selectedSection === 'metrics' && (
          <div className="flex items-center justify-center h-full"><p className="text-sm text-gray-400">Metrics — coming soon</p></div>
        )}
        {selectedSection === 'logs' && (
          <div className="flex items-center justify-center h-full"><p className="text-sm text-gray-400">Logs — coming soon</p></div>
        )}
      </div>

      {showDeleteConfirm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
          <div className="bg-white rounded-lg shadow-xl p-6 w-[400px]">
            <h3 className="text-sm font-semibold text-gray-900 mb-2">Delete Resource Set</h3>
            <p className="text-xs text-gray-600 mb-5">
              Are you sure you want to delete this resource set? All resources within it will become unassigned. This action cannot be undone.
            </p>
            <div className="flex items-center justify-end gap-3">
              <button onClick={() => setShowDeleteConfirm(false)} className="px-4 py-2 text-xs font-medium text-gray-600 border border-gray-300 rounded-md hover:bg-gray-50 transition-colors">Cancel</button>
              <button onClick={handleDelete} disabled={deleteMutation.isPending} className="px-4 py-2 text-xs font-medium text-white bg-red-600 hover:bg-red-700 rounded-md transition-colors disabled:opacity-50">
                {deleteMutation.isPending ? 'Deleting…' : 'Delete'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

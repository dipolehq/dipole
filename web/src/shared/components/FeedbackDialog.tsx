import { useState } from 'react'
import { apiClient } from '@/shared/lib/api-client'

interface FeedbackDialogProps {
  open: boolean
  onClose: () => void
}

export function FeedbackDialog({ open, onClose }: FeedbackDialogProps) {
  const [message, setMessage] = useState('')
  const [submitted, setSubmitted] = useState(false)
  const [imagePreview, setImagePreview] = useState<string | null>(null)

  function handleImageChange(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    if (!file) return
    const reader = new FileReader()
    reader.onload = () => setImagePreview(reader.result as string)
    reader.readAsDataURL(file)
  }

  function handleRemoveImage() {
    setImagePreview(null)
  }

  async function handleSubmit() {
    if (!message.trim()) return
    try {
      await apiClient.post('/feedback', {
        message: message.trim(),
        image: imagePreview ?? undefined,
      })
      setSubmitted(true)
    } catch {
      // Still show success to the user — feedback is best-effort
      setSubmitted(true)
    }
  }

  function handleClose() {
    onClose()
    // Reset state after close animation
    setTimeout(() => {
      setMessage('')
      setSubmitted(false)
      setImagePreview(null)
    }, 200)
  }

  if (!open) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/40" />
      <div className="relative bg-white rounded-lg shadow-xl border border-border w-full max-w-lg mx-4">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4">
          <h3 className="text-base font-semibold text-gray-900">Help us improve Dipole</h3>
          <button onClick={handleClose} className="text-gray-400 hover:text-gray-600 transition-colors">
            <span className="text-lg leading-none">×</span>
          </button>
        </div>

        {/* Body */}
        <div className="px-6 pb-5">
          {submitted ? (
            <div className="py-8 text-center">
              <p className="text-sm text-gray-700 font-medium">Thank you for your feedback!</p>
              <p className="text-xs text-gray-500 mt-1">We'll use it to make Dipole better.</p>
            </div>
          ) : (
            <>
              <p className="text-sm text-gray-600 mb-3">Let us know how we can improve Dipole. Your feedback helps us build a better product.</p>
              <textarea
                value={message}
                onChange={(e) => setMessage(e.target.value)}
                placeholder="What's working well? What's confusing? What's missing?"
                rows={5}
                className="w-full px-3 py-2.5 text-sm border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-gray-300 focus:border-gray-400 resize-none placeholder:text-gray-400"
              />

              {/* Image upload */}
              <div className="mt-3">
                {imagePreview ? (
                  <div className="relative inline-block">
                    <img src={imagePreview} alt="Screenshot" className="max-h-32 rounded-md border border-border" />
                    <button
                      type="button"
                      onClick={handleRemoveImage}
                      className="absolute -top-2 -right-2 w-5 h-5 bg-white border border-border rounded-full flex items-center justify-center text-gray-500 hover:text-gray-700 shadow-sm text-xs"
                    >
                      ×
                    </button>
                  </div>
                ) : (
                  <label className="inline-flex items-center gap-1.5 px-3 py-1.5 border border-dashed border-gray-300 rounded-md text-xs text-gray-500 hover:text-gray-700 hover:border-gray-400 cursor-pointer transition-colors">
                    <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" /></svg>
                    Attach screenshot
                    <input type="file" accept="image/*" onChange={handleImageChange} className="hidden" />
                  </label>
                )}
              </div>

              <p className="text-xs text-gray-400 mt-4">
                Have an issue? Email us at{' '}
                <a href="mailto:support@dipolehq.com" className="text-gray-600 hover:text-gray-700">support@dipolehq.com</a>.
              </p>
            </>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end gap-2 px-6 py-3 border-t border-border">
          <button type="button" onClick={handleClose} className="h-9 px-4 text-sm font-medium text-gray-700 border border-border rounded-md hover:bg-gray-50 transition-colors">
            {submitted ? 'Close' : 'Cancel'}
          </button>
          {!submitted && (
            <button type="button" onClick={handleSubmit} disabled={!message.trim()} className="h-9 px-4 text-sm font-medium text-white bg-gray-900 rounded-md hover:bg-gray-800 transition-colors disabled:opacity-40 disabled:cursor-not-allowed">
              Send
            </button>
          )}
        </div>
      </div>
    </div>
  )
}

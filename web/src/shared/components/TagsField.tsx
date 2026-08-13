import { useState } from 'react'
import { FloatingInput } from '@/shared/components/FloatingInput'
import type { FormTag } from '@/shared/components/forms/types'

/**
 * TagsField edits a list of key/value tags.
 *
 * Pass `value` + `onChange` to run controlled — this is REQUIRED for any form
 * that persists tags, so the parent owns the data and can submit it. If both
 * are omitted the field falls back to internal state (edits are NOT lifted to
 * any parent and will not be saved); that mode exists only for pages not yet
 * wired to a backend and must not be used on a page that persists tags.
 */
interface TagsFieldProps {
  value?: FormTag[]
  onChange?: (tags: FormTag[]) => void
  disabled?: boolean
}

export function TagsField({ value, onChange, disabled = false }: TagsFieldProps) {
  const [internal, setInternal] = useState<FormTag[]>([])
  const tags = value ?? internal
  const setTags = onChange ?? setInternal

  function addTag() {
    setTags([...tags, { key: '', value: '' }])
  }

  function updateTag(idx: number, field: 'key' | 'value', val: string) {
    setTags(tags.map((tag, i) => (i === idx ? { ...tag, [field]: val } : tag)))
  }

  function removeTag(idx: number) {
    setTags(tags.filter((_, i) => i !== idx))
  }

  return (
    <div>
      {tags.length > 0 && (
        <div className="space-y-2 mb-3">
          {tags.map((tag, idx) => (
            <div key={idx} className="flex items-center gap-2">
              <div className="flex-1">
                <FloatingInput
                  label="Key"
                  value={tag.key}
                  onChange={(v) => updateTag(idx, 'key', v)}
                  disabled={disabled}
                />
              </div>
              <div className="flex-1">
                <FloatingInput
                  label="Value"
                  value={tag.value}
                  onChange={(v) => updateTag(idx, 'value', v)}
                  disabled={disabled}
                />
              </div>
              <button
                type="button"
                onClick={() => removeTag(idx)}
                className="text-gray-400 hover:text-gray-600 text-sm"
              >
                ×
              </button>
            </div>
          ))}
        </div>
      )}
      <button
        type="button"
        onClick={addTag}
        className="text-xs text-gray-500 hover:text-gray-700 font-medium"
      >
        + New Tag
      </button>
    </div>
  )
}

import { useState } from 'react'
import { FloatingInput } from '@/shared/components/FloatingInput'

export function TagsField() {
  const [tags, setTags] = useState<{ key: string; value: string }[]>([])

  function addTag() {
    setTags([...tags, { key: '', value: '' }])
  }

  function updateTag(idx: number, field: 'key' | 'value', val: string) {
    setTags(tags.map((tag, i) => i === idx ? { ...tag, [field]: val } : tag))
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
                <FloatingInput label="Key" value={tag.key} onChange={(v) => updateTag(idx, 'key', v)} />
              </div>
              <div className="flex-1">
                <FloatingInput label="Value" value={tag.value} onChange={(v) => updateTag(idx, 'value', v)} />
              </div>
              <button onClick={() => removeTag(idx)} className="text-gray-400 hover:text-red-500 text-sm">×</button>
            </div>
          ))}
        </div>
      )}
      <button
        onClick={addTag}
        className="text-xs text-orange-500 hover:text-orange-600 font-medium"
      >
        + New Tag
      </button>
    </div>
  )
}

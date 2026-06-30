import { useState } from 'react'

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
              <input
                type="text"
                placeholder="Key"
                value={tag.key}
                onChange={(e) => updateTag(idx, 'key', e.target.value)}
                className="flex-1 px-3 py-2 border border-border rounded-md text-xs focus:outline-none focus:ring-1 focus:ring-gray-300"
              />
              <input
                type="text"
                placeholder="Value"
                value={tag.value}
                onChange={(e) => updateTag(idx, 'value', e.target.value)}
                className="flex-1 px-3 py-2 border border-border rounded-md text-xs focus:outline-none focus:ring-1 focus:ring-gray-300"
              />
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

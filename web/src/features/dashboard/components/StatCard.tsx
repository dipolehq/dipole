import type { LucideIcon } from 'lucide-react'

interface StatCardProps {
  title: string
  count: number
  icon: LucideIcon
  color: string
  lines: string[]
}

export function StatCard({ title, count, icon: Icon, color, lines }: StatCardProps) {
  return (
    <div className="bg-white rounded-lg border border-border p-4 flex flex-col">
      <div className="flex items-center gap-2 mb-2">
        <Icon size={16} className="text-gray-500" />
        <span className="text-sm font-medium text-gray-700">{title}</span>
      </div>
      <span className={`text-2xl font-bold ${color}`}>{count.toLocaleString()}</span>
      <div className="mt-1 space-y-0.5">
        {lines.map((line, idx) => (
          <p key={idx} className="text-xs text-gray-500">
            {line}
          </p>
        ))}
      </div>
    </div>
  )
}

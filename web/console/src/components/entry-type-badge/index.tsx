import {
  Bug,
  Ellipsis,
  LifeBuoy,
  Lightbulb,
  type LucideIcon,
  Sparkles,
  Tag,
} from 'lucide-react'

import type { ApiEntryType } from '@/services/api'

const ICONS: Record<string, LucideIcon> = {
  lightbulb: Lightbulb,
  bug: Bug,
  'life-buoy': LifeBuoy,
  ellipsis: Ellipsis,
}

export function EntryTypeIcon({
  name,
  color,
  className,
}: {
  name?: string
  color?: string
  className?: string
}) {
  const Icon = (name && ICONS[name]) || Tag
  return <Icon className={className ?? 'size-4'} style={color ? { color } : undefined} />
}

export default function EntryTypeBadge({
  entryType,
  appliedByAI,
}: {
  entryType: Pick<ApiEntryType, 'title' | 'icon' | 'color'>
  appliedByAI?: boolean
}) {
  return (
    <span
      className="inline-flex items-center gap-1 rounded px-1.5 py-0.5 text-[10px] uppercase tracking-wide text-white"
      style={{
        backgroundColor: entryType.color,
      }}
    >
      <EntryTypeIcon name={entryType.icon} color="#ffffff" className="size-3" />
      {entryType.title}
      {appliedByAI && (
        <Sparkles
          className="size-3 shrink-0"
          aria-label="Applied by AI"
        />
      )}
    </span>
  )
}

import {
  Bug,
  Ellipsis,
  LifeBuoy,
  Lightbulb,
  type LucideIcon,
  Tag,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import type { ApiEntryType } from '@/services/api'
import { entryTypeInfo } from '@/utils/entry-type'

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
}: {
  entryType: Pick<ApiEntryType, 'value' | 'title' | 'icon' | 'color'>
}) {
  const { t } = useTranslation()
  // Use the local palette titleKey when we recognize the value; fall
  // back to the wire `title` for forward-compat with future enum values.
  const palette = entryTypeInfo(entryType.value)
  const label = palette ? t(palette.titleKey) : entryType.title
  return (
    <span
      className="inline-flex items-center gap-1 whitespace-nowrap rounded px-1.5 py-0.5 text-[10px] uppercase tracking-wide text-white"
      style={{
        backgroundColor: entryType.color,
      }}
    >
      <EntryTypeIcon name={entryType.icon} color="#ffffff" className="size-3" />
      {label}
    </span>
  )
}

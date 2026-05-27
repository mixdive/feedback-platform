import {
  Check,
  CircleDashed,
  FlaskConical,
  Loader,
  type LucideIcon,
  Search,
  Sparkles,
  X,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import type { ApiEntryStatus } from '@/services/api'
import { entryStatusInfo } from '@/utils/entry-status'

const ICONS: Record<string, LucideIcon> = {
  sparkles: Sparkles,
  search: Search,
  loader: Loader,
  'flask-conical': FlaskConical,
  check: Check,
  x: X,
}

export function StatusIcon({
  name,
  color,
  className,
}: {
  name?: string
  color?: string
  className?: string
}) {
  const Icon = (name && ICONS[name]) || CircleDashed
  return <Icon className={className ?? 'size-4'} style={color ? { color } : undefined} />
}

export default function StatusBadge({
  status,
}: {
  status: Pick<ApiEntryStatus, 'value' | 'title' | 'icon' | 'color'>
}) {
  const { t } = useTranslation()
  // Resolve the localized label via the local palette's titleKey, but
  // fall back to the API-provided English title for any value the
  // palette doesn't know about (forward-compat with future statuses
  // that ship on the wire before the TS palette is updated).
  const palette = entryStatusInfo(status.value)
  const label = palette.value === status.value ? t(palette.titleKey) : status.title
  return (
    <span
      className="inline-flex items-center gap-1 rounded px-1.5 py-0.5 text-[10px] uppercase tracking-wide text-white"
      style={{
        backgroundColor: status.color,
      }}
    >
      <StatusIcon name={status.icon} color="#ffffff" className="size-3" />
      {label}
    </span>
  )
}

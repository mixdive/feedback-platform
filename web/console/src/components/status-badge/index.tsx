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

import type { ApiEntryStatus } from '@/services/api'

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
  status: Pick<ApiEntryStatus, 'title' | 'icon' | 'color'>
}) {
  return (
    <span
      className="inline-flex items-center gap-1 rounded px-1.5 py-0.5 text-[10px] uppercase tracking-wide text-white"
      style={{
        backgroundColor: status.color,
      }}
    >
      <StatusIcon name={status.icon} color="#ffffff" className="size-3" />
      {status.title}
    </span>
  )
}

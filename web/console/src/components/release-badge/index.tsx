import { CheckCircle2, Tag } from 'lucide-react'
import clsx from 'clsx'

import type { ApiRelease } from '@/services/api'

// ReleaseBadge is the chip rendered on entry list rows and the detail
// sidebar. The state-completed variant gets a filled green look so a
// shipped feature is distinguishable from a planned one at a glance.
export default function ReleaseBadge({
  release,
  size = 'md',
}: {
  release: Pick<ApiRelease, 'versionName' | 'state' | 'title'>
  size?: 'sm' | 'md'
}) {
  const completed = release.state === 'completed'
  const Icon = completed ? CheckCircle2 : Tag
  const label = release.versionName
  const tooltip = release.title
    ? `${release.versionName}${release.title ? ' — ' + release.title : ''}`
    : release.versionName

  return (
    <span
      title={tooltip}
      className={clsx(
        'inline-flex items-center gap-1 rounded-md border font-mono',
        size === 'sm' ? 'px-1.5 py-0.5 text-[10px]' : 'px-2 py-0.5 text-xs',
        completed
          ? 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-500/30 dark:bg-emerald-500/10 dark:text-emerald-300'
          : 'border-sky-200 bg-sky-50 text-sky-700 dark:border-sky-500/30 dark:bg-sky-500/10 dark:text-sky-300',
      )}
    >
      <Icon className={size === 'sm' ? 'size-2.5' : 'size-3'} />
      {label}
    </span>
  )
}

import clsx from 'clsx'

import { EntryTypeIcon } from '@/components/entry-type-badge'
import type { ApiEntryTypeCounts } from '@/services/api'
import { ENTRY_TYPES } from '@/utils/entry-type'

// EntryTypeCountChips renders one small chip per entry type (Feature
// Request / Bug / Support / Other) with the count of entries of that
// type within the parent group (topic or release). Zero-count chips
// are dimmed so non-zero buckets stand out without losing the full
// breakdown. Used on the Topics + Releases list pages on the right
// side of each row.
export default function EntryTypeCountChips({
  counts,
  className,
}: {
  counts: ApiEntryTypeCounts | undefined
  className?: string
}) {
  const buckets = (
    [
      { value: 'feature-request', count: counts?.featureRequest ?? 0 },
      { value: 'bug', count: counts?.bug ?? 0 },
      { value: 'support', count: counts?.support ?? 0 },
      { value: 'other', count: counts?.other ?? 0 },
    ] as const
  ).map((b) => {
    const meta = ENTRY_TYPES.find((t) => t.value === b.value)!
    return { ...b, meta }
  })

  return (
    <div className={clsx('flex shrink-0 items-center gap-1', className)}>
      {buckets.map((b) => {
        const isZero = b.count === 0
        return (
          <span
            key={b.value}
            title={`${b.meta.title}: ${b.count}`}
            className={clsx(
              'inline-flex items-center gap-1 rounded-md border px-1.5 py-0.5 text-[11px] tabular-nums transition-opacity',
              isZero
                ? 'border-zinc-200 dark:border-zinc-800 text-zinc-400 dark:text-zinc-600 opacity-60'
                : 'border-zinc-200 dark:border-zinc-700 bg-white dark:bg-zinc-950 text-zinc-700 dark:text-zinc-200',
            )}
            style={!isZero ? { borderColor: `${b.meta.color}55` } : undefined}
          >
            <EntryTypeIcon
              name={b.meta.icon}
              color={isZero ? '#9CA3AF' : b.meta.color}
              className="size-3"
            />
            {b.count}
          </span>
        )
      })}
    </div>
  )
}

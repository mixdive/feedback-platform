import { Lock } from 'lucide-react'
import clsx from 'clsx'

// InternalBadge marks entries the Portal cannot show. Filled amber pill
// + lock icon — visual weight is intentional so admins can scan a list
// and spot internal items at a glance.
export default function InternalBadge({
  size = 'sm',
  className,
}: {
  size?: 'sm' | 'md'
  className?: string
}) {
  return (
    <span
      title="Internal — hidden from the portal"
      className={clsx(
        'inline-flex items-center gap-1 rounded font-semibold uppercase tracking-wide',
        'border border-amber-300 bg-amber-100 text-amber-800',
        'dark:border-amber-500/40 dark:bg-amber-500/15 dark:text-amber-200',
        size === 'sm' ? 'px-1.5 py-0.5 text-[10px]' : 'px-2 py-0.5 text-xs',
        className,
      )}
    >
      <Lock className={size === 'sm' ? 'size-3' : 'size-3.5'} />
      Internal
    </span>
  )
}

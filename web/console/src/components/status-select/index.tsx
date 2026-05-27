import type { ApiEntryStatusValue } from '@/services/api'
import { ENTRY_STATUSES } from '@/utils/entry-status'

// Status is a hardcoded enum (see web/console/src/utils/entry-status.ts).
// The select reads from that list directly — no API call needed.
export default function StatusSelect({
  value,
  onChange,
  disabled,
  className,
}: {
  value: ApiEntryStatusValue
  onChange: (value: ApiEntryStatusValue) => void
  disabled?: boolean
  className?: string
}) {
  return (
    <select
      value={value}
      disabled={disabled}
      onChange={(e) => onChange(e.target.value as ApiEntryStatusValue)}
      className={
        'block w-full h-10 rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 text-sm focus:border-sky-500 focus:outline-none ' +
        (className ?? '')
      }
    >
      {ENTRY_STATUSES.map((s) => (
        <option key={s.value} value={s.value}>
          {s.title}
        </option>
      ))}
    </select>
  )
}

import type { ApiEntryTypeValue } from '@/services/api'
import { ENTRY_TYPES } from '@/utils/entry-type'

// Entry type is a hardcoded enum (see
// web/console/src/utils/entry-type.ts). The select reads from that
// list directly — no API call needed.
export default function EntryTypeSelect({
  value,
  onChange,
  disabled,
  allowEmpty,
  emptyLabel = 'No entry type',
  className,
}: {
  value: ApiEntryTypeValue | ''
  onChange: (value: ApiEntryTypeValue | '') => void
  disabled?: boolean
  allowEmpty?: boolean
  emptyLabel?: string
  className?: string
}) {
  return (
    <select
      value={value}
      disabled={disabled}
      onChange={(e) => onChange(e.target.value as ApiEntryTypeValue | '')}
      className={
        'block w-full h-10 rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 text-sm focus:border-sky-500 focus:outline-none ' +
        (className ?? '')
      }
    >
      {allowEmpty && <option value="">{emptyLabel}</option>}
      {ENTRY_TYPES.map((t) => (
        <option key={t.value} value={t.value}>
          {t.title}
        </option>
      ))}
    </select>
  )
}

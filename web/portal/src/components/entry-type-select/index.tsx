import { useTranslation } from 'react-i18next'

import { EntryTypeIcon } from '@/components/entry-type-badge'
import type { ApiEntryTypeValue } from '@/services/api'
import { ENTRY_TYPES, entryTypeInfo } from '@/utils/entry-type'

// Entry type is a hardcoded enum (see
// web/portal/src/utils/entry-type.ts). The select reads from that
// list directly — no API call needed.
export default function EntryTypeSelect({
  value,
  onChange,
  disabled,
  allowEmpty,
  emptyLabel,
  className,
}: {
  value: ApiEntryTypeValue | ''
  onChange: (value: ApiEntryTypeValue | '') => void
  disabled?: boolean
  allowEmpty?: boolean
  emptyLabel?: string
  className?: string
}) {
  const { t } = useTranslation()
  const selected = entryTypeInfo(value || undefined)
  const resolvedEmptyLabel = emptyLabel ?? t('entryTypeSelect.noEntryType')
  return (
    <div className={'flex items-center gap-2 ' + (className ?? '')}>
      {selected && (
        <EntryTypeIcon
          name={selected.icon}
          color={selected.color}
          className="size-4 shrink-0"
        />
      )}
      <select
        value={value}
        disabled={disabled}
        onChange={(e) => onChange(e.target.value as ApiEntryTypeValue | '')}
        className="block w-full h-10 rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 text-sm focus:border-sky-500 focus:outline-none"
      >
        {allowEmpty && <option value="">{resolvedEmptyLabel}</option>}
        {ENTRY_TYPES.map((tt) => (
          <option key={tt.value} value={tt.value}>
            {t(tt.titleKey)}
          </option>
        ))}
      </select>
    </div>
  )
}

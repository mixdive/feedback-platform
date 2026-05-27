import type { ApiRelease } from '@/services/api'

// ReleaseSelect is the single-select picker for assigning ONE release
// to an entry. Native <select> to match Status/Category. onChange
// receives the new releaseId or '' when the user clears.
export default function ReleaseSelect({
  releases,
  value,
  onChange,
  disabled,
  emptyLabel = 'No release',
  className,
}: {
  releases: ApiRelease[] | undefined
  value: string
  onChange: (id: string) => void
  disabled?: boolean
  emptyLabel?: string
  className?: string
}) {
  return (
    <select
      value={value}
      disabled={disabled || !releases}
      onChange={(e) => onChange(e.target.value)}
      className={
        'block w-full h-10 rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 text-sm focus:border-sky-500 focus:outline-none ' +
        (className ?? '')
      }
    >
      <option value="">{emptyLabel}</option>
      {(releases ?? []).map((r) => {
        const suffix = r.title ? ` — ${r.title}` : ''
        const state = r.state ? ` (${r.state})` : ''
        return (
          <option key={r.id} value={r.id}>
            {r.versionName}
            {suffix}
            {state}
          </option>
        )
      })}
    </select>
  )
}

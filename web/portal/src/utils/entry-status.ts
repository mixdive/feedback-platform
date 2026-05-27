// Hardcoded entry status palette. Mirrors the Go enum in
// models/entry.go and the display metadata baked into api/shared.go's
// entryStatusDisplay map. Update all three in lockstep when adding or
// renaming a status.
//
// `titleKey` is the i18n key consumed by StatusBadge (and any other
// caller that needs the localized label). The API still ships an
// English `title` on the wire so non-React consumers (Console / future
// channels) keep working; Portal components prefer the local key.

import type { ApiEntryStatusValue } from '@/services/api'

export type EntryStatusInfo = {
  value: ApiEntryStatusValue
  titleKey: string
  color: string
  icon: string
  closed: boolean
}

export const ENTRY_STATUSES: EntryStatusInfo[] = [
  { value: 'new', titleKey: 'entryStatus.new', color: '#6366F1', icon: 'sparkles', closed: false },
  { value: 'evaluation', titleKey: 'entryStatus.evaluation', color: '#0EA5E9', icon: 'search', closed: false },
  { value: 'in-progress', titleKey: 'entryStatus.in-progress', color: '#F59E0B', icon: 'loader', closed: false },
  { value: 'completed', titleKey: 'entryStatus.completed', color: '#10B981', icon: 'check', closed: true },
  { value: 'cancelled', titleKey: 'entryStatus.cancelled', color: '#E11D48', icon: 'x', closed: true },
]

const BY_VALUE: Record<string, EntryStatusInfo> = Object.fromEntries(
  ENTRY_STATUSES.map((s) => [s.value, s]),
)

export function entryStatusInfo(value: string | undefined): EntryStatusInfo {
  if (value && BY_VALUE[value]) return BY_VALUE[value]
  return ENTRY_STATUSES[0]
}

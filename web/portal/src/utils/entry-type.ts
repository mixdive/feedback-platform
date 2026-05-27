// Hardcoded entry-type palette. Mirrors the Go enum in
// models/entry.go and the display metadata baked into api/shared.go's
// entryTypeDisplay map. Update all three in lockstep when adding or
// renaming a value.
//
// `titleKey` is the i18n key consumed by EntryTypeBadge. The API still
// ships the English `title` so existing wire consumers keep working;
// Portal components prefer the local key.

import type { ApiEntryTypeValue } from '@/services/api'

export type EntryTypeInfo = {
  value: ApiEntryTypeValue
  titleKey: string
  color: string
  icon: string
}

export const ENTRY_TYPES: EntryTypeInfo[] = [
  { value: 'feature-request', titleKey: 'entryType.feature-request', color: '#10B981', icon: 'lightbulb' },
  { value: 'bug', titleKey: 'entryType.bug', color: '#E11D48', icon: 'bug' },
  { value: 'support', titleKey: 'entryType.support', color: '#8B5CF6', icon: 'life-buoy' },
  { value: 'other', titleKey: 'entryType.other', color: '#64748B', icon: 'ellipsis' },
]

const BY_VALUE: Record<string, EntryTypeInfo> = Object.fromEntries(
  ENTRY_TYPES.map((t) => [t.value, t]),
)

export function entryTypeInfo(value: string | undefined): EntryTypeInfo | undefined {
  if (!value) return undefined
  return BY_VALUE[value]
}

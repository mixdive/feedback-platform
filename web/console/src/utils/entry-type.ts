// Hardcoded entry-type palette. Mirrors the Go enum in
// models/entry.go and the display metadata baked into api/shared.go's
// entryTypeDisplay map. Update all three in lockstep when adding or
// renaming a value.

import type { ApiEntryTypeValue } from '@/services/api'

export type EntryTypeInfo = {
  value: ApiEntryTypeValue
  title: string
  color: string
  icon: string
}

export const ENTRY_TYPES: EntryTypeInfo[] = [
  { value: 'feature-request', title: 'Feature Request', color: '#10B981', icon: 'lightbulb' },
  { value: 'bug', title: 'Bug', color: '#E11D48', icon: 'bug' },
  { value: 'support', title: 'Support', color: '#8B5CF6', icon: 'life-buoy' },
  { value: 'other', title: 'Other', color: '#64748B', icon: 'ellipsis' },
]

const BY_VALUE: Record<string, EntryTypeInfo> = Object.fromEntries(
  ENTRY_TYPES.map((t) => [t.value, t]),
)

export function entryTypeInfo(value: string | undefined): EntryTypeInfo | undefined {
  if (!value) return undefined
  return BY_VALUE[value]
}
